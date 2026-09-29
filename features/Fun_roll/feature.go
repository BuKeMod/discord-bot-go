package roll

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

var (
	dicePattern      = regexp.MustCompile(`^(\d{1,3})[dD](\d{1,7})$`)
	singleDPattern   = regexp.MustCompile(`^[dD](\d{1,7})$`)
	singleNumPattern = regexp.MustCompile(`^(\d{1,7})$`)
	rangePattern     = regexp.MustCompile(`^(-?\d+)\s*(?:-|ถึง|~|\.\.)\s*(-?\d+)$`)
	diceEmojis       = []string{"⚀", "⚁", "⚂", "⚃", "⚄", "⚅"}
)

func New(_ feature.Dependencies) feature.Module {
	return feature.Module{
		Name: "Fun_roll",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "roll",
					Description: "ทอยลูกเต๋า หรือสุ่มตัวเลข (เช่น 6, 100, 2d6 หรือระบุ min/max)",
					Options: []*discordgo.ApplicationCommandOption{
						u.TextOption("dice", "จำนวนหน้าหรือรูปแบบ เช่น 6, 100, 2d6 หรือ 1-100 (ค่าเริ่มต้น 6 หน้า)", false, 30),
						u.IntOption("min", "เลขต่ำสุด (เช่น 1 หรือ 10)", -1e9, 1e9),
						u.IntOption("max", "เลขสูงสุด (เช่น 100)", -1e9, 1e9),
					},
				},
				Execute: run,
			},
		},
	}
}

// ParseDice parses dice formats: "2d6", "d20", "6", "100"
func ParseDice(s string) (int, int, error) {
	trimmed := strings.TrimSpace(s)

	if m := dicePattern.FindStringSubmatch(trimmed); m != nil {
		count, _ := strconv.Atoi(m[1])
		sides, _ := strconv.Atoi(m[2])
		if count < 1 || count > 100 || sides < 1 || sides > 1000000 {
			return 0, 0, fmt.Errorf("จำนวนลูกเต๋าต้องอยู่ระหว่าง 1–100 ลูก และหน้าเต๋า 1–1,000,000 หน้า")
		}
		return count, sides, nil
	}

	if m := singleDPattern.FindStringSubmatch(trimmed); m != nil {
		sides, _ := strconv.Atoi(m[1])
		if sides < 1 || sides > 1000000 {
			return 0, 0, fmt.Errorf("หน้าเต๋าต้องอยู่ระหว่าง 1–1,000,000 หน้า")
		}
		return 1, sides, nil
	}

	if m := singleNumPattern.FindStringSubmatch(trimmed); m != nil {
		sides, _ := strconv.Atoi(m[1])
		if sides < 1 || sides > 1000000 {
			return 0, 0, fmt.Errorf("หน้าเต๋าต้องอยู่ระหว่าง 1–1,000,000 หน้า")
		}
		return 1, sides, nil
	}

	return 0, 0, fmt.Errorf("รูปแบบไม่ถูกต้อง เช่น ใส่ 6 (ทอยเต๋า 6 หน้า) หรือ 2d6 (2 ลูก 6 หน้า) หรือ 1-100")
}

func run(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	// 1. Check if min or max is passed
	hasMin := u.Option(i, "min") != nil
	hasMax := u.Option(i, "max") != nil

	if hasMin || hasMax {
		if u.Option(i, "dice") != nil {
			return "กรุณาเลือกระหว่างใส่ช่อง `dice` หรือระบุช่วง `min`/`max` อย่างใดอย่างหนึ่งครับ", nil
		}

		low := int64(1)
		high := int64(100)

		if hasMin && hasMax {
			low = u.Integer(i, "min", 1)
			high = u.Integer(i, "max", 100)
		} else if hasMax {
			high = u.Integer(i, "max", 100)
			if high < 1 {
				low = 0
			}
		} else { // only min
			low = u.Integer(i, "min", 1)
			high = low + 100
		}

		if low > high || low < -1e9 || high > 1e9 {
			return "ช่วงตัวเลขไม่ถูกต้อง (เลขต่ำสุดต้องไม่มากกว่าเลขสูงสุด)", nil
		}

		n, err := u.Random(high - low + 1)
		if err != nil {
			return "", err
		}
		result := low + n

		embed := &discordgo.MessageEmbed{
			Title: "🎯 ผลการสุ่มตัวเลข (Random Number)",
			Color: 0xFEE75C,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "ช่วงตัวเลข",
					Value:  fmt.Sprintf("`%d` ถึง `%d`", low, high),
					Inline: true,
				},
				{
					Name:   "เลขที่สุ่มได้",
					Value:  fmt.Sprintf("🎉 **%d**", result),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("จากความเป็นไปได้ทั้งหมด %d จำนวน", high-low+1),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		feature.SetEmbed(ctx, embed)
		return "", nil
	}

	diceInput := strings.TrimSpace(u.String(i, "dice", "6"))
	if diceInput == "" {
		diceInput = "6"
	}

	// Check if input is a range string like "1-100" or "10 ถึง 50"
	if rm := rangePattern.FindStringSubmatch(diceInput); rm != nil {
		low, err1 := strconv.ParseInt(rm[1], 10, 64)
		high, err2 := strconv.ParseInt(rm[2], 10, 64)
		if err1 == nil && err2 == nil {
			if low > high {
				low, high = high, low
			}
			if high-low+1 > 0 && high-low+1 <= 2e9 {
				n, err := u.Random(high - low + 1)
				if err != nil {
					return "", err
				}
				result := low + n

				embed := &discordgo.MessageEmbed{
					Title: "🎯 ผลการสุ่มตัวเลข (Random Number)",
					Color: 0xFEE75C,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "ช่วงตัวเลข",
							Value:  fmt.Sprintf("`%d` ถึง `%d`", low, high),
							Inline: true,
						},
						{
							Name:   "เลขที่สุ่มได้",
							Value:  fmt.Sprintf("🎉 **%d**", result),
							Inline: true,
						},
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("จากความเป็นไปได้ทั้งหมด %d จำนวน", high-low+1),
					},
					Timestamp: time.Now().Format(time.RFC3339),
				}
				feature.SetEmbed(ctx, embed)
				return "", nil
			}
		}
	}

	count, sides, err := ParseDice(diceInput)
	if err != nil {
		return err.Error(), nil
	}

	values := []int64{}
	valStrs := []string{}
	sum := int64(0)

	for j := 0; j < count; j++ {
		v, err := u.Random(int64(sides))
		if err != nil {
			return "", err
		}
		v++
		sum += v
		values = append(values, v)

		if sides == 6 && v >= 1 && v <= 6 {
			valStrs = append(valStrs, fmt.Sprintf("%s `%d`", diceEmojis[v-1], v))
		} else {
			valStrs = append(valStrs, fmt.Sprintf("`%d`", v))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:     "🎲 ผลการทอยลูกเต๋า (Roll)",
		Color:     0xFEE75C,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if count == 1 {
		resultDisplay := fmt.Sprintf("**%d**", values[0])
		if sides == 6 && values[0] >= 1 && values[0] <= 6 {
			resultDisplay = fmt.Sprintf("%s **%d**", diceEmojis[values[0]-1], values[0])
		}

		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "ลูกเต๋า",
				Value:  fmt.Sprintf("%d หน้า (1 ลูก)", sides),
				Inline: true,
			},
			{
				Name:   "แต้มที่ได้",
				Value:  resultDisplay,
				Inline: true,
			},
		}
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "💡 ทิป: พิมพ์ /roll dice:100 หรือ /roll dice:2d6 (ทอย 2 ลูก)",
		}
	} else {
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "ลูกเต๋า",
				Value:  fmt.Sprintf("%d หน้า จำนวน %d ลูก (`%dd%d`)", sides, count, count, sides),
				Inline: false,
			},
			{
				Name:   "แต้มแต่ละลูก",
				Value:  strings.Join(valStrs, "  +  "),
				Inline: false,
			},
			{
				Name:   "แต้มรวมทั้งหมด",
				Value:  fmt.Sprintf("🏆 **%d** (ค่าเฉลี่ย: %.1f)", sum, float64(sum)/float64(count)),
				Inline: false,
			},
		}
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("ทอยทั้งหมด %d ลูก ด้านละ %d หน้า", count, sides),
		}
	}

	feature.SetEmbed(ctx, embed)
	return "", nil
}

