package commands

import (
	"context"

	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/config"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/storage"
)

var CommandToModule = map[string]string{
	"ban": "moderation", "unban": "moderation", "kick": "moderation",
	"timeout": "moderation", "untimeout": "moderation", "warn": "moderation", "purge": "moderation",
	"logconfig": "logs",
	"ticket":    "tickets",
	"party":     "party",
	"voiceactivity": "voice", "voicetop": "voice",
	"welcome":    "welcome",
	"role":       "roles", "roleconfig": "roles",
	"roll":       "fun", "coinflip": "fun", "choose": "fun",
	"avatar":     "utility", "userinfo": "utility", "serverinfo": "utility",
	"roleinfo":   "utility", "poll": "utility", "remind": "utility", "timestamp": "utility",
}

func ModuleDisplayName(mod string) string {
	switch mod {
	case "moderation":
		return "🛡️ Moderation (จัดการเซิร์ฟเวอร์)"
	case "logs":
		return "📋 Logs (บันทึกการทำงาน)"
	case "tickets":
		return "🎫 Tickets (ระบบตั๋วช่วยเหลือ)"
	case "party":
		return "🎮 Party (การ์ดนัดเล่นเกม)"
	case "voice":
		return "🎙️ Voice Activity (กิจกรรมห้องเสียง)"
	case "welcome":
		return "🎭 Welcome (ระบบต้อนรับ)"
	case "roles":
		return "🏷️ Roles (ระบบเลือกยศ)"
	case "fun":
		return "🎲 Fun (มินิเกม)"
	case "utility":
		return "⚙️ Utility (ยูทิลิตี้)"
	default:
		return mod
	}
}

func ComponentToModule(customID string) string {
	if strings.HasPrefix(customID, "party_") {
		return "party"
	}
	if strings.HasPrefix(customID, "ticket_") {
		return "tickets"
	}
	return ""
}

type Handler struct {
	Registry *feature.Registry
	Config   config.Config
	Store    *storage.Store
}

func (h *Handler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !h.Config.MultiGuild && i.GuildID != h.Config.GuildID {
		return
	}
	if i.Member == nil || i.Member.User == nil {
		return
	}
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID
		if mod := ComponentToModule(customID); mod != "" && h.Store != nil {
			if !h.Store.IsModuleEnabled(context.Background(), i.GuildID, mod) {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("⚠️ ฟังก์ชันนี้อยู่ในโมดูล **%s** ซึ่งถูกปิดใช้งานในเซิร์ฟเวอร์นี้โดยผู้ดูแลระบบ", ModuleDisplayName(mod)),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := h.Registry.ExecuteComponent(ctx, s, i); err != nil {
			slog.Error("component handler failed", "custom_id", customID, "error", err)
		}
		return
	}
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	cmdName := i.ApplicationCommandData().Name
	if mod, ok := CommandToModule[cmdName]; ok && h.Store != nil {
		if !h.Store.IsModuleEnabled(context.Background(), i.GuildID, mod) {
			errMsg := fmt.Sprintf("⚠️ คำสั่ง `/%s` อยู่ในโมดูล **%s** ซึ่งถูกปิดใช้งานในเซิร์ฟเวอร์นี้โดยผู้ดูแลระบบ", cmdName, ModuleDisplayName(mod))
			embed := BuildEmbed(cmdName, errMsg, true)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
					Flags:  discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		slog.Warn("interaction acknowledgement failed")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	ctx, holder := feature.ContextWithResponse(ctx)
	content, err := h.execute(ctx, s, i)
	if err != nil {
		slog.Error("command failed", "command", i.ApplicationCommandData().Name, "error", err)
		content = "ทำรายการไม่สำเร็จ โปรดตรวจสอบสิทธิ์บอทและ log"
	}

	var embeds []*discordgo.MessageEmbed
	if holder.Embeds != nil && len(holder.Embeds) > 0 {
		embeds = holder.Embeds
	} else if holder.Embed != nil {
		embeds = []*discordgo.MessageEmbed{holder.Embed}
	} else {
		embeds = []*discordgo.MessageEmbed{BuildEmbed(cmdName, content, err != nil)}
	}

	edit := &discordgo.WebhookEdit{
		Embeds:          &embeds,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	}
	if holder.Components != nil {
		edit.Components = &holder.Components
	}
	_, err = s.InteractionResponseEdit(i.Interaction, edit)
	if err != nil {
		slog.Warn("interaction response failed")
	}
}
func (h *Handler) execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	return h.Registry.Execute(ctx, s, i)
}
