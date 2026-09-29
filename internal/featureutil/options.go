package featureutil

import (
	"crypto/rand"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func Option(i *discordgo.InteractionCreate, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == name {
			return o
		}
	}
	return nil
}
func String(i *discordgo.InteractionCreate, name, fallback string) string {
	if o := Option(i, name); o != nil {
		if v, ok := o.Value.(string); ok {
			return v
		}
	}
	return fallback
}
func Integer(i *discordgo.InteractionCreate, name string, fallback int64) int64 {
	if o := Option(i, name); o != nil {
		return o.IntValue()
	}
	return fallback
}
func Boolean(i *discordgo.InteractionCreate, name string) bool {
	if o := Option(i, name); o != nil {
		return o.BoolValue()
	}
	return false
}
func UserID(i *discordgo.InteractionCreate) string { return String(i, "user", i.Member.User.ID) }
func TextOption(name, description string, required bool, max int) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: name, Description: description, Required: required, MaxLength: max}
}
func IntOption(name, description string, min, max float64) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: name, Description: description, MinValue: &min, MaxValue: max}
}
func Choices(text string, min, max, length int) ([]string, error) {
	values := strings.Split(text, "|")
	if len(values) < min || len(values) > max {
		return nil, fmt.Errorf("ใส่ %d–%d ตัวเลือก คั่นด้วย |", min, max)
	}
	seen := map[string]bool{}
	for i, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || utf8.RuneCountInString(v) > length {
			return nil, fmt.Errorf("ตัวเลือกต้องไม่ว่างและยาวไม่เกิน %d ตัวอักษร", length)
		}
		if seen[strings.ToLower(v)] {
			return nil, fmt.Errorf("ตัวเลือกต้องไม่ซ้ำกัน")
		}
		seen[strings.ToLower(v)] = true
		values[i] = v
	}
	return values, nil
}
func Random(n int64) (int64, error) {
	if n < 1 {
		return 0, fmt.Errorf("invalid random bound")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}
func Time(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04 -07:00"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil && unix >= 0 && unix <= 253402300799 {
		return time.Unix(unix, 0), nil
	}
	return time.Time{}, fmt.Errorf("ใส่เวลา เช่น 2026-10-01T20:00:00+07:00 หรือ Unix seconds")
}
func Clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
