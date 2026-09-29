package welcome

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
	"personal-discord-bot/internal/storage"
)

type handler struct{ feature.Dependencies }

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}
	admin := int64(discordgo.PermissionAdministrator)
	m := feature.Module{Name: "welcome", Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "welcome", Description: "Administrator: configure welcome messages", DefaultMemberPermissions: &admin, Options: []*discordgo.ApplicationCommandOption{
		feature.Action("set", "disable", "show"),
		{Type: discordgo.ApplicationCommandOptionChannel, Name: "channel", Description: "Welcome text channel", ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}},
		{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "Template: {user}, {server}; max 1500 characters", MaxLength: 1500},
	}}, Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
		return h.configure(ctx, s, i.GuildID, feature.Options(i))
	}}}}
	if d.Config.WelcomeEnabled {
		m.Intents = discordgo.IntentsGuildMembers
		welcomeGuild := d.Config.GuildID
		if d.Config.MultiGuild {
			welcomeGuild = ""
		}
		m.MemberAdd = Welcome(d.Store, welcomeGuild)
	}
	return m
}
func (h *handler) configure(ctx context.Context, s *discordgo.Session, guild string, o map[string]string) (string, error) {
	switch o["action"] {
	case "show":
		w, err := h.Store.Welcome(ctx, guild)
		return fmt.Sprintf("Members intent enabled: %t\nChannel: %s\nTemplate: %s", h.Config.WelcomeEnabled, w.Channel, w.Message), err
	case "disable":
		return "ปิด welcome แล้ว", h.Store.SetWelcome(ctx, guild, storage.Welcome{})
	case "set":
		if !h.Config.WelcomeEnabled {
			return "ตั้ง WELCOME_ENABLED=true และเปิด Server Members Intent แล้ว restart ก่อน", nil
		}
		if o["channel"] == "" {
			return "โปรดระบุ channel", nil
		}
		c, err := s.Channel(o["channel"], discordgo.WithContext(ctx))
		if err != nil {
			return "", err
		}
		if c.GuildID != guild || c.Type != discordgo.ChannelTypeGuildText {
			return "เลือก text channel ในเซิร์ฟเวอร์นี้", nil
		}
		needed := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages)
		ok, err := featureutil.BotHasChannelPermissions(ctx, s, guild, h.BotID, c, needed)
		if err != nil {
			return "", err
		}
		if !ok {
			return "บอทต้องมี View Channel และ Send Messages ในห้องนี้", nil
		}
		message := o["message"]
		if message == "" {
			message = "ยินดีต้อนรับ {user} เข้าสู่ {server}!"
		}
		return "บันทึก welcome แล้ว", h.Store.SetWelcome(ctx, guild, storage.Welcome{Channel: c.ID, Message: message})
	}
	return "Action ไม่ถูกต้อง", nil
}
