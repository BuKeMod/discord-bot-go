package timestamp

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(_ feature.Dependencies) feature.Module {
	styles := []*discordgo.ApplicationCommandOptionChoice{}
	for _, v := range []string{"t", "T", "d", "D", "f", "F", "R"} {
		styles = append(styles, &discordgo.ApplicationCommandOptionChoice{Name: v, Value: v})
	}
	return feature.Module{Name: "Utility_timestamp", Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "timestamp", Description: "สร้าง Discord timestamp ที่แสดงตามเขตเวลาผู้ดู", Options: []*discordgo.ApplicationCommandOption{u.TextOption("time", "เช่น 2026-10-01T20:00:00+07:00 หรือ Unix seconds", true, 80), {Type: discordgo.ApplicationCommandOptionString, Name: "style", Description: "รูปแบบ (ค่าเริ่มต้น F)", Choices: styles}}}, Execute: run}}}
}
func run(_ context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	t, err := u.Time(u.String(i, "time", ""))
	if err != nil {
		return err.Error(), nil
	}
	style := u.String(i, "style", "F")
	if len(style) != 1 || !contains(style) {
		return "รูปแบบ timestamp ไม่ถูกต้อง", nil
	}
	code := fmt.Sprintf("<t:%d:%s>", t.Unix(), style)
	return code + "\nคัดลอก: `" + code + "`", nil
}
func contains(s string) bool {
	for _, v := range []string{"t", "T", "d", "D", "f", "F", "R"} {
		if s == v {
			return true
		}
	}
	return false
}
