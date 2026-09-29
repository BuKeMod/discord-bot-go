package voice_activity

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/featureutil"
)

type activeUser struct {
	guildID  string
	userID   string
	joinedAt time.Time
}

type module struct {
	feature.Dependencies
	mu     sync.Mutex
	active map[string]activeUser // "guildID:userID" -> activeUser
}

func New(d feature.Dependencies) feature.Module {
	m := &module{
		Dependencies: d,
		active:       make(map[string]activeUser),
	}

	return feature.Module{
		Name:    "Voice_activity",
		Intents: discordgo.IntentsGuildVoiceStates,
		Start: func(ctx context.Context, s *discordgo.Session) error {
			if m.Store != nil {
				return m.migrate(ctx)
			}
			return nil
		},
		Stop: func() {
			m.flushAll()
		},
		VoiceStateUpdate: m.onVoiceStateUpdate,
		Commands: []feature.Command{
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "voiceactivity",
					Description: "ดูสถิติเวลาที่อยู่ในห้องเสียงของตัวเองหรือสมาชิกคนอื่น",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "เลือกสมาชิกที่ต้องการดูสถิติ (ค่าเริ่มต้น: ตัวเอง)",
							Required:    false,
						},
					},
				},
				Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
					return m.showActivity(ctx, s, i)
				},
			},
			{
				Definition: &discordgo.ApplicationCommand{
					Name:        "voicetop",
					Description: "แสดง 10 อันดับสมาชิกที่อยู่ในห้องเสียงนานที่สุดในเซิร์ฟเวอร์",
				},
				Execute: func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
					return m.showLeaderboard(ctx, i.GuildID)
				},
			},
		},
	}
}

func (m *module) migrate(ctx context.Context) error {
	_, err := m.Store.SQL().ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS voice_activity (
    guild TEXT NOT NULL,
    user TEXT NOT NULL,
    total_seconds INTEGER NOT NULL DEFAULT 0,
    last_active INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(guild, user)
);
CREATE INDEX IF NOT EXISTS idx_voice_activity_top ON voice_activity(guild, total_seconds DESC);`)
	return err
}

func (m *module) onVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	if e == nil || e.VoiceState == nil || m.Store == nil {
		return
	}
	guildID := e.GuildID
	if guildID == "" {
		guildID = m.Config.GuildID
	}
	if !m.Store.IsModuleEnabled(context.Background(), guildID, "voice") {
		return
	}
	userID := e.UserID
	newChannelID := e.ChannelID

	key := guildID + ":" + userID

	m.mu.Lock()
	old, hadOld := m.active[key]
	now := time.Now()

	if !hadOld && newChannelID != "" {
		// Connected
		m.active[key] = activeUser{guildID: guildID, userID: userID, joinedAt: now}
		m.mu.Unlock()
		return
	}

	if hadOld && newChannelID == "" {
		// Disconnected
		delete(m.active, key)
		m.mu.Unlock()
		elapsed := int64(now.Sub(old.joinedAt).Seconds())
		if elapsed > 0 {
			m.saveDuration(guildID, userID, elapsed, now.Unix())
		}
		return
	}

	if hadOld && newChannelID != "" {
		// Moved or updated
		elapsed := int64(now.Sub(old.joinedAt).Seconds())
		m.active[key] = activeUser{guildID: guildID, userID: userID, joinedAt: now}
		m.mu.Unlock()
		if elapsed > 0 {
			m.saveDuration(guildID, userID, elapsed, now.Unix())
		}
		return
	}

	m.mu.Unlock()
}

func (m *module) saveDuration(guildID, userID string, seconds, nowUnix int64) {
	if m.Store == nil || seconds <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = m.Store.SQL().ExecContext(ctx, `
INSERT INTO voice_activity (guild, user, total_seconds, last_active)
VALUES (?, ?, ?, ?)
ON CONFLICT(guild, user) DO UPDATE SET
    total_seconds = total_seconds + excluded.total_seconds,
    last_active = excluded.last_active;`, guildID, userID, seconds, nowUnix)
}

func (m *module) flushAll() {
	if m.Store == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, sess := range m.active {
		elapsed := int64(now.Sub(sess.joinedAt).Seconds())
		if elapsed > 0 {
			m.saveDuration(sess.guildID, sess.userID, elapsed, now.Unix())
		}
	}
	m.active = make(map[string]activeUser)
}

func (m *module) showActivity(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	targetID := featureutil.UserID(i)
	guildID := i.GuildID

	var dbSeconds, lastActive int64
	err := m.Store.SQL().QueryRowContext(ctx, "SELECT total_seconds, last_active FROM voice_activity WHERE guild=? AND user=?", guildID, targetID).Scan(&dbSeconds, &lastActive)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	m.mu.Lock()
	activeSess, isCurrentlyInVoice := m.active[guildID+":"+targetID]
	var currentSessionDuration time.Duration
	if isCurrentlyInVoice {
		currentSessionDuration = time.Since(activeSess.joinedAt)
		dbSeconds += int64(currentSessionDuration.Seconds())
	}
	m.mu.Unlock()

	totalText := formatSeconds(dbSeconds)

	statusText := ""
	if isCurrentlyInVoice {
		statusText = fmt.Sprintf("🟢 **กำลังอยู่ในห้องเสียง** (อยู่มาแล้ว %s)", formatSeconds(int64(currentSessionDuration.Seconds())))
	} else if lastActive > 0 {
		statusText = fmt.Sprintf("⚪ **ไม่ได้อยู่ในห้องเสียง** (ออนไลน์ล่าสุด <t:%d:R>)", lastActive)
	} else {
		statusText = "⚪ **ยังไม่เคยมีบันทึกการเข้าห้องเสียง**"
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎙️ สถิติการใช้งานห้องเสียง (Voice Activity)",
		Color: 0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "👤 สมาชิก", Value: fmt.Sprintf("<@%s>", targetID), Inline: true},
			{Name: "⏱️ เวลาสะสมทั้งหมด", Value: totalText, Inline: true},
			{Name: "📶 สถานะปัจจุบัน", Value: statusText, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("User ID: %s", targetID),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	targetUser, err := s.User(targetID, discordgo.WithContext(ctx))
	if err == nil && targetUser != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL("256"),
		}
	}

	feature.SetEmbed(ctx, embed)
	return "", nil
}

func (m *module) showLeaderboard(ctx context.Context, guildID string) (string, error) {
	rows, err := m.Store.SQL().QueryContext(ctx, "SELECT user, total_seconds FROM voice_activity WHERE guild=? ORDER BY total_seconds DESC LIMIT 10", guildID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type entry struct {
		user    string
		seconds int64
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.user, &e.seconds); err != nil {
			return "", err
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		return "📊 ยังไม่มีข้อมูลการใช้งานห้องเสียงในเซิร์ฟเวอร์นี้", nil
	}

	medals := []string{"🥇", "🥈", "🥉"}
	result := ""
	for idx, e := range entries {
		badge := fmt.Sprintf("`#%d`", idx+1)
		if idx < len(medals) {
			badge = medals[idx]
		}
		result += fmt.Sprintf("%s <@%s> — **%s**\n", badge, e.user, formatSeconds(e.seconds))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 10 อันดับสมาชิกที่อยู่ในห้องเสียงนานที่สุด",
		Description: result,
		Color:       0xF1C40F,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Voice Leaderboard",
		},
	}
	feature.SetEmbed(ctx, embed)
	return "", nil
}


func formatSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0 วินาที"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	res := ""
	if days > 0 {
		res += fmt.Sprintf("%d วัน ", days)
	}
	if hours > 0 || days > 0 {
		res += fmt.Sprintf("%d ชั่วโมง ", hours)
	}
	if minutes > 0 || hours > 0 || days > 0 {
		res += fmt.Sprintf("%d นาที ", minutes)
	}
	res += fmt.Sprintf("%d วินาที", secs)
	return res
}
