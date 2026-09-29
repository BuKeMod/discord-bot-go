package utility_party

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
)

type handler struct {
	feature.Dependencies
	repo   *repository
	worker *worker
}

func New(d feature.Dependencies) feature.Module {
	h := &handler{
		Dependencies: d,
	}

	two := float64(2)
	fifty := float64(50)
	five := float64(5)
	dayMinutes := float64(1440)

	return feature.Module{
		Name: "Utility_party",
		Start: func(ctx context.Context, s *discordgo.Session) error {
			if h.Store == nil || h.Store.SQL() == nil {
				return fmt.Errorf("Utility_party requires SQLite")
			}
			h.repo = &repository{db: h.Store.SQL()}
			h.worker = &worker{repo: h.repo}
			if err := h.repo.migrate(ctx); err != nil {
				return err
			}
			return h.worker.start(ctx, s)
		},
		Stop: func() {
			if h.worker != nil {
				h.worker.stop()
			}
		},
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "party",
					Description: "สร้างการ์ดนัดเล่นเกม / ตั้งตี้พร้อมปุ่มกดเข้าร่วม (Interactive LFG)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "game",
							Description: "ชื่อเกม เช่น Valorant, Apex, RoV, Dead by Daylight",
							Required:    true,
							MaxLength:   80,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "max_players",
							Description: "จำนวนผู้เล่นตัวจริงที่ต้องการ (ค่าเริ่มต้น 5 คน)",
							Required:    false,
							MinValue:    &two,
							MaxValue:    fifty,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "time",
							Description: "เวลานัดเล่น เช่น 21:00, ใน 30 นาที, ตอนนี้เลย",
							Required:    false,
							MaxLength:   60,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "note",
							Description: "หมายเหตุ เช่น ลงแรงค์ Diamond+, เล่นขำๆ ไม่ตึง",
							Required:    false,
							MaxLength:   200,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "expire_minutes",
							Description: "หมดเวลารับสมัครในกี่นาที (ค่าเริ่มต้น 60 นาที)",
							Required:    false,
							MinValue:    &five,
							MaxValue:    dayMinutes,
						},
					},
				},
				Execute: h.executeCommand,
			},
		},
		ComponentHandler: h.handleComponent,
	}
}

func (h *handler) getRepo() *repository {
	if h.repo != nil {
		return h.repo
	}
	if h.Store != nil && h.Store.SQL() != nil {
		h.repo = &repository{db: h.Store.SQL()}
	}
	return h.repo
}

func (h *handler) executeCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	channel, err := s.Channel(i.ChannelID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if channel.GuildID != i.GuildID || channel.Type != discordgo.ChannelTypeGuildText {
		return "สร้างการ์ดนัดเล่นเกมได้ในห้องข้อความปกติของเซิร์ฟเวอร์เท่านั้นครับ", nil
	}

	neededPerms := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks)
	if i.AppPermissions != 0 {
		if i.AppPermissions&discordgo.PermissionAdministrator == 0 && (i.AppPermissions&neededPerms) != neededPerms {
			return "บอทต้องการสิทธิ์ View Channel, Send Messages และ Embed Links ในห้องนี้ครับ", nil
		}
	} else {
		hasPerms, err := featureutil.BotHasChannelPermissions(ctx, s, i.GuildID, h.BotID, channel, neededPerms)
		if err != nil || !hasPerms {
			return "บอทต้องการสิทธิ์ View Channel, Send Messages และ Embed Links ในห้องนี้ครับ", nil
		}
	}

	game := featureutil.String(i, "game", "เกมทั่วไป")
	maxPlayers := int(featureutil.Integer(i, "max_players", 5))
	rawTime := featureutil.String(i, "time", "")
	note := featureutil.String(i, "note", "")
	expireMinutes := int(featureutil.Integer(i, "expire_minutes", 60))

	timeDisplay, timeTimestamp := ParseGameTime(rawTime)

	now := time.Now().Unix()
	expiresAt := now + int64(expireMinutes*60)
	if timeTimestamp > 0 && timeTimestamp > now && timeTimestamp < expiresAt {
		// If game time is earlier than the default expire window, expire at game time
		expiresAt = timeTimestamp
	}

	party := &Party{
		GuildID:    i.GuildID,
		ChannelID:  i.ChannelID,
		HostID:     i.Member.User.ID,
		Game:       game,
		MaxPlayers: maxPlayers,
		TargetTime: timeDisplay,
		Note:       note,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		Status:     "open",
	}

	partyID, err := h.getRepo().createParty(ctx, party)
	if err != nil {
		return "", fmt.Errorf("ไม่สามารถสร้างตี้ในฐานข้อมูลได้: %w", err)
	}

	members, _ := h.getRepo().getMembers(ctx, partyID)

	embed := buildPartyEmbed(party, members)
	components := buildPartyComponents(partyID, false)

	msg, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content:    fmt.Sprintf("📢 <@%s> ได้เปิดตี้ **%s** กดปุ่มด้านล่างเพื่อร่วมเล่นกันเลย!", party.HostID, party.Game),
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}, discordgo.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("ไม่สามารถส่งการ์ดตี้ในห้องนี้ได้: %w", err)
	}

	_ = h.getRepo().setMessageID(ctx, partyID, msg.ID)

	return fmt.Sprintf("✅ สร้างการ์ดนัดเล่นเกม **%s** สำเร็จแล้ว: https://discord.com/channels/%s/%s/%s", game, i.GuildID, i.ChannelID, msg.ID), nil
}

func (h *handler) handleComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	customID := i.MessageComponentData().CustomID
	if !strings.HasPrefix(customID, "party_") {
		return false, nil
	}

	parts := strings.Split(customID, ":")
	if len(parts) != 2 {
		return false, nil
	}

	action := parts[0]
	partyID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false, nil
	}

	party, err := h.getRepo().getParty(ctx, partyID)
	if err != nil || party == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ ไม่พบข้อมูลตี้นี้ในระบบ หรือตี้นี้ถูกปิดไปแล้วครับ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return true, nil
	}

	userID := i.Member.User.ID

	if party.Status == "cancelled" || party.Status == "expired" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ ตี้นี้ปิดรับสมัครแล้วครับ",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return true, nil
	}

	switch action {
	case "party_join":
		res, err := h.getRepo().join(ctx, partyID, userID, "main")
		if err != nil {
			return true, err
		}

		if res.ActionTaken == "main_full" {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "⚠️ ผู้เล่นตัวจริงเต็มแล้ว",
							Description: "ตำแหน่งตัวจริงเต็มแล้วครับ! คุณสามารถกดปุ่ม **\"🟡 สำรอง\"** เพื่อต่อคิวเป็นตัวสำรองได้ครับ",
							Color:       0xFEE75C,
						},
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return true, nil
		}

		if res.ActionTaken == "already_in_role" {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "คุณอยู่ในรายชื่อผู้เล่นตัวจริงอยู่แล้วครับ 👍",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return true, nil
		}

		// Reload and update
		updatedParty, _ := h.getRepo().getParty(ctx, partyID)
		members, _ := h.getRepo().getMembers(ctx, partyID)

		embed := buildPartyEmbed(updatedParty, members)
		components := buildPartyComponents(partyID, false)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})

		// If party just became full, send mention announcement in channel!
		if res.IsNowFull {
			mentions := []string{}
			for _, m := range members {
				if m.Role == "main" {
					mentions = append(mentions, fmt.Sprintf("<@%s>", m.UserID))
				}
			}
			announcement := fmt.Sprintf("🎉 **ตี้ %s คนครบแล้ว!** สมาชิกตัวจริงเตรียมตัวให้พร้อม: %s",
				updatedParty.Game, strings.Join(mentions, " "))
			_, _ = s.ChannelMessageSend(updatedParty.ChannelID, announcement)
		}

		return true, nil

	case "party_sub":
		res, err := h.getRepo().join(ctx, partyID, userID, "sub")
		if err != nil {
			return true, err
		}

		if res.ActionTaken == "already_in_role" {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "คุณอยู่ในรายชื่อผู้เล่นตัวสำรองอยู่แล้วครับ 👍",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return true, nil
		}

		updatedParty, _ := h.getRepo().getParty(ctx, partyID)
		members, _ := h.getRepo().getMembers(ctx, partyID)

		embed := buildPartyEmbed(updatedParty, members)
		components := buildPartyComponents(partyID, false)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
		return true, nil

	case "party_leave":
		res, err := h.getRepo().leave(ctx, partyID, userID)
		if err != nil {
			return true, err
		}

		if !res.WasMember {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "คุณไม่ได้อยู่ในตี้ หรือถอนตัวออกไปแล้วครับ",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return true, nil
		}

		if res.PartyCancelled {
			party.Status = "cancelled"
			embed := buildPartyEmbed(party, []PartyMember{})
			components := buildPartyComponents(partyID, true)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				},
			})
			_, _ = s.ChannelMessageSend(party.ChannelID, fmt.Sprintf("🛑 ตี้ **%s** ถูกยกเลิกแล้วเนื่องจากไม่มีสมาชิกเหลืออยู่", party.Game))
			return true, nil
		}

		updatedParty, _ := h.getRepo().getParty(ctx, partyID)
		members, _ := h.getRepo().getMembers(ctx, partyID)

		embed := buildPartyEmbed(updatedParty, members)
		components := buildPartyComponents(partyID, false)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})

		// Notify if a sub member was automatically promoted to main
		if res.PromotedUserID != "" {
			_, _ = s.ChannelMessageSend(updatedParty.ChannelID,
				fmt.Sprintf("📢 <@%s> ได้รับการเลื่อนขึ้นเป็นตัวจริงในตี้ **%s** แทนที่ว่างแล้ว!", res.PromotedUserID, updatedParty.Game))
		}

		return true, nil

	case "party_cancel":
		// Only host or Admin/ManageServer can cancel
		hasAdmin := false
		if i.Member != nil && i.Member.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageServer) != 0 {
			hasAdmin = true
		}

		if userID != party.HostID && !hasAdmin {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "❌ ไม่มีสิทธิ์",
							Description: fmt.Sprintf("เฉพาะหัวตี้ (<@%s>) หรือผู้ดูแลเซิร์ฟเวอร์เท่านั้นที่สามารถยกเลิกตี้นี้ได้ครับ", party.HostID),
							Color:       0xED4245,
						},
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return true, nil
		}

		_ = h.getRepo().cancel(ctx, partyID)
		party.Status = "cancelled"

		members, _ := h.getRepo().getMembers(ctx, partyID)
		embed := buildPartyEmbed(party, members)
		components := buildPartyComponents(partyID, true) // Disabled

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})

		_, _ = s.ChannelMessageSend(party.ChannelID, fmt.Sprintf("🛑 ตี้ **%s** ถูกยกเลิกโดย <@%s>", party.Game, userID))
		return true, nil
	}

	return false, nil
}

func buildPartyEmbed(p *Party, members []PartyMember) *discordgo.MessageEmbed {
	var mains, subs []PartyMember
	for _, m := range members {
		if m.Role == "main" {
			mains = append(mains, m)
		} else {
			subs = append(subs, m)
		}
	}

	mainLines := []string{}
	for idx, m := range mains {
		tag := ""
		if m.UserID == p.HostID {
			tag = " 👑 (หัวตี้)"
		}
		mainLines = append(mainLines, fmt.Sprintf("**%d.** <@%s>%s", idx+1, m.UserID, tag))
	}
	for idx := len(mains); idx < p.MaxPlayers; idx++ {
		mainLines = append(mainLines, fmt.Sprintf("**%d.** ⚪ *(ว่าง)*", idx+1))
	}

	subLines := []string{}
	if len(subs) == 0 {
		subLines = append(subLines, "*(ยังไม่มีตัวสำรอง)*")
	} else {
		for idx, m := range subs {
			subLines = append(subLines, fmt.Sprintf("**%d.** <@%s>", idx+1, m.UserID))
		}
	}

	title := fmt.Sprintf("🎮 นัดเล่นเกม: %s [%d/%d คน]", p.Game, len(mains), p.MaxPlayers)
	color := 0x5865F2 // Blurple
	statusText := fmt.Sprintf("🟢 กำลังเปิดรับสมัคร (ขาดอีก **%d** คน)", p.MaxPlayers-len(mains))

	if p.Status == "cancelled" {
		title = fmt.Sprintf("🛑 นัดเล่นเกม: %s [ยกเลิกแล้ว]", p.Game)
		color = 0xED4245 // Red
		statusText = "🔴 ตี้นี้ถูกยกเลิกแล้ว"
	} else if p.Status == "expired" {
		title = fmt.Sprintf("⏰ นัดเล่นเกม: %s [หมดเวลารับสมัคร]", p.Game)
		color = 0x747F8D // Gray
		statusText = "⚪ หมดเวลารับสมัครแล้ว (คนยังไม่ครบ)"
	} else if len(mains) >= p.MaxPlayers {
		title = fmt.Sprintf("🎉 นัดเล่นเกม: %s [คนครบแล้ว!]", p.Game)
		color = 0x57F287 // Green
		statusText = "✅ สมาชิกตัวจริงครบแล้ว! เตรียมตัวลุยกันได้เลย"
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "👑 หัวตี้",
			Value:  fmt.Sprintf("<@%s>", p.HostID),
			Inline: true,
		},
		{
			Name:   "⏰ เวลานัดเล่น",
			Value:  p.TargetTime,
			Inline: true,
		},
	}

	if p.Status == "open" || p.Status == "full" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "⏳ หมดเวลารับสมัคร",
			Value:  fmt.Sprintf("<t:%d:R>", p.ExpiresAt),
			Inline: true,
		})
	}

	if p.Note != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📝 หมายเหตุ",
			Value:  p.Note,
			Inline: false,
		})
	}

	fields = append(fields,
		&discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("👥 ผู้เล่นตัวจริง (%d/%d คน)", len(mains), p.MaxPlayers),
			Value:  strings.Join(mainLines, "\n"),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🟡 ผู้เล่นสำรอง (%d คน)", len(subs)),
			Value:  strings.Join(subLines, "\n"),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name:   "📊 สถานะ",
			Value:  statusText,
			Inline: false,
		},
	)

	return &discordgo.MessageEmbed{
		Title:       title,
		Color:       color,
		Fields:      fields,
		Timestamp:   time.Unix(p.CreatedAt, 0).Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Party ID: #%d • กดปุ่มด้านล่างเพื่อเข้าร่วมหรือสำรอง", p.ID),
		},
	}
}

func buildPartyComponents(partyID int64, disabled bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "เข้าร่วม",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("party_join:%d", partyID),
					Emoji:    &discordgo.ComponentEmoji{Name: "🟢"},
					Disabled: disabled,
				},
				discordgo.Button{
					Label:    "สำรอง",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("party_sub:%d", partyID),
					Emoji:    &discordgo.ComponentEmoji{Name: "🟡"},
					Disabled: disabled,
				},
				discordgo.Button{
					Label:    "ถอนตัว",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("party_leave:%d", partyID),
					Emoji:    &discordgo.ComponentEmoji{Name: "⚪"},
					Disabled: disabled,
				},
				discordgo.Button{
					Label:    "ยกเลิกตี้",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("party_cancel:%d", partyID),
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
					Disabled: disabled,
				},
			},
		},
	}
}
