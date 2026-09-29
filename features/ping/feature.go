package ping

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "ping",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "ping",
					Description: "ตรวจสอบความหน่วงและการตอบสนองของบอท (Bot Latency)",
				},
				Execute: func(ctx context.Context, s *discordgo.Session, _ *discordgo.InteractionCreate) (string, error) {
					latency := s.HeartbeatLatency().Round(time.Millisecond)
					color := 0x57F287 // Green
					status := "🟢 ยอดเยี่ยม (Excellent)"
					if latency > 200*time.Millisecond {
						color = 0xFEE75C // Yellow
						status = "🟡 ปานกลาง (Moderate)"
					}
					if latency > 500*time.Millisecond {
						color = 0xED4245 // Red
						status = "🔴 ช้า (Slow)"
					}

					embed := &discordgo.MessageEmbed{
						Title: "🏓 Pong!",
						Color: color,
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "📶 Gateway Heartbeat",
								Value:  fmt.Sprintf("`%s`", latency),
								Inline: true,
							},
							{
								Name:   "⚡ สถานะการเชื่อมต่อ",
								Value:  status,
								Inline: true,
							},
						},
						Timestamp: time.Now().Format(time.RFC3339),
					}
					feature.SetEmbed(ctx, embed)
					return "", nil
				},
			},
		},
	}
}

