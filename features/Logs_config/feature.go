package logs_config

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
)

type handler struct{ feature.Dependencies }

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}
	admin := int64(discordgo.PermissionAdministrator)

	return feature.Module{
		Name: "Logs_config",
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "logconfig",
					Description:              "Administrator: ตั้งค่าช่องสำหรับส่งบันทึก Logs (Voice, Message, Member)",
					DefaultMemberPermissions: &admin,
					Options: []*discordgo.ApplicationCommandOption{
						feature.Action("set", "disable", "show"),
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "type",
							Description: "ประเภท Log ที่ต้องการตั้งค่า",
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "all (ทุกประเภทในห้องเดียว)", Value: "all"},
								{Name: "mod (บันทึก ban/kick/timeout/warn)", Value: "mod"},
								{Name: "ticket (บันทึกประวัติ ticket)", Value: "ticket"},
								{Name: "voice (เข้า/ออก/ย้ายห้องเสียง)", Value: "voice"},
								{Name: "message (ลบ/แก้ไขข้อความ)", Value: "message"},
								{Name: "member (สมาชิก เข้า/ออก/อัปเดต)", Value: "member"},
							},
						},
						{
							Type:         discordgo.ApplicationCommandOptionChannel,
							Name:         "channel",
							Description:  "ห้องข้อความสำหรับส่ง Log",
							Required:     false,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
						},
					},
				},
				Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
					return h.configure(ctx, s, i.GuildID, i)
				},
			},
		},
	}
}

func (h *handler) configure(ctx context.Context, s *discordgo.Session, guild string, i *discordgo.InteractionCreate) (string, error) {
	action := featureutil.String(i, "action", "show")
	logType := featureutil.String(i, "type", "all")

	switch action {
	case "show":
		channels, err := h.Store.AllLogChannels(ctx, guild)
		if err != nil {
			return "", err
		}

		formatCh := func(t string) string {
			if chID, ok := channels[t]; ok && chID != "" {
				return fmt.Sprintf("<#%s>", chID)
			}
			return "`ไม่ได้ตั้งค่า (ปิดใช้งาน)`"
		}

		return fmt.Sprintf("📋 **การตั้งค่าช่องบันทึก Log ปัจจุบัน**:\n• 🛡️ **Mod Log**: %s\n• 🎫 **Ticket Log**: %s\n• 🔊 **Voice Log**: %s\n• 💬 **Message Log**: %s\n• 👤 **Member Log**: %s",
			formatCh("mod"), formatCh("ticket"), formatCh("voice"), formatCh("message"), formatCh("member")), nil

	case "disable":
		if err := h.Store.DeleteLogChannel(ctx, guild, logType); err != nil {
			return "", err
		}
		if logType == "all" {
			return "ปิดการส่ง Log ทุกประเภทเรียบร้อยแล้ว", nil
		}
		return fmt.Sprintf("ปิดการส่ง Log สำหรับ **%s** เรียบร้อยแล้ว", logType), nil

	case "set":
		chID := featureutil.String(i, "channel", "")
		if chID == "" {
			return "โปรดระบุห้อง (channel) ที่ต้องการให้ส่ง Log", nil
		}
		ch, err := s.Channel(chID, discordgo.WithContext(ctx))
		if err != nil {
			return "", err
		}
		if ch.GuildID != guild || ch.Type != discordgo.ChannelTypeGuildText {
			return "โปรดเลือก text channel ภายในเซิร์ฟเวอร์นี้", nil
		}

		needed := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks)
		ok, err := featureutil.BotHasChannelPermissions(ctx, s, guild, h.BotID, ch, needed)
		if err != nil {
			return "", err
		}
		if !ok {
			return "บอทต้องการสิทธิ์ View Channel, Send Messages และ Embed Links ในห้องนั้นเพื่อส่ง Log", nil
		}

		if err := h.Store.SetLogChannel(ctx, guild, logType, ch.ID); err != nil {
			return "", err
		}
		if logType == "all" {
			return fmt.Sprintf("ตั้งค่าช่องบันทึก Log ทุกประเภทเป็น <#%s> สำเร็จ", ch.ID), nil
		}
		return fmt.Sprintf("ตั้งค่าช่องบันทึก Log สำหรับ **%s** เป็น <#%s> สำเร็จ", logType, ch.ID), nil

	default:
		return "คำสั่งไม่ถูกต้อง", nil
	}
}
