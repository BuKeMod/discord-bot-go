package feature

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// Start is called once before accepting interactions. Factories remain side-effect free.
func (r *Registry) Start(parent context.Context, s *discordgo.Session) error {
	if r.cancel != nil {
		return fmt.Errorf("features already started")
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	for _, m := range r.modules {
		if m.Start != nil {
			if err := m.Start(ctx, s); err != nil {
				r.Stop()
				return fmt.Errorf("start feature %s: %w", m.Name, err)
			}
			if m.Stop != nil {
				r.stops = append(r.stops, m.Stop)
			}
		}
	}
	return nil
}

// Stop cancels workers and waits for them before SQLite is closed; safe to repeat.
func (r *Registry) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	for i := len(r.stops) - 1; i >= 0; i-- {
		r.stops[i]()
	}
	r.stops = nil
}
