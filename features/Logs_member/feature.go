package logs_member

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

type memberSnapshot struct {
	nick  string
	roles []string
}

type module struct {
	feature.Dependencies
	mu        sync.Mutex
	snapshots map[string]memberSnapshot // userID -> snapshot
}

func New(d feature.Dependencies) feature.Module {
	m := &module{
		Dependencies: d,
		snapshots:    make(map[string]memberSnapshot),
	}

	return feature.Module{
		Name:         "Logs_member",
		Intents:      discordgo.IntentsGuildMembers,
		MemberAdd:    m.onMemberAdd,
		MemberRemove: m.onMemberRemove,
		MemberUpdate: m.onMemberUpdate,
	}
}

func (m *module) onMemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	if e == nil || e.Member == nil || e.User == nil || m.Store == nil {
		return
	}

	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}

	m.mu.Lock()
	m.snapshots[e.User.ID] = memberSnapshot{nick: e.Nick, roles: append([]string(nil), e.Roles...)}
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "member")
	if err != nil || logChannelID == "" {
		return
	}

	createdAt := snowflakeTimestamp(e.User.ID)
	createdText := "ไม่ทราบ"
	if !createdAt.IsZero() {
		createdText = fmt.Sprintf("<t:%d:F> (<t:%d:R>)", createdAt.Unix(), createdAt.Unix())
	}

	now := time.Now()
	authorTag := e.User.Username
	if e.User.Discriminator != "" && e.User.Discriminator != "0" {
		authorTag += "#" + e.User.Discriminator
	}

	embed := &discordgo.MessageEmbed{
		Title: "📥 สมาชิกเข้าร่วมเซิร์ฟเวอร์",
		Color: 0x57F287, // Green
		Author: &discordgo.MessageEmbedAuthor{
			Name:    authorTag,
			IconURL: e.User.AvatarURL("128"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
			{Name: "สร้างบัญชีเมื่อ", Value: createdText, Inline: false},
			{Name: "เวลาที่เข้าร่วม", Value: fmt.Sprintf("<t:%d:F> (<t:%d:R>)", now.Unix(), now.Unix()), Inline: false},
		},
		Timestamp: now.Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
}

func (m *module) onMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	if e == nil || e.Member == nil || e.User == nil || m.Store == nil {
		return
	}

	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}

	m.mu.Lock()
	snap, hadSnap := m.snapshots[e.User.ID]
	delete(m.snapshots, e.User.ID)
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "member")
	if err != nil || logChannelID == "" {
		return
	}

	now := time.Now()
	authorTag := e.User.Username
	if e.User.Discriminator != "" && e.User.Discriminator != "0" {
		authorTag += "#" + e.User.Discriminator
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
		{Name: "เวลาที่ออก", Value: fmt.Sprintf("<t:%d:F> (<t:%d:R>)", now.Unix(), now.Unix()), Inline: false},
	}

	roles := e.Roles
	if len(roles) == 0 && hadSnap {
		roles = snap.roles
	}
	if len(roles) > 0 {
		var roleMentions []string
		for _, r := range roles {
			roleMentions = append(roleMentions, fmt.Sprintf("<@&%s>", r))
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Roles ที่เคยมี",
			Value:  strings.Join(roleMentions, " "),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title: "📤 สมาชิกออกจากเซิร์ฟเวอร์",
		Color: 0xED4245, // Red
		Author: &discordgo.MessageEmbedAuthor{
			Name:    authorTag,
			IconURL: e.User.AvatarURL("128"),
		},
		Fields:    fields,
		Timestamp: now.Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
}

func (m *module) onMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	if e == nil || e.Member == nil || e.User == nil || m.Store == nil {
		return
	}

	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}

	m.mu.Lock()
	old, hadOld := m.snapshots[e.User.ID]
	m.snapshots[e.User.ID] = memberSnapshot{nick: e.Nick, roles: append([]string(nil), e.Roles...)}
	m.mu.Unlock()

	if !hadOld {
		// First time seeing this member in memory, no diff available
		return
	}

	var changes []string

	// Check Nickname change
	if old.nick != e.Nick {
		oldNick := old.nick
		newNick := e.Nick
		if oldNick == "" {
			oldNick = "*(ไม่มี)*"
		}
		if newNick == "" {
			newNick = "*(ล้างชื่อเล่น)*"
		}
		changes = append(changes, fmt.Sprintf("• **ชื่อเล่น**: `%s` ➔ `%s`", oldNick, newNick))
	}

	// Check Roles change
	oldSet := make(map[string]bool)
	for _, r := range old.roles {
		oldSet[r] = true
	}
	newSet := make(map[string]bool)
	for _, r := range e.Roles {
		newSet[r] = true
	}

	var addedRoles, removedRoles []string
	for r := range newSet {
		if !oldSet[r] {
			addedRoles = append(addedRoles, fmt.Sprintf("<@&%s>", r))
		}
	}
	for r := range oldSet {
		if !newSet[r] {
			removedRoles = append(removedRoles, fmt.Sprintf("<@&%s>", r))
		}
	}

	if len(addedRoles) > 0 {
		changes = append(changes, fmt.Sprintf("• **เพิ่ม Role**: %s", strings.Join(addedRoles, " ")))
	}
	if len(removedRoles) > 0 {
		changes = append(changes, fmt.Sprintf("• **ถอด Role**: %s", strings.Join(removedRoles, " ")))
	}

	if len(changes) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "member")
	if err != nil || logChannelID == "" {
		return
	}

	now := time.Now()
	authorTag := e.User.Username
	if e.User.Discriminator != "" && e.User.Discriminator != "0" {
		authorTag += "#" + e.User.Discriminator
	}

	embed := &discordgo.MessageEmbed{
		Title: "👤 อัปเดตข้อมูลสมาชิก",
		Color: 0x9B59B6, // Purple
		Author: &discordgo.MessageEmbedAuthor{
			Name:    authorTag,
			IconURL: e.User.AvatarURL("128"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
			{Name: "รายละเอียดการเปลี่ยนแปลง", Value: strings.Join(changes, "\n"), Inline: false},
		},
		Timestamp: now.Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
}

func snowflakeTimestamp(idStr string) time.Time {
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return time.Time{}
	}
	milli := int64((id >> 22) + 1420070400000)
	return time.UnixMilli(milli)
}
