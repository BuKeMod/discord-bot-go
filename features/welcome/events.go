package welcome

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/storage"
)

func Welcome(store *storage.Store, guildID string) func(*discordgo.Session, *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
		if e.Member == nil || e.User == nil || e.User.Bot {
			return
		}
		if guildID != "" && e.GuildID != guildID {
			return
		}
		targetGuild := e.GuildID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if store != nil && !store.IsModuleEnabled(ctx, targetGuild, "welcome") {
			return
		}
		w, err := store.Welcome(ctx, targetGuild)
		if err != nil {
			slog.Error("load welcome failed", "error", err)
			return
		}
		if w.Channel == "" {
			return
		}
		g, err := s.State.Guild(targetGuild)
		if err != nil {
			g, _ = s.Guild(targetGuild, discordgo.WithContext(ctx))
		}
		guildName := targetGuild
		if g != nil {
			guildName = g.Name
		}
		message := strings.NewReplacer("{user}", "<@"+e.User.ID+">", "{server}", guildName).Replace(w.Message)
		_, err = s.ChannelMessageSendComplex(w.Channel, &discordgo.MessageSend{Content: message, AllowedMentions: &discordgo.MessageAllowedMentions{Users: []string{e.User.ID}}}, discordgo.WithContext(ctx))
		if err != nil {
			slog.Error("send welcome failed", "error", err)
		}
	}
}
