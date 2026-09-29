package admin_status

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

func TestFormatDuration(t *testing.T) {
	d1 := 45 * time.Second
	if res := formatDuration(d1); res != "45 วินาที" {
		t.Errorf("expected '45 วินาที', got %q", res)
	}

	d2 := 2*time.Hour + 30*time.Minute + 15*time.Second
	if res := formatDuration(d2); res != "2 ชั่วโมง 30 นาที 15 วินาที" {
		t.Errorf("expected '2 ชั่วโมง 30 นาที 15 วินาที', got %q", res)
	}

	d3 := 3*24*time.Hour + 4*time.Hour + 5*time.Minute + 6*time.Second
	if res := formatDuration(d3); res != "3 วัน 4 ชั่วโมง 5 นาที 6 วินาที" {
		t.Errorf("expected '3 วัน 4 ชั่วโมง 5 นาที 6 วินาที', got %q", res)
	}
}

func TestAdminStatusModule(t *testing.T) {
	m := New(feature.Dependencies{BotID: "bot_123"})
	if m.Name != "Admin_status" {
		t.Fatalf("expected name Admin_status, got %s", m.Name)
	}
	if len(m.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Commands))
	}
	cmd := m.Commands[0]
	if cmd.Definition.Name != "status" {
		t.Fatalf("expected command status, got %s", cmd.Definition.Name)
	}
	if cmd.Definition.DefaultMemberPermissions == nil {
		t.Fatal("expected DefaultMemberPermissions to be set")
	}
	expectedPerm := int64(discordgo.PermissionAdministrator)
	if *cmd.Definition.DefaultMemberPermissions != expectedPerm {
		t.Fatalf("expected permission %d, got %d", expectedPerm, *cmd.Definition.DefaultMemberPermissions)
	}
}
