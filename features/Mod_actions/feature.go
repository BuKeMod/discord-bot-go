package mod_actions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
)

type handler struct{ feature.Dependencies }

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}

	banPerm := int64(discordgo.PermissionBanMembers)
	kickPerm := int64(discordgo.PermissionKickMembers)
	modPerm := int64(discordgo.PermissionModerateMembers)
	purgePerm := int64(discordgo.PermissionManageMessages)

	zero := float64(0)
	seven := float64(7)
	one := float64(1)
	hundred := float64(100)
	maxTimeout := float64(40320) // 28 days

	return feature.Module{
		Name: "Mod_actions",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "ban",
					Description:              "แบนสมาชิกออกจากเซิร์ฟเวอร์",
					DefaultMemberPermissions: &banPerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการแบน",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการแบน",
							Required:    false,
							MaxLength:   500,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "delete_days",
							Description: "จำนวนวันของข้อความที่ต้องการลบ (0–7 วัน)",
							Required:    false,
							MinValue:    &zero,
							MaxValue:    seven,
						},
					},
				},
				Execute: h.ban,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "unban",
					Description:              "ปลดแบนผู้ใช้งานด้วย User ID",
					DefaultMemberPermissions: &banPerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "user_id",
							Description: "Discord User ID ที่ต้องการปลดแบน",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการปลดแบน",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
				Execute: h.unban,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "kick",
					Description:              "เตะสมาชิกออกจากเซิร์ฟเวอร์",
					DefaultMemberPermissions: &kickPerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการเตะ",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการเตะ",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
				Execute: h.kick,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "timeout",
					Description:              "ระงับการส่งข้อความ/เสียงของสมาชิกชั่วคราว (Timeout)",
					DefaultMemberPermissions: &modPerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการ Timeout",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "minutes",
							Description: "ระยะเวลาที่ระงับ (นาที, สูงสุด 28 วัน = 40,320 นาที)",
							Required:    true,
							MinValue:    &one,
							MaxValue:    maxTimeout,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการ Timeout",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
				Execute: h.timeout,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "untimeout",
					Description:              "ยกเลิกการ Timeout สมาชิก",
					DefaultMemberPermissions: &modPerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการยกเลิก Timeout",
							Required:    true,
						},
					},
				},
				Execute: h.untimeout,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "warn",
					Description:              "ระบบจัดการการตักเตือนสมาชิก (Add / List / Remove)",
					DefaultMemberPermissions: &modPerm,
					Options: []*discordgo.ApplicationCommandOption{
						feature.Action("add", "list", "remove"),
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการจัดการการตักเตือน (ใช้กับ add หรือ list)",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการตักเตือน (ใช้กับ add)",
							Required:    false,
							MaxLength:   500,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "id",
							Description: "รหัสการตักเตือน (ใช้กับ remove)",
							Required:    false,
						},
					},
				},
				Execute: h.warn,
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "purge",
					Description:              "ลบข้อความจำนวนมากในห้องนี้ (สำหรับผู้ดูแล/แอดมิน)",
					DefaultMemberPermissions: &purgePerm,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "จำนวนข้อความที่ต้องการลบ (1–100 ข้อความ)",
							Required:    true,
							MinValue:    &one,
							MaxValue:    hundred,
						},
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "ลบเฉพาะข้อความของสมาชิกที่ระบุ (ไม่บังคับ)",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "contains",
							Description: "ลบเฉพาะข้อความที่มีคำนี้ (ไม่บังคับ)",
							Required:    false,
							MaxLength:   100,
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "only_bots",
							Description: "ลบเฉพาะข้อความของบอท (ไม่บังคับ)",
							Required:    false,
						},
					},
				},
				Execute: h.purge,
			},
		},
	}
}

func (h *handler) ban(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.UserID(i)
	if targetID == "" || targetID == i.Member.User.ID || targetID == h.BotID {
		return "ไม่สามารถแบนผู้ใช้นี้ได้", nil
	}

	reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")
	deleteDays := int(featureutil.Integer(i, "delete_days", 0))

	auditReason := fmt.Sprintf("โดย %s: %s", i.Member.User.Username, reason)
	err := s.GuildBanCreateWithReason(i.GuildID, targetID, auditReason, deleteDays, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	h.sendModEmbed(s, i.GuildID, 0xED4245, "🔨 แบนสมาชิก (Ban)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ถูกแบน", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})

	return fmt.Sprintf("แบน <@%s> สำเร็จ (เหตุผล: %s)", targetID, reason), nil
}

func (h *handler) unban(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.String(i, "user_id", "")
	if targetID == "" {
		return "กรุณาระบุ User ID ของผู้ที่ต้องการปลดแบน", nil
	}

	reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")
	auditReason := fmt.Sprintf("โดย %s: %s", i.Member.User.Username, reason)
	err := s.GuildBanDelete(i.GuildID, targetID, discordgo.WithAuditLogReason(auditReason), discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	h.sendModEmbed(s, i.GuildID, 0x57F287, "🔓 ปลดแบนสมาชิก (Unban)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ปลดแบน", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})

	return fmt.Sprintf("ปลดแบนผู้ใช้งาน <@%s> สำเร็จ", targetID), nil
}

func (h *handler) kick(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.UserID(i)
	if targetID == "" || targetID == i.Member.User.ID || targetID == h.BotID {
		return "ไม่สามารถเตะผู้ใช้นี้ได้", nil
	}

	reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")
	auditReason := fmt.Sprintf("โดย %s: %s", i.Member.User.Username, reason)
	err := s.GuildMemberDeleteWithReason(i.GuildID, targetID, auditReason, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	h.sendModEmbed(s, i.GuildID, 0xE67E22, "👢 เตะสมาชิก (Kick)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ถูกเตะ", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})

	return fmt.Sprintf("เตะ <@%s> สำเร็จ (เหตุผล: %s)", targetID, reason), nil
}

func (h *handler) timeout(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.UserID(i)
	if targetID == "" || targetID == i.Member.User.ID || targetID == h.BotID {
		return "ไม่สามารถ Timeout ผู้ใช้นี้ได้", nil
	}

	minutes := featureutil.Integer(i, "minutes", 5)
	reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")

	until := time.Now().Add(time.Duration(minutes) * time.Minute)
	auditReason := fmt.Sprintf("โดย %s (%d นาที): %s", i.Member.User.Username, minutes, reason)
	err := s.GuildMemberTimeout(i.GuildID, targetID, &until, discordgo.WithAuditLogReason(auditReason), discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	h.sendModEmbed(s, i.GuildID, 0xF1C40F, "⏳ ระงับการใช้งาน (Timeout)", []*discordgo.MessageEmbedField{
		{Name: "สมาชิกที่ถูก Timeout", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
		{Name: "ระยะเวลา", Value: fmt.Sprintf("%d นาที (สิ้นสุด <t:%d:R>)", minutes, until.Unix()), Inline: false},
		{Name: "เหตุผล", Value: reason, Inline: false},
	})

	return fmt.Sprintf("Timeout <@%s> เป็นเวลา %d นาที สำเร็จ (สิ้นสุด <t:%d:R>)", targetID, minutes, until.Unix()), nil
}

func (h *handler) untimeout(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.UserID(i)
	if targetID == "" {
		return "กรุณาระบุสมาชิกที่ต้องการยกเลิก Timeout", nil
	}

	err := s.GuildMemberTimeout(i.GuildID, targetID, nil, discordgo.WithAuditLogReason(fmt.Sprintf("ยกเลิก Timeout โดย %s", i.Member.User.Username)), discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	h.sendModEmbed(s, i.GuildID, 0x57F287, "⌛ ยกเลิก Timeout", []*discordgo.MessageEmbedField{
		{Name: "สมาชิก", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
	})

	return fmt.Sprintf("ยกเลิก Timeout สำหรับ <@%s> สำเร็จ", targetID), nil
}

func (h *handler) warn(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	action := featureutil.String(i, "action", "list")

	switch action {
	case "add":
		targetID := featureutil.UserID(i)
		if targetID == "" || targetID == i.Member.User.ID || targetID == h.BotID {
			return "ไม่สามารถตักเตือนผู้ใช้นี้ได้", nil
		}
		reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")

		randomBytes := make([]byte, 3)
		_, _ = rand.Read(randomBytes)
		warnID := "w_" + hex.EncodeToString(randomBytes)

		now := time.Now()
		if err := h.Store.AddWarn(ctx, warnID, i.GuildID, targetID, i.Member.User.ID, reason, now.Unix()); err != nil {
			return "", err
		}

		h.sendModEmbed(s, i.GuildID, 0xE74C3C, "⚠️ ตักเตือนสมาชิก (Warn)", []*discordgo.MessageEmbedField{
			{Name: "สมาชิกที่ถูกเตือน", Value: fmt.Sprintf("<@%s> (`%s`)", targetID, targetID), Inline: true},
			{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
			{Name: "รหัสการเตือน", Value: fmt.Sprintf("`%s`", warnID), Inline: true},
			{Name: "เหตุผล", Value: reason, Inline: false},
		})

		return fmt.Sprintf("ตักเตือน <@%s> สำเร็จ (รหัสการเตือน: `%s`, เหตุผล: %s)", targetID, warnID, reason), nil

	case "list":
		targetID := featureutil.UserID(i)
		if targetID == "" {
			return "กรุณาระบุสมาชิกที่ต้องการดูประวัติการตักเตือน", nil
		}

		warns, err := h.Store.ListWarns(ctx, i.GuildID, targetID)
		if err != nil {
			return "", err
		}

		if len(warns) == 0 {
			return fmt.Sprintf("<@%s> ไม่เคยมีประวัติการตักเตือนในเซิร์ฟเวอร์นี้", targetID), nil
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("⚠️ **ประวัติการตักเตือนของ** <@%s> (ทั้งหมด %d ครั้ง):", targetID, len(warns)))
		for idx, w := range warns {
			lines = append(lines, fmt.Sprintf("%d. รหัส: `%s` | ผู้เตือน: <@%s> | เวลา: <t:%d:R>\n   └ เหตุผล: %s",
				idx+1, w.ID, w.Moderator, w.CreatedAt, w.Reason))
		}
		return strings.Join(lines, "\n"), nil

	case "remove":
		warnID := featureutil.String(i, "id", "")
		if warnID == "" {
			return "กรุณาระบุรหัสการเตือน (เช่น `w_xxxxxx`) ที่ต้องการลบ", nil
		}

		deleted, err := h.Store.RemoveWarn(ctx, i.GuildID, warnID)
		if err != nil {
			return "", err
		}
		if !deleted {
			return fmt.Sprintf("ไม่พบรายการตักเตือนรหัส `%s` ในเซิร์ฟเวอร์", warnID), nil
		}

		return fmt.Sprintf("ลบรายการตักเตือนรหัส `%s` เรียบร้อยแล้ว", warnID), nil

	default:
		return "Action ไม่ถูกต้อง", nil
	}
}

func (h *handler) purge(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	channel, err := s.Channel(i.ChannelID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if channel.GuildID != i.GuildID || channel.Type != discordgo.ChannelTypeGuildText {
		return "คำสั่งนี้สามารถใช้ได้เฉพาะในห้องข้อความปกติของเซิร์ฟเวอร์เท่านั้น", nil
	}

	neededPerms := int64(discordgo.PermissionViewChannel | discordgo.PermissionReadMessageHistory | discordgo.PermissionManageMessages)
	if i.AppPermissions != 0 {
		if i.AppPermissions&discordgo.PermissionAdministrator == 0 && (i.AppPermissions&neededPerms) != neededPerms {
			return "บอทต้องการสิทธิ์ View Channel, Read Message History และ Manage Messages ในห้องนี้เพื่อลบข้อความ", nil
		}
	} else {
		hasPerms, err := featureutil.BotHasChannelPermissions(ctx, s, i.GuildID, h.BotID, channel, neededPerms)
		if err != nil || !hasPerms {
			return "บอทต้องการสิทธิ์ View Channel, Read Message History และ Manage Messages ในห้องนี้เพื่อลบข้อความ", nil
		}
	}

	amount := int(featureutil.Integer(i, "amount", 1))
	if amount < 1 {
		amount = 1
	}
	if amount > 100 {
		amount = 100
	}

	targetUser := featureutil.String(i, "user", "")
	contains := strings.ToLower(featureutil.String(i, "contains", ""))
	onlyBots := featureutil.Boolean(i, "only_bots")

	fetchLimit := amount
	if targetUser != "" || contains != "" || onlyBots {
		fetchLimit = 100
	}

	messages, err := s.ChannelMessages(i.ChannelID, fetchLimit, "", "", "", discordgo.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("ไม่สามารถอ่านข้อความในห้องนี้ได้: %w", err)
	}

	twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
	var toDelete []string
	var skippedOldCount int

	for _, msg := range messages {
		if len(toDelete) >= amount {
			break
		}
		if msg.Timestamp.Before(twoWeeksAgo) {
			skippedOldCount++
			continue
		}

		if targetUser != "" && (msg.Author == nil || msg.Author.ID != targetUser) {
			continue
		}
		if onlyBots && (msg.Author == nil || !msg.Author.Bot) {
			continue
		}
		if contains != "" && !strings.Contains(strings.ToLower(msg.Content), contains) {
			continue
		}

		toDelete = append(toDelete, msg.ID)
	}

	if len(toDelete) == 0 {
		if skippedOldCount > 0 {
			return "ไม่พบข้อความที่ตรงตามเงื่อนไข หรือข้อความมีอายุเกิน 14 วัน (Discord ไม่อนุญาตให้ลบข้อความที่เก่าเกิน 14 วัน)", nil
		}
		return "ไม่พบข้อความที่ตรงตามเงื่อนไขที่ระบุ", nil
	}

	if len(toDelete) == 1 {
		err = s.ChannelMessageDelete(i.ChannelID, toDelete[0], discordgo.WithContext(ctx))
	} else {
		err = s.ChannelMessagesBulkDelete(i.ChannelID, toDelete, discordgo.WithContext(ctx))
	}

	if err != nil {
		return "", fmt.Errorf("เกิดข้อผิดพลาดในการลบข้อความ: %w", err)
	}

	filterNotes := []string{}
	if targetUser != "" {
		filterNotes = append(filterNotes, fmt.Sprintf("เฉพาะสมาชิก <@%s>", targetUser))
	}
	if contains != "" {
		filterNotes = append(filterNotes, fmt.Sprintf("คำว่า `%s`", contains))
	}
	if onlyBots {
		filterNotes = append(filterNotes, "เฉพาะข้อความของบอท")
	}

	filterStr := "ทั้งหมด (ไม่มีตัวกรอง)"
	if len(filterNotes) > 0 {
		filterStr = strings.Join(filterNotes, ", ")
	}

	h.sendModEmbed(s, i.GuildID, 0xE67E22, "🧹 ลบข้อความจำนวนมาก (Purge)", []*discordgo.MessageEmbedField{
		{Name: "ห้อง", Value: fmt.Sprintf("<#%s>", i.ChannelID), Inline: true},
		{Name: "ผู้ดำเนินการ", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
		{Name: "จำนวนที่ลบ", Value: fmt.Sprintf("**%d** ข้อความ", len(toDelete)), Inline: true},
		{Name: "ตัวกรอง", Value: filterStr, Inline: false},
	})

	note := ""
	if skippedOldCount > 0 {
		note = fmt.Sprintf("\n*(ข้ามข้อความที่เก่าเกิน 14 วันไป %d ข้อความตามข้อจำกัดของ Discord)*", skippedOldCount)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🧹 ลบข้อความเรียบร้อย (Purge)",
		Description: fmt.Sprintf("ลบข้อความในห้อง <#%s> จำนวน **%d** ข้อความเรียบร้อยแล้ว%s", i.ChannelID, len(toDelete), note),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ห้อง", Value: fmt.Sprintf("<#%s>", i.ChannelID), Inline: true},
			{Name: "จำนวนที่ลบ", Value: fmt.Sprintf("%d ข้อความ", len(toDelete)), Inline: true},
			{Name: "ตัวกรอง", Value: filterStr, Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	feature.SetEmbed(ctx, embed)

	return "", nil
}

func (h *handler) sendModEmbed(s *discordgo.Session, guildID string, color int, title string, fields []*discordgo.MessageEmbedField) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	channelID, err := h.Store.LogChannel(ctx, guildID, "mod")
	if err != nil || channelID == "" {
		channelID, _ = h.Store.LogChannel(ctx, guildID, "all")
	}
	if channelID == "" {
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

