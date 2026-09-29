package logs_mod

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

type timeoutState struct {
	until *time.Time
}

type module struct {
	feature.Dependencies
	mu       sync.Mutex
	timeouts map[string]timeoutState // userID -> state
}

func New(d feature.Dependencies) feature.Module {
	m := &module{
		Dependencies: d,
		timeouts:     make(map[string]timeoutState),
	}

	return feature.Module{
		Name:           "Logs_mod",
		Intents:        discordgo.IntentsGuildBans | discordgo.IntentsGuildMembers,
		GuildBanAdd:    m.onBanAdd,
		GuildBanRemove: m.onBanRemove,
		MemberRemove:   m.onMemberRemove,
		MemberUpdate:   m.onMemberUpdate,
	}
}

func (m *module) onBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	if e == nil || e.User == nil || m.Store == nil {
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

	time.Sleep(500 * time.Millisecond) // brief wait for audit log propagation

	moderator := "ไม่ทราบ"
	reason := "ไม่ระบุเหตุผล"

	auditLogs, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMemberBanAdd), 2)
	if err == nil && auditLogs != nil {
		for _, entry := range auditLogs.AuditLogEntries {
			if entry.TargetID == e.User.ID {
				moderator = fmt.Sprintf("<@%s> (`%s`)", entry.UserID, entry.UserID)
				if entry.Reason != "" {
					reason = entry.Reason
				}
				break
			}
		}
	}

	m.sendModEmbed(s, guildID, 0xED4245, "🔨 สมาชิกถูกแบน (Ban Detected)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ถูกแบน", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: moderator, Inline: true},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})
}

func (m *module) onBanRemove(s *discordgo.Session, e *discordgo.GuildBanRemove) {
	if e == nil || e.User == nil || m.Store == nil {
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

	time.Sleep(500 * time.Millisecond)

	moderator := "ไม่ทราบ"
	reason := "ไม่ระบุเหตุผล"

	auditLogs, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMemberBanRemove), 2)
	if err == nil && auditLogs != nil {
		for _, entry := range auditLogs.AuditLogEntries {
			if entry.TargetID == e.User.ID {
				moderator = fmt.Sprintf("<@%s> (`%s`)", entry.UserID, entry.UserID)
				if entry.Reason != "" {
					reason = entry.Reason
				}
				break
			}
		}
	}

	m.sendModEmbed(s, guildID, 0x57F287, "🔓 สมาชิกถูกปลดแบน (Unban Detected)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ปลดแบน", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: moderator, Inline: true},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})
}

func (m *module) onMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	if e == nil || e.Member == nil || e.User == nil || m.Store == nil {
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

	time.Sleep(500 * time.Millisecond)

	// Check if this was a kick action
	auditLogs, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMemberKick), 2)
	if err != nil || auditLogs == nil {
		return
	}

	for _, entry := range auditLogs.AuditLogEntries {
		if entry.TargetID == e.User.ID {
			moderator := fmt.Sprintf("<@%s> (`%s`)", entry.UserID, entry.UserID)
			reason := entry.Reason
			if reason == "" {
				reason = "ไม่ระบุเหตุผล"
			}

			m.sendModEmbed(s, guildID, 0xE67E22, "👢 สมาชิกถูกเตะออก (Kick Detected)", []*discordgo.MessageEmbedField{
				{Name: "สมาชิกที่ถูกเตะ", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
				{Name: "ผู้ดำเนินการ", Value: moderator, Inline: true},
				{Name: "เหตุผล", Value: reason, Inline: false},
			})
			break
		}
	}
}

func (m *module) onMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	if e == nil || e.Member == nil || e.User == nil || m.Store == nil {
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

	m.mu.Lock()
	old, hadOld := m.timeouts[e.User.ID]
	m.timeouts[e.User.ID] = timeoutState{until: e.CommunicationDisabledUntil}
	m.mu.Unlock()

	now := time.Now()

	// 1. Newly timed out
	if e.CommunicationDisabledUntil != nil && e.CommunicationDisabledUntil.After(now) {
		if !hadOld || old.until == nil || !old.until.Equal(*e.CommunicationDisabledUntil) {
			dur := e.CommunicationDisabledUntil.Sub(now).Round(time.Minute)
			moderator := "ไม่ทราบ"
			reason := "ไม่ระบุเหตุผล"

			time.Sleep(500 * time.Millisecond)
			auditLogs, err := s.GuildAuditLog(guildID, "", "", int(discordgo.AuditLogActionMemberUpdate), 2)
			if err == nil && auditLogs != nil {
				for _, entry := range auditLogs.AuditLogEntries {
					if entry.TargetID == e.User.ID {
						moderator = fmt.Sprintf("<@%s> (`%s`)", entry.UserID, entry.UserID)
						if entry.Reason != "" {
							reason = entry.Reason
						}
						break
					}
				}
			}

			m.sendModEmbed(s, guildID, 0xF1C40F, "⏳ สมาชิกถูกระงับการใช้งาน (Timeout Detected)", []*discordgo.MessageEmbedField{
				{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
				{Name: "ผู้ดำเนินการ", Value: moderator, Inline: true},
				{Name: "สิ้นสุดเมื่อ", Value: fmt.Sprintf("<t:%d:F> (<t:%d:R>, ประมาณ %d นาที)", e.CommunicationDisabledUntil.Unix(), e.CommunicationDisabledUntil.Unix(), int(dur.Minutes())), Inline: false},
				{Name: "เหตุผล", Value: reason, Inline: false},
			})
		}
	} else if hadOld && old.until != nil && old.until.After(now) && (e.CommunicationDisabledUntil == nil || !e.CommunicationDisabledUntil.After(now)) {
		// 2. Untimed out early
		m.sendModEmbed(s, guildID, 0x57F287, "⌛ สมาชิกพ้นการระงับการใช้งาน (Untimeout Detected)", []*discordgo.MessageEmbedField{
			{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", e.User.ID, e.User.ID), Inline: true},
		})
	}
}

func (m *module) sendModEmbed(s *discordgo.Session, guildID string, color int, title string, fields []*discordgo.MessageEmbedField) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if m.Store != nil && !m.Store.IsModuleEnabled(ctx, guildID, "logs") {
		return
	}

	channelID, err := m.Store.LogChannel(ctx, guildID, "mod")
	if err != nil || channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:     title,
		Color:     color,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}
