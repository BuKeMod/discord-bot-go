package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func BuildEmbed(cmdName string, content string, isError bool) *discordgo.MessageEmbed {
	now := time.Now()
	if isError {
		return &discordgo.MessageEmbed{
			Title:       "❌ เกิดข้อผิดพลาด",
			Description: content,
			Color:       0xED4245, // Red
			Timestamp:   now.Format(time.RFC3339),
		}
	}

	title := "/" + cmdName
	color := 0x5865F2 // Discord Blurple

	switch cmdName {
	case "ping":
		title = "🏓 Pong!"
		color = 0x57F287 // Green
	case "help":
		title = "📖 รายการคำสั่งทั้งหมด (Commands Help)"
		color = 0x5865F2
	case "serverinfo":
		title = "ℹ️ ข้อมูลเซิร์ฟเวอร์ (Server Info)"
		color = 0x5865F2
	case "userinfo":
		title = "👤 ข้อมูลสมาชิก (User Info)"
		color = 0x5865F2
	case "roleinfo":
		title = "🏷️ ข้อมูล Role"
		color = 0x9B59B6
	case "avatar":
		title = "🖼️ รูปโปรไฟล์ (Avatar)"
		color = 0x5865F2
	case "poll":
		title = "📊 โพลสำรวจความคิดเห็น (Poll)"
		color = 0x5865F2
	case "remind":
		title = "⏰ ระบบแจ้งเตือนความจำ (Reminder)"
		color = 0x3498DB
	case "roll":
		title = "🎲 ทอยลูกเต๋า (Roll)"
		color = 0xFEE75C // Gold
	case "coinflip":
		title = "🪙 โยนหัวก้อย (Coinflip)"
		color = 0xFEE75C
	case "choose":
		title = "✨ ผลการสุ่มเลือก (Choose)"
		color = 0xFEE75C
	case "timestamp":
		title = "⏱️ เวลา Discord Timestamp"
		color = 0x3498DB
	case "welcome":
		title = "👋 การตั้งค่าระบบต้อนรับ (Welcome)"
		color = 0x5865F2
	case "role", "roleconfig":
		title = "🎭 ระบบจัดการ Role"
		color = 0x9B59B6
	case "logconfig":
		title = "📋 การตั้งค่าช่องบันทึก Logs"
		color = 0x5865F2
	case "voiceactivity":
		title = "🎙️ สถิติการใช้งานห้องเสียง (Voice Activity)"
		color = 0x2ECC71
	case "voicetop":
		title = "🏆 อันดับการใช้งานห้องเสียง (Voice Leaderboard)"
		color = 0xF1C40F
	case "ban", "unban", "kick", "timeout", "untimeout", "warn", "purge":
		title = "🛡️ จัดการสมาชิกและข้อความ (Moderation)"
		color = 0xE67E22
	case "ticket":
		title = "🎫 ระบบ Ticket ช่วยเหลือ"
		color = 0x5865F2
	case "party":
		title = "🎮 ระบบตั้งตี้เล่นเกม (Party LFG)"
		color = 0x5865F2
	case "status":
		title = "📊 สถานะระบบบอท (Bot Status)"
		color = 0x5865F2
	case "module":
		title = "🎛️ จัดการโมดูล (Module Management)"
		color = 0x5865F2
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: content,
		Color:       color,
		Timestamp:   now.Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("เรียกใช้คำสั่ง /%s", cmdName),
		},
	}
}
