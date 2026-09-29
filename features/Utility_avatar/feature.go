package avatar

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "Utility_avatar",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "avatar",
					Description: "ดูรูปโปรไฟล์ของคุณหรือสมาชิกอื่น",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "สมาชิกที่ต้องการดูรูป",
						},
					},
				},
				Execute: run,
			},
		},
	}
}

func run(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	user, err := s.User(u.UserID(i), discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}

	avatarURL := user.AvatarURL("1024")
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🖼️ รูปโปรไฟล์ของ %s", user.Username),
		Description: fmt.Sprintf("[คลิกเพื่อดูภาพขนาดเต็ม](%s)", avatarURL),
		Color:       0x5865F2,
		Image: &discordgo.MessageEmbedImage{
			URL: avatarURL,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", user.ID),
		},
	}
	feature.SetEmbed(ctx, embed)
	return "", nil
}

