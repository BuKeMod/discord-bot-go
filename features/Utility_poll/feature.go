package poll

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
)

func New(d feature.Dependencies) feature.Module {
	permission := int64(discordgo.PermissionSendPolls)
	return feature.Module{Name: "Utility_poll", Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "poll", Description: "สร้างโหวตให้สมาชิกในห้องตอบ", DefaultMemberPermissions: &permission, Options: []*discordgo.ApplicationCommandOption{
		u.TextOption("question", "คำถาม (สูงสุด 300 ตัวอักษร)", true, 300), u.TextOption("choices", "2–10 ตัวเลือก คั่นด้วย | ตัวเลือกละไม่เกิน 55 ตัวอักษร", true, 600), u.IntOption("hours", "ระยะเวลาเป็นชั่วโมง (ค่าเริ่มต้น 24)", 1, 768), {Type: discordgo.ApplicationCommandOptionBoolean, Name: "multiple", Description: "ให้เลือกได้หลายคำตอบ"},
	}}, Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
		return run(ctx, s, i, d.BotID)
	}}}}
}
func Build(question, choices string, hours int, multiple bool) (*discordgo.Poll, error) {
	if len([]rune(question)) < 1 || len([]rune(question)) > 300 {
		return nil, fmt.Errorf("คำถามต้องยาว 1–300 ตัวอักษร")
	}
	if hours < 1 || hours > 768 {
		return nil, fmt.Errorf("ระยะเวลาต้องเป็น 1–768 ชั่วโมง")
	}
	values, err := u.Choices(choices, 2, 10, 55)
	if err != nil {
		return nil, err
	}
	poll := &discordgo.Poll{Question: discordgo.PollMedia{Text: question}, Duration: hours, AllowMultiselect: multiple, LayoutType: discordgo.PollLayoutTypeDefault}
	for _, v := range values {
		poll.Answers = append(poll.Answers, discordgo.PollAnswer{Media: &discordgo.PollMedia{Text: v}})
	}
	return poll, nil
}
func run(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, botID string) (string, error) {
	poll, err := Build(u.String(i, "question", ""), u.String(i, "choices", ""), int(u.Integer(i, "hours", 24)), u.Boolean(i, "multiple"))
	if err != nil {
		return err.Error(), nil
	}
	channel, err := s.Channel(i.ChannelID, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	if channel.GuildID != i.GuildID || channel.Type != discordgo.ChannelTypeGuildText {
		return "สร้างโหวตได้ในห้องข้อความปกติของเซิร์ฟเวอร์เท่านั้น", nil
	}
	needed := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionSendPolls)
	if i.AppPermissions != 0 {
		if i.AppPermissions&discordgo.PermissionAdministrator == 0 && (i.AppPermissions&needed) != needed {
			return "บอทต้องมี View Channel, Send Messages และ Send Polls ในห้องนี้", nil
		}
	} else {
		ok, err := u.BotHasChannelPermissions(ctx, s, i.GuildID, botID, channel, needed)
		if err != nil {
			return "", err
		}
		if !ok {
			return "บอทต้องมี View Channel, Send Messages และ Send Polls ในห้องนี้", nil
		}
	}
	message, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{Content: "โหวตโดย <@" + i.Member.User.ID + ">", Poll: poll, AllowedMentions: &discordgo.MessageAllowedMentions{}}, discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("สร้างโหวตแล้ว: https://discord.com/channels/%s/%s/%s", i.GuildID, i.ChannelID, message.ID), nil
}
