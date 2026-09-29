package remind

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log/slog"
	"time"
)

func (w *service) start(ctx context.Context, s *discordgo.Session) error {
	if w.repo == nil {
		return fmt.Errorf("remind requires SQLite")
	}
	if err := w.repo.migrate(ctx); err != nil {
		return err
	}
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			w.deliverBatch(ctx, s)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}
func (w *service) stop() {
	if w.done != nil {
		<-w.done
	}
}
func (w *service) deliverBatch(ctx context.Context, s *discordgo.Session) {
	for n := 0; n < 20; n++ {
		if ctx.Err() != nil {
			return
		}
		j, err := w.repo.claim(ctx, w.guild, time.Now().Unix())
		if err != nil {
			slog.Error("reminder claim failed", "error", err)
			return
		}
		if j == nil {
			return
		}
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		channel, sendErr := s.UserChannelCreate(j.User, discordgo.WithContext(sendCtx))
		if sendErr == nil {
			_, sendErr = s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{Content: fmt.Sprintf("⏰ เตือนที่คุณตั้งไว้ <t:%d:F>\n%s", j.Due, j.Message), AllowedMentions: &discordgo.MessageAllowedMentions{}}, discordgo.WithContext(sendCtx))
		}
		cancel()
		if ctx.Err() != nil {
			return
		}
		if sendErr == nil {
			err = w.repo.complete(ctx, j.ID)
		} else {
			var rest *discordgo.RESTError
			permanent := errors.As(sendErr, &rest) && rest.Response != nil && (rest.Response.StatusCode == 403 || rest.Response.StatusCode == 404)
			err = w.repo.retry(ctx, *j, time.Now().Unix(), permanent)
			// Avoid writing private reminder text or REST response bodies into logs.
			slog.Warn("reminder delivery failed", "id", j.ID, "attempt", j.Attempts, "permanent", permanent)
		}
		if err != nil {
			slog.Error("reminder state update failed", "id", j.ID, "error", err)
			return
		}
	}
}
