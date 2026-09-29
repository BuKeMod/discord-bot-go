package help

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

func New(d feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "help",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "help",
					Description: "แสดงรายการคำสั่งทั้งหมดและวิธีใช้งาน",
				},
				Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
					targetGuild := i.GuildID
					if targetGuild == "" {
						targetGuild = d.Config.GuildID
					}
					defs, err := s.ApplicationCommands(d.BotID, targetGuild, discordgo.WithContext(ctx))
					if err != nil || len(defs) == 0 {
						defs, err = s.ApplicationCommands(d.BotID, "", discordgo.WithContext(ctx))
					}
					if err != nil {
						return "", err
					}
					sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

					categories := map[string][]string{
						"🛡️ จัดการเซิร์ฟเวอร์ (Moderation)": {},
						"🎮 นัดเล่นเกม (Party/LFG)":           {},
						"🎫 ระบบตั๋วช่วยเหลือ (Ticket)":     {},
						"📋 การตั้งค่าล็อก (Logs Config)":     {},
						"🎙️ กิจกรรมห้องเสียง (Voice)":         {},
						"🎭 บทบาท & ระบบต้อนรับ":             {},
						"⚙️ ข้อมูล & ยูทิลิตี้ (Utility)":     {},
						"🎲 บันเทิง & มินิเกม (Fun)":          {},
					}

					for _, def := range defs {
						if !feature.CanUse(def, i.Member) {
							continue
						}
						line := fmt.Sprintf("`/%s` — %s", def.Name, def.Description)

						switch def.Name {
						case "ban", "unban", "kick", "timeout", "untimeout", "warn", "purge", "status", "module":
							categories["🛡️ จัดการเซิร์ฟเวอร์ (Moderation)"] = append(categories["🛡️ จัดการเซิร์ฟเวอร์ (Moderation)"], line)
						case "party":
							categories["🎮 นัดเล่นเกม (Party/LFG)"] = append(categories["🎮 นัดเล่นเกม (Party/LFG)"], line)
						case "ticket":
							categories["🎫 ระบบตั๋วช่วยเหลือ (Ticket)"] = append(categories["🎫 ระบบตั๋วช่วยเหลือ (Ticket)"], line)
						case "logconfig":
							categories["📋 การตั้งค่าล็อก (Logs Config)"] = append(categories["📋 การตั้งค่าล็อก (Logs Config)"], line)
						case "voiceactivity", "voicetop":
							categories["🎙️ กิจกรรมห้องเสียง (Voice)"] = append(categories["🎙️ กิจกรรมห้องเสียง (Voice)"], line)
						case "role", "roleconfig", "welcome":
							categories["🎭 บทบาท & ระบบต้อนรับ"] = append(categories["🎭 บทบาท & ระบบต้อนรับ"], line)
						case "roll", "coinflip", "choose":
							categories["🎲 บันเทิง & มินิเกม (Fun)"] = append(categories["🎲 บันเทิง & มินิเกม (Fun)"], line)
						default:
							categories["⚙️ ข้อมูล & ยูทิลิตี้ (Utility)"] = append(categories["⚙️ ข้อมูล & ยูทิลิตี้ (Utility)"], line)
						}
					}

					embed := &discordgo.MessageEmbed{
						Title:       "📖 รายการคำสั่งทั้งหมด (Commands Help)",
						Description: "รายการคำสั่ง slash command ที่คุณมีสิทธิ์ใช้งานในเซิร์ฟเวอร์นี้:",
						Color:       0x5865F2,
						Timestamp:   time.Now().Format(time.RFC3339),
						Footer: &discordgo.MessageEmbedFooter{
							Text: "พิมพ์ / ตามด้วยชื่อคำสั่งเพื่อดูตัวเลือกเพิ่มเติม",
						},
					}

					catOrder := []string{
						"🛡️ จัดการเซิร์ฟเวอร์ (Moderation)",
						"🎮 นัดเล่นเกม (Party/LFG)",
						"🎫 ระบบตั๋วช่วยเหลือ (Ticket)",
						"📋 การตั้งค่าล็อก (Logs Config)",
						"🎙️ กิจกรรมห้องเสียง (Voice)",
						"🎭 บทบาท & ระบบต้อนรับ",
						"⚙️ ข้อมูล & ยูทิลิตี้ (Utility)",
						"🎲 บันเทิง & มินิเกม (Fun)",
					}

					for _, cat := range catOrder {
						cmds := categories[cat]
						if len(cmds) > 0 {
							embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
								Name:   cat,
								Value:  strings.Join(cmds, "\n"),
								Inline: false,
							})
						}
					}

					feature.SetEmbed(ctx, embed)
					return "", nil
				},
			},
		},
	}
}

