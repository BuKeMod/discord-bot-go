package utility_party

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestParseGameTime(t *testing.T) {
	cases := []struct {
		input       string
		expectEmpty bool
	}{
		{"now", false},
		{"ตอนนี้", false},
		{"30m", false},
		{"21:00", false},
		{"", false},
	}

	for _, c := range cases {
		display, _ := ParseGameTime(c.input)
		if display == "" {
			t.Errorf("ParseGameTime(%q) returned empty display", c.input)
		}
	}
}

func TestPartyStorage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer db.Close()

	repo := &repository{db: db}
	ctx := context.Background()

	if err := repo.migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	now := time.Now().Unix()
	p := &Party{
		GuildID:    "guild_1",
		ChannelID:  "chan_1",
		HostID:     "user_host",
		Game:       "Valorant",
		MaxPlayers: 2,
		TargetTime: "21:00 น.",
		Note:       "Test note",
		CreatedAt:  now,
		ExpiresAt:  now + 3600,
		Status:     "open",
	}

	partyID, err := repo.createParty(ctx, p)
	if err != nil {
		t.Fatalf("createParty failed: %v", err)
	}
	if partyID <= 0 {
		t.Fatalf("expected positive partyID, got %d", partyID)
	}

	// Host should be automatically in main
	members, err := repo.getMembers(ctx, partyID)
	if err != nil {
		t.Fatalf("getMembers failed: %v", err)
	}
	if len(members) != 1 || members[0].UserID != "user_host" || members[0].Role != "main" {
		t.Fatalf("unexpected members: %+v", members)
	}

	// Join user 2 into main -> should become full (MaxPlayers = 2)
	res, err := repo.join(ctx, partyID, "user_2", "main")
	if err != nil {
		t.Fatalf("join user_2 failed: %v", err)
	}
	if !res.IsNowFull || res.ActionTaken != "joined_main" {
		t.Fatalf("expected party to be full, got %+v", res)
	}

	// Join user 3 into main -> should be rejected because main is full
	res3, err := repo.join(ctx, partyID, "user_3", "main")
	if err != nil {
		t.Fatalf("join user_3 failed: %v", err)
	}
	if res3.ActionTaken != "main_full" {
		t.Fatalf("expected main_full, got %s", res3.ActionTaken)
	}

	// Join user 3 into sub -> should succeed
	res3Sub, err := repo.join(ctx, partyID, "user_3", "sub")
	if err != nil {
		t.Fatalf("join sub failed: %v", err)
	}
	if res3Sub.ActionTaken != "joined_sub" {
		t.Fatalf("expected joined_sub, got %s", res3Sub.ActionTaken)
	}

	// User 2 leaves -> User 3 (sub) should be automatically promoted to main!
	leaveRes, err := repo.leave(ctx, partyID, "user_2")
	if err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	if !leaveRes.WasMember {
		t.Fatal("expected WasMember true")
	}
	if leaveRes.PromotedUserID != "user_3" {
		t.Fatalf("expected user_3 promoted, got %s", leaveRes.PromotedUserID)
	}

	// Cancel party
	if err := repo.cancel(ctx, partyID); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	cancelledParty, _ := repo.getParty(ctx, partyID)
	if cancelledParty.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %s", cancelledParty.Status)
	}
}
