package utility_party

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

type worker struct {
	repo *repository
	done chan struct{}
}

func (w *worker) start(ctx context.Context, s *discordgo.Session) error {
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.checkExpired(ctx, s)
			}
		}
	}()
	return nil
}

func (w *worker) stop() {
	if w.done != nil {
		<-w.done
	}
}

func (w *worker) checkExpired(ctx context.Context, s *discordgo.Session) {
	now := time.Now().Unix()
	expired, err := w.repo.getExpiredParties(ctx, now)
	if err != nil {
		slog.Error("party worker fetch expired failed", "error", err)
		return
	}

	for _, p := range expired {
		if ctx.Err() != nil {
			return
		}

		_ = w.repo.markExpired(ctx, p.ID)
		p.Status = "expired"

		members, err := w.repo.getMembers(ctx, p.ID)
		if err != nil {
			continue
		}

		embed := buildPartyEmbed(&p, members)
		components := buildPartyComponents(p.ID, true) // Disabled buttons

		if p.ChannelID != "" && p.MessageID != "" {
			embeds := []*discordgo.MessageEmbed{embed}
			_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    p.ChannelID,
				ID:         p.MessageID,
				Embeds:     &embeds,
				Components: &components,
			})

			mainCount := 0
			for _, m := range members {
				if m.Role == "main" {
					mainCount++
				}
			}

			_, _ = s.ChannelMessageSend(p.ChannelID, fmt.Sprintf("⏰ ตี้ **%s** ของ <@%s> หมดเวลารับสมัครแล้ว (รวบรวมได้ %d/%d คน)", p.Game, p.HostID, mainCount, p.MaxPlayers))
		}
	}
}
