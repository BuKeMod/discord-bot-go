package userinfo

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
		Name: "Utility_userinfo",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "userinfo",
					Description: "วันสร้างบัญชี วันเข้าเซิร์ฟเวอร์ และ roles",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการดู",
						},
					},
				},
				Execute: run,
			},
		},
	}
}

func run(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	id := u.UserID(i)
	user, err := s.User(id, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	created, err := discordgo.SnowflakeTimestamp(id)
	if err != nil {
		return "", err
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("👤 ข้อมูลสมาชิก: %s", user.Username),
		Color: 0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🆔 User ID",
				Value:  fmt.Sprintf("`%s`", id),
				Inline: true,
			},
			{
				Name:   "🎂 วันสร้างบัญชี",
				Value:  fmt.Sprintf("<t:%d:F>\n(<t:%d:R> • %d วันที่แล้ว)", created.Unix(), created.Unix(), int(time.Since(created).Hours()/24)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", id),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	member, err := s.GuildMember(i.GuildID, id, discordgo.WithContext(ctx))
	if err == nil && member != nil {
		if !member.JoinedAt.IsZero() {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "📥 วันเข้าร่วมเซิร์ฟเวอร์",
				Value:  fmt.Sprintf("<t:%d:F>\n(<t:%d:R>)", member.JoinedAt.Unix(), member.JoinedAt.Unix()),
				Inline: false,
			})
		}
		roles := []string{}
		for _, r := range member.Roles {
			roles = append(roles, "<@&"+r+">")
		}
		rolesText := "ไม่มี"
		if len(roles) > 0 {
			var sb strings.Builder
			for n, r := range roles {
				if sb.Len()+len(r) > 900 {
					sb.WriteString(fmt.Sprintf("… อีก %d roles", len(roles)-n))
					break
				}
				sb.WriteString(r + " ")
			}
			rolesText = sb.String()
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🏷️ Roles (%d)", len(roles)),
			Value:  rolesText,
			Inline: false,
		})
	}

	feature.SetEmbed(ctx, embed)
	return "", nil
}

