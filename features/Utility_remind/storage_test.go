package remind

import (
	"context"
	"errors"
	"path/filepath"
	"personal-discord-bot/internal/storage"
	"testing"
	"time"
)

func openRepo(t *testing.T, path string) (*repository, *storage.Store) {
	t.Helper()
	s, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	r := &repository{s.SQL()}
	if err = r.migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r, s
}
func TestPersistIsolationLeaseAndRetry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bot.db")
	r, s := openRepo(t, path)
	if err := r.add(ctx, job{ID: "1", Guild: "g", User: "u", Message: "private", Due: 100}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	r, s = openRepo(t, path)
	defer s.Close()
	if list, err := r.list(ctx, "g", "other"); err != nil || len(list) != 0 {
		t.Fatal("cross-user read", err)
	}
	if ok, err := r.cancel(ctx, "g", "other", "1"); err != nil || ok {
		t.Fatal("cross-user cancel", err)
	}
	if j, err := r.claim(ctx, "other-guild", 200); err != nil || j != nil {
		t.Fatal("cross-guild delivery", err)
	}
	if j, err := r.claim(ctx, "g", 99); err != nil || j != nil {
		t.Fatal("early delivery", err)
	}
	j, err := r.claim(ctx, "g", 100)
	if err != nil || j == nil || j.Attempts != 1 {
		t.Fatal(j, err)
	}
	if again, err := r.claim(ctx, "g", 101); err != nil || again != nil {
		t.Fatal("duplicate claim", err)
	}
	if ok, err := r.cancel(ctx, "g", "u", "1"); err != nil || ok {
		t.Fatal("cancelled in-flight message", err)
	}
	j, err = r.claim(ctx, "g", 161)
	if err != nil || j == nil || j.Attempts != 2 {
		t.Fatal("lease recovery", j, err)
	}
	if err = r.retry(ctx, *j, 161, true); err != nil {
		t.Fatal(err)
	}
	list, err := r.list(ctx, "g", "u")
	if err != nil || len(list) != 1 || list[0].Status != "failed" {
		t.Fatal(list, err)
	}
	if ok, err := r.cancel(ctx, "g", "u", "1"); err != nil || !ok {
		t.Fatal("owner cancel failed", err)
	}
}
func TestQuotaAndDue(t *testing.T) {
	r, s := openRepo(t, filepath.Join(t.TempDir(), "bot.db"))
	defer s.Close()
	ctx := context.Background()
	for n := 0; n < 20; n++ {
		if err := r.add(ctx, job{ID: time.Unix(int64(n), 0).String(), Guild: "g", User: "u", Due: 100}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.add(ctx, job{ID: "extra", Guild: "g", User: "u", Due: 100}); !errors.Is(err, errLimit) {
		t.Fatal("quota not enforced", err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if due, err := Due(10, "", now); err != nil || due.Sub(now) != 10*time.Minute {
		t.Fatal(due, err)
	}
	for _, v := range []struct {
		m  int64
		at string
	}{{0, ""}, {-1, ""}, {1, "2026-10-01T20:00:00+07:00"}, {0, "2020-01-01T00:00:00Z"}, {525601, ""}} {
		if _, err := Due(v.m, v.at, now); err == nil {
			t.Fatal("invalid reminder accepted", v)
		}
	}
}
