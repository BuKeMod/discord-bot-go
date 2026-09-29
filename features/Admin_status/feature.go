package admin_status

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

var startTime = time.Now()

type handler struct {
	feature.Dependencies
}

func New(d feature.Dependencies) feature.Module {
	h := &handler{d}
	adminPerm := int64(discordgo.PermissionAdministrator)

	return feature.Module{
		Name: "Admin_status",
		Start: func(_ context.Context, _ *discordgo.Session) error {
			startTime = time.Now()
			return nil
		},
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:                     "status",
					Description:              "ดูสถานะระบบ การใช้ CPU, RAM และ Uptime ของบอท (สำหรับ Admin)",
					DefaultMemberPermissions: &adminPerm,
				},
				Execute: h.executeStatus,
			},
		},
	}
}

func (h *handler) executeStatus(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	uptime := time.Since(startTime)

	// Memory Stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	gcCount := m.NumGC

	// CPU & Runtime Stats
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()
	goVersion := runtime.Version()
	osArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	userCPUTime := time.Duration(ru.Utime.Sec)*time.Second + time.Duration(ru.Utime.Usec)*time.Microsecond
	sysCPUTime := time.Duration(ru.Stime.Sec)*time.Second + time.Duration(ru.Stime.Usec)*time.Microsecond
	totalCPUTime := userCPUTime + sysCPUTime

	// Gateway Latency
	latency := s.HeartbeatLatency().Round(time.Millisecond)

	// Bot Avatar
	avatarURL := ""
	botUserID := h.BotID
	if s.State != nil && s.State.User != nil {
		avatarURL = s.State.User.AvatarURL("256")
		botUserID = s.State.User.ID
	}

	embed := &discordgo.MessageEmbed{
		Title: "📊 สถานะระบบและประสิทธิภาพบอท (Bot System Status)",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "⏱️ Uptime (ระยะเวลาทำงาน)",
				Value: fmt.Sprintf("เปิดมาแล้ว: **%s**\nเริ่มทำงานเมื่อ: <t:%d:F>\n(<t:%d:R>)",
					formatDuration(uptime), startTime.Unix(), startTime.Unix()),
				Inline: false,
			},
			{
				Name: "🧠 การใช้หน่วยความจำ (RAM)",
				Value: fmt.Sprintf("• ใช้งานอยู่ (Alloc): **%.2f MB**\n• สะสมทั้งหมด (Total): **%.2f MB**\n• ดึงจากระบบ (Sys): **%.2f MB**\n• รอบการเคลียร์ (GC): **%d ครั้ง**",
					allocMB, totalAllocMB, sysMB, gcCount),
				Inline: true,
			},
			{
				Name: "⚡ หน่วยประมวลผล (CPU & OS)",
				Value: fmt.Sprintf("• CPU Cores: **%d คอร์**\n• CPU Time สะสม: **%s**\n• Goroutines: **%d ตัว**\n• ระบบปฏิบัติการ: `%s`",
					numCPU, totalCPUTime.Round(time.Millisecond), numGoroutines, osArch),
				Inline: true,
			},
			{
				Name: "📶 การเชื่อมต่อ & ฐานข้อมูล",
				Value: fmt.Sprintf("• Gateway Latency: **`%s`**\n• Go Runtime: `%s`\n• ฐานข้อมูล: `SQLite (WAL mode)`",
					latency, goVersion),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Bot ID: %s • ระบบทำงานปกติ (Healthy)", botUserID),
		},
	}

	if avatarURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: avatarURL,
		}
	}

	feature.SetEmbed(ctx, embed)
	return "", nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d วัน", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d ชั่วโมง", hours))
	}
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%d นาที", mins))
	}
	parts = append(parts, fmt.Sprintf("%d วินาที", secs))
	return strings.Join(parts, " ")
}
