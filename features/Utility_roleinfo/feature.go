package roleinfo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "Utility_roleinfo",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "roleinfo",
					Description: "ดูรายละเอียด role โดยไม่แก้ไขสิทธิ์",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "Role ที่ต้องการดู",
							Required:    true,
						},
					},
				},
				Execute: run,
			},
		},
	}
}

func run(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	roles, err := s.GuildRoles(i.GuildID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	id := u.String(i, "role", "")
	for _, r := range roles {
		if r.ID == id {
			created, err := discordgo.SnowflakeTimestamp(id)
			if err != nil {
				return "", err
			}
			flags := []string{}
			for _, p := range []struct {
				name string
				bit  int64
			}{
				{"Administrator", discordgo.PermissionAdministrator},
				{"Manage Server", discordgo.PermissionManageServer},
				{"Manage Roles", discordgo.PermissionManageRoles},
				{"Manage Channels", discordgo.PermissionManageChannels},
				{"Kick Members", discordgo.PermissionKickMembers},
				{"Ban Members", discordgo.PermissionBanMembers},
				{"Manage Messages", discordgo.PermissionManageMessages},
				{"Mention Everyone", discordgo.PermissionMentionEveryone},
				{"View Audit Log", discordgo.PermissionViewAuditLogs},
			} {
				if r.Permissions&p.bit != 0 {
					flags = append(flags, p.name)
				}
			}
			info := "ไม่มีสิทธิ์จัดการหลัก"
			if len(flags) > 0 {
				info = strings.Join(flags, ", ")
			}

			color := r.Color
			if color == 0 {
				color = 0x9B59B6
			}

			embed := &discordgo.MessageEmbed{
				Title: fmt.Sprintf("🏷️ ข้อมูล Role: %s", r.Name),
				Color: color,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "🆔 Role ID",
						Value:  fmt.Sprintf("`%s`", r.ID),
						Inline: true,
					},
					{
						Name:   "🎨 สี (Hex)",
						Value:  fmt.Sprintf("#%06X", r.Color),
						Inline: true,
					},
					{
						Name:   "📊 ตำแหน่ง (Position)",
						Value:  fmt.Sprintf("%d", r.Position),
						Inline: true,
					},
					{
						Name:   "👥 แสดงแยกกลุ่ม (Hoist)",
						Value:  fmt.Sprintf("%t", r.Hoist),
						Inline: true,
					},
					{
						Name:   "📢 เมนชั่นได้ (Mentionable)",
						Value:  fmt.Sprintf("%t", r.Mentionable),
						Inline: true,
					},
					{
						Name:   "🤖 จัดการโดยบอท/ระบบ (Managed)",
						Value:  fmt.Sprintf("%t", r.Managed),
						Inline: true,
					},
					{
						Name:   "📅 วันที่สร้าง",
						Value:  fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()),
						Inline: false,
					},
					{
						Name:   "🛡️ สิทธิ์จัดการหลัก (Key Permissions)",
						Value:  info,
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Role ID: %s", r.ID),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			feature.SetEmbed(ctx, embed)
			return "", nil
		}
	}
	return "ไม่พบ role นี้", nil
}

