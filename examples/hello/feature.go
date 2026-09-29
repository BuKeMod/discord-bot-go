package hello

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

// Copy this folder to features/hello; the next build discovers it automatically.
func New(_ feature.Dependencies) feature.Module {
	return feature.Module{Name: "hello", Commands: []feature.Command{{
		Definition: &discordgo.ApplicationCommand{Name: "hello", Description: "Say hello"},
		Execute: func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (string, error) {
			return "สวัสดีครับ!", nil
		},
	}}}
}
