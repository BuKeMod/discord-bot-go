package remind

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	u "personal-discord-bot/internal/featureutil"
	"strings"
	"time"
)

type service struct {
	repo  *repository
	guild string
	done  chan struct{}
}

func New(d feature.Dependencies) feature.Module {
	workerGuild := d.Config.GuildID
	if d.Config.MultiGuild {
		workerGuild = ""
	}
	worker := &service{guild: workerGuild}
	if d.Store != nil {
		worker.repo = &repository{db: d.Store.SQL()}
	}
	return feature.Module{Name: "Utility_remind", Start: worker.start, Stop: worker.stop, Commands: []feature.Command{{Definition: &discordgo.ApplicationCommand{Name: "remind", Description: "เตือนตัวเองทาง DM พร้อมบันทึกข้าม restart", Options: []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "add", Description: "เพิ่มรายการเตือน", Options: []*discordgo.ApplicationCommandOption{u.TextOption("message", "ข้อความที่จะเตือน", true, 800), u.IntOption("minutes", "เตือนในอีกกี่นาที (1–525600)", 1, 525600), u.TextOption("at", "หรือเวลา เช่น 2026-10-01T20:00:00+07:00", false, 80)}},
		{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "list", Description: "ดูรายการเตือนของตัวเอง"},
		{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "cancel", Description: "ยกเลิกรายการของตัวเอง", Options: []*discordgo.ApplicationCommandOption{u.TextOption("id", "ID จาก /remind list", true, 30)}},
	}}, Execute: worker.execute}}}
}
func Due(minutes int64, at string, now time.Time) (time.Time, error) {
	if (minutes != 0) == (at != "") {
		return time.Time{}, fmt.Errorf("เลือก minutes หรือ at อย่างใดอย่างหนึ่ง")
	}
	if minutes != 0 {
		if minutes < 1 || minutes > 525600 {
			return time.Time{}, fmt.Errorf("minutes ต้องอยู่ระหว่าง 1–525600")
		}
		return now.Add(time.Duration(minutes) * time.Minute), nil
	}
	t, err := u.Time(at)
	if err != nil {
		return t, err
	}
	if !t.After(now) || t.After(now.Add(365*24*time.Hour)) {
		return t, fmt.Errorf("ระบุเวลาในอนาคตภายใน 365 วัน")
	}
	return t, nil
}
func (w *service) execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	opts := i.ApplicationCommandData().Options
	if len(opts) != 1 {
		return "เลือก add, list หรือ cancel", nil
	}
	sub := opts[0]
	values := map[string]string{}
	minutes := int64(0)
	for _, o := range sub.Options {
		if o.Name == "minutes" {
			minutes = o.IntValue()
		} else if v, ok := o.Value.(string); ok {
			values[o.Name] = v
		}
	}
	user := i.Member.User.ID
	switch sub.Name {
	case "add":
		message := strings.TrimSpace(values["message"])
		if message == "" || len([]rune(message)) > 800 {
			return "ข้อความต้องยาว 1–800 ตัวอักษร", nil
		}
		due, err := Due(minutes, values["at"], time.Now())
		if err != nil {
			return err.Error(), nil
		}
		err = w.repo.add(ctx, job{ID: i.ID, Guild: i.GuildID, User: user, Message: message, Due: due.Unix()})
		if errors.Is(err, errLimit) {
			return err.Error(), nil
		}
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("บันทึกแล้ว จะส่ง DM <t:%d:F> (<t:%d:R>)\nID: `%s`\nเปิดรับ DM จากเซิร์ฟเวอร์นี้ด้วย หากส่งไม่ได้ให้ดู /remind list", due.Unix(), due.Unix(), i.ID), nil
	case "list":
		jobs, err := w.repo.list(ctx, i.GuildID, user)
		if err != nil {
			return "", err
		}
		if len(jobs) == 0 {
			return "ไม่มีรายการเตือนที่ค้างอยู่", nil
		}
		lines := []string{}
		for _, j := range jobs {
			status := map[string]string{"pending": "รอส่ง", "processing": "กำลังส่ง", "failed": "ส่งไม่สำเร็จ"}[j.Status]
			lines = append(lines, fmt.Sprintf("`%s` <t:%d:R> %s\n%s", j.ID, j.Due, status, u.Clip(j.Message, 16)))
		}
		return u.Clip(strings.Join(lines, "\n"), 1950), nil
	case "cancel":
		ok, err := w.repo.cancel(ctx, i.GuildID, user, values["id"])
		if err != nil {
			return "", err
		}
		if !ok {
			return "ไม่พบรายการของคุณ หรือรายการกำลังส่ง/ส่งไปแล้ว", nil
		}
		return "ยกเลิกแล้ว", nil
	}
	return "ไม่พบคำสั่งย่อย", nil
}
