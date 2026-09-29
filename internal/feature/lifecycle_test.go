package feature

import (
	"context"
	"errors"
	"github.com/bwmarrin/discordgo"
	"testing"
)

func TestLifecycleFailureStopsPreviousModules(t *testing.T) {
	var workerContext context.Context
	stops := 0
	r, err := New(Module{Name: "worker", Start: func(ctx context.Context, _ *discordgo.Session) error { workerContext = ctx; return nil }, Stop: func() { stops++ }}, Module{Name: "broken", Start: func(context.Context, *discordgo.Session) error { return errors.New("broken") }})
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Start(context.Background(), nil); err == nil {
		t.Fatal("startup error ignored")
	}
	if workerContext.Err() == nil || stops != 1 {
		t.Fatal("worker not stopped")
	}
	r.Stop()
	if stops != 1 {
		t.Fatal("double stop")
	}
}
