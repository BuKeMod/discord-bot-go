package utility_ticket

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
)

type handler struct{ feature.Dependencies }

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}
	adminPerm := int64(discordgo.PermissionAdministrator)

	return feature.Module{
		Name: "Utility_ticket",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "ticket",
					Description:              "ระบบ Ticket สำหรับเปิดห้องช่วยเหลือสมาชิก",
					DefaultMemberPermissions: &adminPerm,
					Options: []*discordgo.ApplicationCommandOption{
						feature.Action("panel", "close"),
						{
							Type:         discordgo.ApplicationCommandOptionChannel,
							Name:         "channel",
							Description:  "ห้องที่ต้องการส่ง Panel สร้าง Ticket (ใช้กับ panel)",
							Required:     false,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "title",
							Description: "หัวข้อบน Ticket Panel",
							Required:    false,
							MaxLength:   150,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "description",
							Description: "คำอธิบายบน Ticket Panel",
							Required:    false,
							MaxLength:   1000,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "button_text",
							Description: "ข้อความบนปุ่มสร้าง Ticket",
							Required:    false,
							MaxLength:   80,
						},
						{
							Type:         discordgo.ApplicationCommandOptionChannel,
							Name:         "category",
							Description:  "หมวดหมู่ (Category) ที่จะสร้างห้อง Ticket",
							Required:     false,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory},
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "support_role",
							Description: "Role ของทีมงานที่สามารถมองเห็นห้อง Ticket",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "reason",
							Description: "เหตุผลในการปิด Ticket (ใช้กับ close)",
							Required:    false,
							MaxLength:   300,
						},
					},
				},
				Execute: h.executeCommand,
			},
		},
		ComponentHandler: h.handleComponent,
	}
}

func (h *handler) executeCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	action := featureutil.String(i, "action", "panel")

	switch action {
	case "panel":
		chID := featureutil.String(i, "channel", i.ChannelID)
		title := featureutil.String(i, "title", "📩 ศูนย์บริการช่วยเหลือ (Support Tickets)")
		desc := featureutil.String(i, "description", "หากคุณมีคำถาม พบปัญหา หรือต้องการความช่วยเหลือจากทีมงาน\nกรุณากดปุ่ม **\"สร้าง Ticket\"** ด้านล่างนี้เพื่อเปิดห้องสนทนาส่วนตัวครับ")
		btnText := featureutil.String(i, "button_text", "📩 สร้าง Ticket (Create Ticket)")
		category := featureutil.String(i, "category", "")
		supportRole := featureutil.String(i, "support_role", "")

		if err := h.Store.SetTicketConfig(ctx, i.GuildID, category, supportRole); err != nil {
			return "", err
		}

		embed := &discordgo.MessageEmbed{
			Title:       title,
			Description: desc,
			Color:       0x5865F2, // Blurple
			Footer: &discordgo.MessageEmbedFooter{
				Text: "กดปุ่มด้านล่างเพื่อเปิดห้องสนทนาส่วนตัวกับทีมงาน",
			},
		}

		button := discordgo.Button{
			Label:    btnText,
			Style:    discordgo.PrimaryButton,
			CustomID: "ticket_create",
			Emoji: &discordgo.ComponentEmoji{
				Name: "📩",
			},
		}

		_, err := s.ChannelMessageSendComplex(chID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{button},
				},
			},
		})
		if err != nil {
			return "", fmt.Errorf("ไม่สามารถส่ง Ticket Panel ในห้องนั้นได้: %w", err)
		}

		return fmt.Sprintf("ส่ง Ticket Panel ในห้อง <#%s> สำเร็จเรียบร้อย", chID), nil

	case "close":
		reason := featureutil.String(i, "reason", "ไม่ระบุเหตุผล")
		return h.startTicketClose(ctx, s, i.GuildID, i.ChannelID, i.Member.User.ID, reason)

	default:
		return "Action ไม่ถูกต้อง", nil
	}
}

func (h *handler) handleComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	customID := i.MessageComponentData().CustomID

	switch customID {
	case "ticket_create":
		h.createTicket(ctx, s, i)
		return true, nil

	case "ticket_close":
		// Show confirmation buttons
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "🔒 ยืนยันการปิด Ticket",
						Description: "คุณแน่ใจหรือไม่ว่าต้องการปิดและบันทึกประวัติ (Transcript) ของ Ticket นี้?\nห้องจะถูกลบหลังจากบันทึกประวัติเรียบร้อยแล้ว",
						Color:       0xED4245,
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "ยืนยันปิด Ticket",
								Style:    discordgo.DangerButton,
								CustomID: "ticket_close_confirm",
								Emoji:    &discordgo.ComponentEmoji{Name: "🔒"},
							},
							discordgo.Button{
								Label:    "ยกเลิก",
								Style:    discordgo.SecondaryButton,
								CustomID: "ticket_close_cancel",
							},
						},
					},
				},
			},
		})
		return true, nil

	case "ticket_close_cancel":
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "↩️ ยกเลิกการปิด Ticket",
						Description: "ยกเลิกการปิด Ticket แล้ว คุณสามารถสนทนาต่อได้ตามปกติครับ",
						Color:       0x5865F2,
					},
				},
				Components: []discordgo.MessageComponent{},
			},
		})
		return true, nil

	case "ticket_close_confirm":
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "⏳ กำลังบันทึกประวัติและปิดห้อง Ticket",
						Description: "ระบบกำลังรวบรวมข้อความ บันทึกไฟล์ Transcript และเตรียมปิดห้องนี้...",
						Color:       0xFEE75C,
					},
				},
				Components: []discordgo.MessageComponent{},
			},
		})
		go h.finishTicketClose(s, i.GuildID, i.ChannelID, i.Member.User.ID, "ปิดโดยผู้ใช้")
		return true, nil
	}

	return false, nil
}

func (h *handler) createTicket(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	userID := i.Member.User.ID

	// Check if user already has an open ticket
	existingCh, err := h.Store.GetActiveTicket(ctx, guildID, userID)
	if err == nil && existingCh != "" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "⚠️ พบห้อง Ticket เดิมของคุณ",
						Description: fmt.Sprintf("คุณมีห้อง Ticket ที่เปิดอยู่แล้วที่ <#%s>\nกรุณาใช้งานห้องเดิมก่อนครับ", existingCh),
						Color:       0xFEE75C,
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cfg, _ := h.Store.GetTicketConfig(ctx, guildID)
	ticketNum, err := h.Store.NextTicketNumber(ctx, guildID)
	if err != nil {
		ticketNum = int(time.Now().Unix() % 10000)
	}

	channelName := fmt.Sprintf("ticket-%04d", ticketNum)

	// Overwrites
	overwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   guildID, // @everyone
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		{
			ID:    userID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionAttachFiles | discordgo.PermissionEmbedLinks | discordgo.PermissionReadMessageHistory,
		},
		{
			ID:    h.BotID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionAllChannel,
		},
	}

	if cfg.SupportRole != "" {
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID:    cfg.SupportRole,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionAttachFiles | discordgo.PermissionEmbedLinks | discordgo.PermissionReadMessageHistory,
		})
	}

	data := discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             cfg.Category,
		PermissionOverwrites: overwrites,
	}

	ch, err := s.GuildChannelCreateComplex(guildID, data, discordgo.WithContext(ctx))
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "❌ เกิดข้อผิดพลาด",
						Description: fmt.Sprintf("ไม่สามารถสร้างห้อง Ticket ได้: %v", err),
						Color:       0xED4245,
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = h.Store.CreateTicket(ctx, ch.ID, guildID, userID, ticketNum, time.Now().Unix())

	// Send initial message in ticket channel
	mentionText := fmt.Sprintf("<@%s>", userID)
	if cfg.SupportRole != "" {
		mentionText += fmt.Sprintf(" <@&%s>", cfg.SupportRole)
	}

	ticketEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎫 Ticket #%04d", ticketNum),
		Description: fmt.Sprintf("สวัสดี <@%s> ขอบคุณที่ติดต่อเรา กรุณาแจ้งรายละเอียดหรือคำถามที่คุณต้องการความช่วยเหลือ ทีมงานจะเข้ามาตอบกลับโดยเร็วที่สุดครับ", userID),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ผู้เปิด Ticket", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
			{Name: "เวลาที่เปิด", Value: fmt.Sprintf("<t:%d:F>", time.Now().Unix()), Inline: true},
		},
	}

	closeBtn := discordgo.Button{
		Label:    "🔒 ปิด Ticket",
		Style:    discordgo.DangerButton,
		CustomID: "ticket_close",
	}

	_, _ = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content: mentionText,
		Embeds:  []*discordgo.MessageEmbed{ticketEmbed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{closeBtn},
			},
		},
	})

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "✅ สร้าง Ticket สำเร็จ",
					Description: fmt.Sprintf("สร้างห้อง Ticket ของคุณเรียบร้อยแล้ว: <#%s>", ch.ID),
					Color:       0x57F287,
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *handler) startTicketClose(ctx context.Context, s *discordgo.Session, guildID, channelID, closerID, reason string) (string, error) {
	ticket, err := h.Store.GetTicket(ctx, channelID)
	if err != nil || ticket == nil {
		return "คำสั่งนี้ใช้ได้เฉพาะภายในห้อง Ticket เท่านั้น", nil
	}

	go h.finishTicketClose(s, guildID, channelID, closerID, reason)
	return "กำลังสร้าง Transcript และปิดห้อง Ticket ใน 5 วินาที...", nil
}

func (h *handler) finishTicketClose(s *discordgo.Session, guildID, channelID, closerID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticket, _ := h.Store.GetTicket(ctx, channelID)
	ticketNum := 0
	ownerID := closerID
	if ticket != nil {
		ticketNum = ticket.TicketNum
		ownerID = ticket.OwnerID
	}

	// Fetch recent messages
	messages, err := s.ChannelMessages(channelID, 100, "", "", "")
	if err != nil {
		messages = []*discordgo.Message{}
	}

	// Generate HTML transcript
	htmlContent := generateHTMLTranscript(guildID, channelID, ticketNum, ownerID, closerID, reason, messages)
	filename := fmt.Sprintf("transcript-ticket-%04d.html", ticketNum)

	// Send to Log channel
	logChannelID, _ := h.Store.LogChannel(ctx, guildID, "ticket")
	if logChannelID == "" {
		logChannelID, _ = h.Store.LogChannel(ctx, guildID, "mod")
	}
	if logChannelID == "" {
		logChannelID, _ = h.Store.LogChannel(ctx, guildID, "all")
	}

	logEmbed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📁 บันทึกประวัติ Ticket #%04d (Transcript)", ticketNum),
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ห้อง", Value: fmt.Sprintf("#ticket-%04d", ticketNum), Inline: true},
			{Name: "ผู้เปิด", Value: fmt.Sprintf("<@%s>", ownerID), Inline: true},
			{Name: "ผู้ปิด", Value: fmt.Sprintf("<@%s>", closerID), Inline: true},
			{Name: "เหตุผล", Value: reason, Inline: false},
			{Name: "จำนวนข้อความ", Value: fmt.Sprintf("%d ข้อความ", len(messages)), Inline: true},
			{Name: "เวลาที่ปิด", Value: fmt.Sprintf("<t:%d:F>", time.Now().Unix()), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if logChannelID != "" {
		_, _ = s.ChannelMessageSendComplex(logChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{logEmbed},
			Files: []*discordgo.File{
				{
					Name:   filename,
					Reader: bytes.NewReader([]byte(htmlContent)),
				},
			},
		})
	}

	// Try sending DM to ticket owner
	dmChannel, err := s.UserChannelCreate(ownerID)
	if err == nil && dmChannel != nil {
		_, _ = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
			Content: fmt.Sprintf("Ticket #%04d ของคุณในเซิร์ฟเวอร์ถูกปิดเรียบร้อยแล้ว แนบไฟล์ประวัติการสนทนามาให้ด้วยครับ", ticketNum),
			Embeds:  []*discordgo.MessageEmbed{logEmbed},
			Files: []*discordgo.File{
				{
					Name:   filename,
					Reader: bytes.NewReader([]byte(htmlContent)),
				},
			},
		})
	}

	_ = h.Store.CloseTicket(ctx, channelID)

	// Send message before deleting
	_, _ = s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
		Title:       "🔒 ปิด Ticket สำเร็จแล้ว",
		Description: "บันทึกประวัติ (Transcript) เรียบร้อยแล้ว ห้องนี้จะถูกลบใน 5 วินาที...",
		Color:       0xED4245,
	})

	time.Sleep(5 * time.Second)
	_, _ = s.ChannelDelete(channelID)
}

func generateHTMLTranscript(guildID, channelID string, ticketNum int, ownerID, closerID, reason string, messages []*discordgo.Message) string {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Ticket #` + fmt.Sprintf("%04d", ticketNum) + ` Transcript</title>
<style>
body { margin: 0; padding: 20px; background-color: #313338; color: #dbdee1; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; }
.header { background-color: #2b2d31; border-radius: 8px; padding: 20px; margin-bottom: 20px; border-left: 4px solid #5865f2; }
.header h1 { margin: 0 0 10px 0; color: #ffffff; font-size: 24px; }
.header p { margin: 4px 0; font-size: 14px; color: #949ba4; }
.message-list { display: flex; flex-direction: column; gap: 12px; }
.message { display: flex; gap: 16px; padding: 8px 12px; border-radius: 6px; }
.message:hover { background-color: #2e3035; }
.avatar { width: 42px; height: 42px; border-radius: 50%; background-color: #5865f2; flex-shrink: 0; }
.msg-body { flex-grow: 1; }
.msg-header { margin-bottom: 4px; }
.username { font-weight: 600; color: #ffffff; font-size: 15px; margin-right: 8px; }
.timestamp { font-size: 12px; color: #949ba4; }
.content { font-size: 14px; line-height: 1.4; color: #dbdee1; word-break: break-word; }
.attachment { margin-top: 6px; font-size: 13px; }
.attachment a { color: #00a8fc; text-decoration: none; }
.attachment a:hover { text-decoration: underline; }
</style>
</head>
<body>
<div class="header">
  <h1>🎫 Ticket #` + fmt.Sprintf("%04d", ticketNum) + ` Transcript</h1>
  <p><strong>ผู้เปิด Ticket:</strong> User ID: ` + ownerID + `</p>
  <p><strong>ผู้ปิด Ticket:</strong> User ID: ` + closerID + `</p>
  <p><strong>เหตุผล:</strong> ` + html.EscapeString(reason) + `</p>
  <p><strong>จำนวนข้อความ:</strong> ` + fmt.Sprintf("%d", len(messages)) + ` ข้อความ</p>
  <p><strong>บันทึกเมื่อ:</strong> ` + time.Now().Format("2006-01-02 15:04:05 MST") + `</p>
</div>
<div class="message-list">
`)

	// Render messages in chronological order (reverse of fetched)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil {
			continue
		}
		authorName := "Unknown"
		avatarURL := ""
		if msg.Author != nil {
			authorName = msg.Author.Username
			if msg.Author.Discriminator != "" && msg.Author.Discriminator != "0" {
				authorName += "#" + msg.Author.Discriminator
			}
			avatarURL = msg.Author.AvatarURL("64")
		}

		timeStr := msg.Timestamp.Format("02/01/2006 15:04:05")

		buf.WriteString(`<div class="message">`)
		if avatarURL != "" {
			buf.WriteString(`<img class="avatar" src="` + html.EscapeString(avatarURL) + `" alt="Avatar">`)
		} else {
			buf.WriteString(`<div class="avatar"></div>`)
		}
		buf.WriteString(`<div class="msg-body">`)
		buf.WriteString(`<div class="msg-header">`)
		buf.WriteString(`<span class="username">` + html.EscapeString(authorName) + `</span>`)
		buf.WriteString(`<span class="timestamp">` + timeStr + `</span>`)
		buf.WriteString(`</div>`)

		contentHTML := html.EscapeString(msg.Content)
		contentHTML = strings.ReplaceAll(contentHTML, "\n", "<br>")
		buf.WriteString(`<div class="content">` + contentHTML + `</div>`)

		for _, att := range msg.Attachments {
			buf.WriteString(`<div class="attachment">📎 ไฟล์แนบ: <a href="` + html.EscapeString(att.URL) + `" target="_blank">` + html.EscapeString(att.Filename) + `</a></div>`)
		}

		buf.WriteString(`</div></div>`)
	}

	buf.WriteString(`
</div>
</body>
</html>`)

	return buf.String()
}
