package voice_log

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

type session struct {
	channelID string
	joinedAt  time.Time
}

type module struct {
	feature.Dependencies
	mu       sync.Mutex
	sessions map[string]session // userID -> session
}

func New(d feature.Dependencies) feature.Module {
	m := &module{
		Dependencies: d,
		sessions:     make(map[string]session),
	}

	return feature.Module{
		Name:             "Voice_log",
		Intents:          discordgo.IntentsGuildVoiceStates,
		VoiceStateUpdate: m.onVoiceStateUpdate,
	}
}

func (m *module) onVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	if e == nil || e.VoiceState == nil || m.Store == nil {
		return
	}
	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}
	if !m.Store.IsModuleEnabled(context.Background(), guildID, "logs") {
		return
	}
	userID := e.UserID
	newChannelID := e.ChannelID

	m.mu.Lock()
	old, hadOld := m.sessions[userID]
	if !hadOld && e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID != "" {
		old = session{channelID: e.BeforeUpdate.ChannelID, joinedAt: time.Now()}
		hadOld = true
	}

	var actionType string
	var oldChannelID string
	var duration time.Duration

	now := time.Now()
	if !hadOld && newChannelID != "" {
		actionType = "join"
		m.sessions[userID] = session{channelID: newChannelID, joinedAt: now}
	} else if hadOld && newChannelID == "" {
		actionType = "leave"
		oldChannelID = old.channelID
		duration = now.Sub(old.joinedAt).Round(time.Second)
		delete(m.sessions, userID)
	} else if hadOld && newChannelID != "" && newChannelID != old.channelID {
		actionType = "move"
		oldChannelID = old.channelID
		m.sessions[userID] = session{channelID: newChannelID, joinedAt: now}
	} else {
		// Mute / Deafen or other voice state update, no channel transition
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logChannelID, err := m.Store.LogChannel(ctx, guildID, "voice")
	if err != nil || logChannelID == "" {
		return
	}

	var embed *discordgo.MessageEmbed
	timestamp := fmt.Sprintf("<t:%d:F> (<t:%d:R>)", now.Unix(), now.Unix())

	switch actionType {
	case "join":
		embed = &discordgo.MessageEmbed{
			Title: "📥 สมาชิกเข้าห้องเสียง",
			Color: 0x57F287, // Green
			Fields: []*discordgo.MessageEmbedField{
				{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", userID, userID), Inline: true},
				{Name: "ห้องเสียง", Value: fmt.Sprintf("<#%s>", newChannelID), Inline: true},
				{Name: "เวลา", Value: timestamp, Inline: false},
			},
			Timestamp: now.Format(time.RFC3339),
		}
	case "leave":
		durText := formatDuration(duration)
		fields := []*discordgo.MessageEmbedField{
			{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", userID, userID), Inline: true},
			{Name: "เวลา", Value: timestamp, Inline: false},
		}
		if oldChannelID != "" {
			fields = append([]*discordgo.MessageEmbedField{
				fields[0],
				{Name: "ห้องเดิม", Value: fmt.Sprintf("<#%s>", oldChannelID), Inline: true},
				{Name: "ระยะเวลาที่อยู่ในห้อง", Value: durText, Inline: true},
			}, fields[1:]...)
		}
		embed = &discordgo.MessageEmbed{
			Title:     "📤 สมาชิกออกจากห้องเสียง",
			Color:     0xED4245, // Red
			Fields:    fields,
			Timestamp: now.Format(time.RFC3339),
		}
	case "move":
		embed = &discordgo.MessageEmbed{
			Title: "🔄 สมาชิกย้ายห้องเสียง",
			Color: 0xFEE75C, // Yellow
			Fields: []*discordgo.MessageEmbedField{
				{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", userID, userID), Inline: true},
				{Name: "จากห้อง", Value: fmt.Sprintf("<#%s>", oldChannelID), Inline: true},
				{Name: "ไปยังห้อง", Value: fmt.Sprintf("<#%s>", newChannelID), Inline: true},
				{Name: "เวลา", Value: timestamp, Inline: false},
			},
			Timestamp: now.Format(time.RFC3339),
		}
	}

	if embed != nil {
		_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "ไม่กี่วินาที"
	}
	seconds := int64(d.Seconds())
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	res := ""
	if days > 0 {
		res += fmt.Sprintf("%d วัน ", days)
	}
	if hours > 0 || days > 0 {
		res += fmt.Sprintf("%d ชั่วโมง ", hours)
	}
	if minutes > 0 || hours > 0 || days > 0 {
		res += fmt.Sprintf("%d นาที ", minutes)
	}
	res += fmt.Sprintf("%d วินาที", secs)
	return res
}
