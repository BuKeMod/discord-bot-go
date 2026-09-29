package choose

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{Name: "Fun_choose", Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "choose", Description: "สุ่มเลือกหนึ่งคำตอบจากตัวเลือก", Options: []*discordgo.ApplicationCommandOption{u.TextOption("choices", "2–25 ตัวเลือก คั่นด้วย | เช่น ข้าว | ก๋วยเตี๋ยว", true, 2000)}}, Execute: run}}}
}
func run(_ context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	values, err := u.Choices(u.String(i, "choices", ""), 2, 25, 80)
	if err != nil {
		return err.Error(), nil
	}
	n, err := u.Random(int64(len(values)))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("🎯 %s", values[n]), nil
}
