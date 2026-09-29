package admin_module

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/storage"
)

func TestAdminModule(t *testing.T) {
	d := feature.Dependencies{}
	m := New(d)

	if m.Name != "Admin_module" {
		t.Fatalf("expected feature name Admin_module, got %s", m.Name)
	}

	if len(m.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Commands))
	}

	cmd := m.Commands[0]
	if cmd.Definition.Name != "module" {
		t.Fatalf("expected command name module, got %s", cmd.Definition.Name)
	}

	expectedPerm := int64(discordgo.PermissionAdministrator)
	if cmd.Definition.DefaultMemberPermissions == nil || *cmd.Definition.DefaultMemberPermissions != expectedPerm {
		t.Fatalf("expected command to require Administrator permission")
	}

	if m.ComponentHandler == nil {
		t.Fatalf("expected ComponentHandler to be non-nil")
	}

	// Test buildControlPanel
	tmpDir := t.TempDir()
	store, err := storage.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	embed, components, err := buildControlPanel(ctx, store, "guild_test", "Test Server")
	if err != nil {
		t.Fatal(err)
	}

	if embed == nil || len(components) != 2 {
		t.Fatalf("expected embed and 2 component rows, got %v and %d rows", embed, len(components))
	}
}
