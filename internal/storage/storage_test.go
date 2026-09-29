package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPersistenceAndIsolation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bot.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	w := Welcome{Channel: "channel", Message: "Welcome {user}"}
	if err = s.SetWelcome(ctx, "a", w); err != nil {
		t.Fatal(err)
	}
	if err = s.Allow(ctx, "a", "role", true); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Welcome(ctx, "a")
	if err != nil || got != w {
		t.Fatalf("%+v %v", got, err)
	}
	yes, err := s.Allowed(ctx, "a", "role")
	if err != nil || !yes {
		t.Fatal("missing persisted role", err)
	}
	yes, err = s.Allowed(ctx, "b", "role")
	if err != nil || yes {
		t.Fatal("cross-guild leakage", err)
	}
	if err = s.Allow(ctx, "a", "role", false); err != nil {
		t.Fatal(err)
	}
	yes, err = s.Allowed(ctx, "a", "role")
	if err != nil || yes {
		t.Fatal("revoke failed", err)
	}
	if err = s.SetWelcome(ctx, "a", Welcome{}); err != nil {
		t.Fatal(err)
	}
	got, err = s.Welcome(ctx, "a")
	if err != nil || got.Channel != "" {
		t.Fatal("disable failed", err)
	}

	// Module toggle tests
	if !s.IsModuleEnabled(ctx, "a", "party") {
		t.Fatal("expected party to be enabled by default")
	}
	if s.IsModuleEnabled(ctx, "a", "roles") {
		t.Fatal("expected roles to be disabled by default")
	}
	if err = s.SetModuleEnabled(ctx, "a", "party", false); err != nil {
		t.Fatal(err)
	}
	if s.IsModuleEnabled(ctx, "a", "party") {
		t.Fatal("expected party to be disabled in guild a")
	}
	if !s.IsModuleEnabled(ctx, "b", "party") {
		t.Fatal("guild b party should remain enabled")
	}
}
