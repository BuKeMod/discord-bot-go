package coinflip

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{Name: "Fun_coinflip", Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "coinflip", Description: "โยนเหรียญหัวหรือก้อย"}, Execute: func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (string, error) {
		v, err := u.Random(2)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("🪙 %s", []string{"หัว", "ก้อย"}[v]), nil
	}}}}
}
