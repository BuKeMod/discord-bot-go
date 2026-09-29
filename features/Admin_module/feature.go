package admin_module

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/storage"
)

type handler struct {
	feature.Dependencies
}

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}
	adminPerm := int64(discordgo.PermissionAdministrator)

	return feature.Module{
		Name: "Admin_module",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "module",
					Description:              "จัดการเปิด/ปิดโมดูลของบอทในเซิร์ฟเวอร์นี้ (สำหรับผู้ดูแลระบบ)",
					DefaultMemberPermissions: &adminPerm,
				},
				Execute: h.executeModule,
			},
		},
		ComponentHandler: h.handleComponent,
	}
}

func (h *handler) executeModule(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	if h.Store == nil {
		return "ระบบฐานข้อมูลไม่พร้อมใช้งาน ไม่สามารถเปิด/ปิดโมดูลได้", nil
	}

	guildName := "เซิร์ฟเวอร์นี้"
	if s.State != nil {
		if g, err := s.State.Guild(i.GuildID); err == nil && g != nil {
			guildName = g.Name
		}
	}

	embed, components, err := buildControlPanel(ctx, h.Store, i.GuildID, guildName)
	if err != nil {
		return "", err
	}

	feature.SetEmbed(ctx, embed)
	feature.SetComponents(ctx, components)
	return "", nil
}

func (h *handler) handleComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	customID := i.MessageComponentData().CustomID
	if !strings.HasPrefix(customID, "module_") {
		return false, nil
	}

	// Security check: Only administrator can toggle modules
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ เฉพาะผู้ดูแลระบบ (Administrator) เท่านั้นที่มีสิทธิ์เปลี่ยนการตั้งค่าโมดูล",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return true, nil
	}

	if h.Store == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ ฐานข้อมูลไม่พร้อมใช้งานในขณะนี้",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return true, nil
	}

	allModules := []string{"moderation", "logs", "tickets", "party", "voice", "welcome", "roles", "fun", "utility"}

	switch customID {
	case "module_toggle_select":
		values := i.MessageComponentData().Values
		if len(values) > 0 {
			targetMod := values[0]
			current := h.Store.IsModuleEnabled(ctx, i.GuildID, targetMod)
			_ = h.Store.SetModuleEnabled(ctx, i.GuildID, targetMod, !current)
		}
	case "module_action_enable_all":
		for _, mod := range allModules {
			_ = h.Store.SetModuleEnabled(ctx, i.GuildID, mod, true)
		}
	case "module_action_disable_all":
		for _, mod := range allModules {
			_ = h.Store.SetModuleEnabled(ctx, i.GuildID, mod, false)
		}
	case "module_action_refresh":
		// Just re-fetch and re-render
	}

	guildName := "เซิร์ฟเวอร์นี้"
	if s.State != nil {
		if g, err := s.State.Guild(i.GuildID); err == nil && g != nil {
			guildName = g.Name
		}
	}

	embed, components, err := buildControlPanel(ctx, h.Store, i.GuildID, guildName)
	if err != nil {
		return true, err
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	return true, nil
}

func buildControlPanel(ctx context.Context, store *storage.Store, guildID, guildName string) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	modules, err := store.GetGuildModules(ctx, guildID)
	if err != nil {
		return nil, nil, err
	}

	ordered := []struct {
		key   string
		name  string
		desc  string
		emoji string
	}{
		{"moderation", "Moderation", "ระบบจัดการสมาชิก /ban, /kick, /timeout, /warn, /purge", "🛡️"},
		{"logs", "Logs", "ระบบบันทึกข้อความ, ย้ายห้องเสียง, เข้า/ออกเซิร์ฟเวอร์", "📋"},
		{"tickets", "Tickets", "ระบบตั๋วช่วยเหลือ Support และ Transcript", "🎫"},
		{"party", "Party (LFG)", "การ์ดนัดเล่นเกมแบบ Interactive พร้อมปุ่มกด", "🎮"},
		{"voice", "Voice Activity", "การจับเวลาในห้องเสียง และจัดอันดับ /voicetop", "🎙️"},
		{"welcome", "Welcome", "ข้อความต้อนรับสมาชิกใหม่เมื่อเข้าเซิร์ฟเวอร์", "🎭"},
		{"roles", "Roles", "ระบบเลือกรับยศด้วยตนเอง /role", "🏷️"},
		{"fun", "Fun", "คำสั่งบันเทิง /roll, /coinflip, /choose", "🎲"},
		{"utility", "Utility", "เครื่องมือทั่วไป /avatar, /poll, /remind ฯลฯ", "⚙️"},
	}

	var statusLines []string
	var selectOptions []discordgo.SelectMenuOption

	for _, item := range ordered {
		enabled := modules[item.key]
		statusEmoji := "🔴 ปิดใช้งาน"
		actionLabel := "[ปิดอยู่ ➔ คลิกเพื่อเปิด]"
		if enabled {
			statusEmoji = "🟢 เปิดใช้งาน"
			actionLabel = "[เปิดอยู่ ➔ คลิกเพื่อปิด]"
		}
		statusLines = append(statusLines, fmt.Sprintf("• %s **%s**: %s", item.emoji, item.name, statusEmoji))

		selectOptions = append(selectOptions, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("%s %s %s", item.emoji, item.name, actionLabel),
			Value:       item.key,
			Description: item.desc,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🎛️ แผงควบคุมโมดูล (Module Control Panel) — %s", guildName),
		Color: 0x5865F2,
		Description: "เลือกโมดูลจาก **เมนูดรอปดาวน์ด้านล่าง** เพื่อสลับเปิดหรือปิดการใช้งานในเซิร์ฟเวอร์นี้ทันที\n" +
			"*(เมื่อปิดโมดูล คำสั่งและระบบเบื้องหลังของโมดูลนั้นจะหยุดทำงานเฉพาะในเซิร์ฟเวอร์นี้)*\n\n" +
			strings.Join(statusLines, "\n"),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "เฉพาะ Administrator เท่านั้นที่มีสิทธิ์เปลี่ยนการตั้งค่าโมดูล",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "module_toggle_select",
					Placeholder: "🔘 เลือกโมดูลที่ต้องการ สลับเปิด/ปิด (Toggle)...",
					MinValues:   &[]int{1}[0],
					MaxValues:   1,
					Options:     selectOptions,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "เปิดทั้งหมด (Enable All)",
					Style:    discordgo.SuccessButton,
					CustomID: "module_action_enable_all",
					Emoji:    &discordgo.ComponentEmoji{Name: "🟢"},
				},
				discordgo.Button{
					Label:    "ปิดทั้งหมด (Disable All)",
					Style:    discordgo.DangerButton,
					CustomID: "module_action_disable_all",
					Emoji:    &discordgo.ComponentEmoji{Name: "🔴"},
				},
				discordgo.Button{
					Label:    "รีเฟรช (Refresh)",
					Style:    discordgo.SecondaryButton,
					CustomID: "module_action_refresh",
					Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
				},
			},
		},
	}

	return embed, components, nil
}
