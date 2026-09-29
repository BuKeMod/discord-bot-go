package serverinfo

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "Utility_serverinfo",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "serverinfo",
					Description: "จำนวนสมาชิก ห้อง และ boosts ของเซิร์ฟเวอร์",
				},
				Execute: run,
			},
		},
	}
}

func run(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	g, err := s.GuildWithCounts(i.GuildID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	channels, err := s.GuildChannels(i.GuildID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	roles, _ := s.GuildRoles(i.GuildID, discordgo.WithContext(ctx))

	count, categories := 0, 0
	for _, c := range channels {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			categories++
		} else {
			count++
		}
	}

	created, _ := discordgo.SnowflakeTimestamp(g.ID)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("ℹ️ ข้อมูลเซิร์ฟเวอร์: %s", u.Clip(g.Name, 100)),
		Color: 0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: g.IconURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "👑 เจ้าของเซิร์ฟเวอร์",
				Value:  fmt.Sprintf("<@%s>", g.OwnerID),
				Inline: true,
			},
			{
				Name:   "🆔 Server ID",
				Value:  fmt.Sprintf("`%s`", g.ID),
				Inline: true,
			},
			{
				Name:   "👥 จำนวนสมาชิก",
				Value:  fmt.Sprintf("ทั้งหมด: **%d** คน\nออนไลน์: **%d** คน", g.ApproximateMemberCount, g.ApproximatePresenceCount),
				Inline: true,
			},
			{
				Name:   "💬 ช่อง & หมวดหมู่",
				Value:  fmt.Sprintf("ห้องข้อความ/เสียง: **%d**\nหมวดหมู่: **%d**", count, categories),
				Inline: true,
			},
			{
				Name:   "🚀 การ Boost",
				Value:  fmt.Sprintf("Boosts: **%d** (ระดับ Tier %d)", g.PremiumSubscriptionCount, g.PremiumTier),
				Inline: true,
			},
			{
				Name:   "🏷️ Roles ทั้งหมด",
				Value:  fmt.Sprintf("**%d** Roles", len(roles)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Server ID: %s", g.ID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if !created.IsZero() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📅 วันที่สร้างเซิร์ฟเวอร์",
			Value:  fmt.Sprintf("<t:%d:F> (<t:%d:R>)", created.Unix(), created.Unix()),
			Inline: false,
		})
	}

	feature.SetEmbed(ctx, embed)
	return "", nil
}

