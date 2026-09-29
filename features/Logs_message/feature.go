package logs_message

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

type cachedMessage struct {
	id           string
	channelID    string
	guildID      string
	authorID     string
	authorTag    string
	authorAvatar string
	content      string
	attachments  []string
	createdAt    time.Time
}

type messageCache struct {
	mu       sync.RWMutex
	capacity int
	entries  map[string]cachedMessage
	order    []string
}

func newCache(cap int) *messageCache {
	return &messageCache{
		capacity: cap,
		entries:  make(map[string]cachedMessage),
		order:    make([]string, 0, cap),
	}
}

func (c *messageCache) Set(m cachedMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[m.id]; !exists {
		if len(c.order) >= c.capacity {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.entries, oldest)
		}
		c.order = append(c.order, m.id)
	}
	c.entries[m.id] = m
}

func (c *messageCache) Get(id string) (cachedMessage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.entries[id]
	return m, ok
}

func (c *messageCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
}

type module struct {
	feature.Dependencies
	cache *messageCache
}

func New(d feature.Dependencies) feature.Module {
	m := &module{
		Dependencies: d,
		cache:        newCache(1000),
	}

	return feature.Module{
		Name:          "Logs_message",
		Intents:       discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent,
		MessageCreate: m.onMessageCreate,
		MessageUpdate: m.onMessageUpdate,
		MessageDelete: m.onMessageDelete,
	}
}

func (m *module) onMessageCreate(s *discordgo.Session, e *discordgo.MessageCreate) {
	if e == nil || e.Message == nil || e.Author == nil || e.Author.Bot {
		return
	}

	var attachNames []string
	for _, a := range e.Attachments {
		if a.Filename != "" {
			attachNames = append(attachNames, a.Filename)
		}
	}

	authorTag := e.Author.Username
	if e.Author.Discriminator != "" && e.Author.Discriminator != "0" {
		authorTag += "#" + e.Author.Discriminator
	}

	m.cache.Set(cachedMessage{
		id:           e.ID,
		channelID:    e.ChannelID,
		guildID:      e.GuildID,
		authorID:     e.Author.ID,
		authorTag:    authorTag,
		authorAvatar: e.Author.AvatarURL("128"),
		content:      e.Content,
		attachments:  attachNames,
		createdAt:    time.Now(),
	})
}

func (m *module) onMessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	if e == nil || e.Message == nil || m.Store == nil {
		return
	}
	if e.Author != nil && e.Author.Bot {
		return
	}

	cached, found := m.cache.Get(e.ID)
	if !found {
		return
	}

	// Only log if content actually changed
	if e.Content == "" || e.Content == cached.content {
		return
	}

	guildID := e.GuildID
	if guildID == "" {
		guildID = cached.guildID
	}
	if guildID == "" {
		guildID = m.Config.GuildID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "message")
	if err != nil || logChannelID == "" {
		return
	}

	oldContent := truncate(cached.content, 1000)
	newContent := truncate(e.Content, 1000)
	if oldContent == "" {
		oldContent = "*(ไม่มีเนื้อหา)*"
	}
	if newContent == "" {
		newContent = "*(ไม่มีเนื้อหา)*"
	}

	now := time.Now()
	jumpURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, cached.channelID, e.ID)

	embed := &discordgo.MessageEmbed{
		Title: "✏️ ข้อความถูกแก้ไข",
		Color: 0x3498DB, // Blue
		Author: &discordgo.MessageEmbedAuthor{
			Name:    cached.authorTag,
			IconURL: cached.authorAvatar,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ผู้ส่ง", Value: fmt.Sprintf("<@%s> (`%s`)", cached.authorID, cached.authorID), Inline: true},
			{Name: "ห้อง", Value: fmt.Sprintf("<#%s>", cached.channelID), Inline: true},
			{Name: "ลิงก์ข้อความ", Value: fmt.Sprintf("[กระโดดไปยังข้อความ](%s)", jumpURL), Inline: false},
			{Name: "ก่อนแก้ไข", Value: oldContent, Inline: false},
			{Name: "หลังแก้ไข", Value: newContent, Inline: false},
		},
		Timestamp: now.Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)

	// Update cached message with new content
	cached.content = e.Content
	m.cache.Set(cached)
}

func (m *module) onMessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	if e == nil || e.Message == nil || m.Store == nil {
		return
	}

	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "message")
	if err != nil || logChannelID == "" {
		return
	}

	cached, found := m.cache.Get(e.ID)
	now := time.Now()

	var embed *discordgo.MessageEmbed

	if found {
		m.cache.Delete(e.ID)

		// Check audit log for deleter
		deleter := "ผู้ส่งข้อความเอง / ไม่ทราบ"
		auditLogs, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMessageDelete), 2)
		if err == nil && auditLogs != nil {
			for _, entry := range auditLogs.AuditLogEntries {
				if entry.TargetID == cached.authorID {
					deleter = fmt.Sprintf("<@%s> (`%s`)", entry.UserID, entry.UserID)
					break
				}
			}
		}

		fields := []*discordgo.MessageEmbedField{
			{Name: "ผู้ส่ง", Value: fmt.Sprintf("<@%s> (`%s`)", cached.authorID, cached.authorID), Inline: true},
			{Name: "ห้อง", Value: fmt.Sprintf("<#%s>", cached.channelID), Inline: true},
			{Name: "ผู้ลบ", Value: deleter, Inline: true},
		}

		content := truncate(cached.content, 1000)
		if content == "" {
			content = "*(ไม่มีเนื้อหาข้อความ)*"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "ข้อความที่ถูกลบ",
			Value:  content,
			Inline: false,
		})

		if len(cached.attachments) > 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "ไฟล์แนบ",
				Value:  strings.Join(cached.attachments, ", "),
				Inline: false,
			})
		}

		embed = &discordgo.MessageEmbed{
			Title: "🗑️ ข้อความถูกลบ",
			Color: 0xED4245, // Red
			Author: &discordgo.MessageEmbedAuthor{
				Name:    cached.authorTag,
				IconURL: cached.authorAvatar,
			},
			Fields:    fields,
			Timestamp: now.Format(time.RFC3339),
		}
	} else {
		// Not found in cache
		embed = &discordgo.MessageEmbed{
			Title: "🗑️ ข้อความถูกลบ (ไม่อยู่ในแคช)",
			Color: 0xED4245, // Red
			Fields: []*discordgo.MessageEmbedField{
				{Name: "รหัสข้อความ (ID)", Value: e.ID, Inline: true},
				{Name: "ห้อง", Value: fmt.Sprintf("<#%s>", e.ChannelID), Inline: true},
				{Name: "รายละเอียด", Value: "*(ข้อความนี้ถูกส่งก่อนบอทเริ่มทำงาน หรือเกินขีดจำกัดแคช)*", Inline: false},
			},
			Timestamp: now.Format(time.RFC3339),
		}
	}

	if embed != nil {
		_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
	}
}

func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) > max {
		return string(runes[:max]) + "... (ถูกตัดทอน)"
	}
	return text
}
