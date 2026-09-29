package commands

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/features"
	"personal-discord-bot/internal/config"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/storage"
)

func TestModuleInterception(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	guildID := "guild-123"

	// Disable 'fun' module in guild-123
	if err := store.SetModuleEnabled(ctx, guildID, "fun", false); err != nil {
		t.Fatal(err)
	}

	reg, err := features.New(feature.Dependencies{Store: store})
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Registry: reg,
		Store:    store,
		Config:   config.Config{MultiGuild: true},
	}

	var capturedBody string
	s, _ := discordgo.New("Bot test")
	s.Client = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})}

	// 1. Execute 'roll' command (belongs to 'fun') -> should respond with disabled warning
	rollCmd := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    "i1",
			Token: "t1",
			Type:  discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "u1"},
			},
			GuildID: guildID,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "roll",
			},
		},
	}

	h.Handle(s, rollCmd)
	if !strings.Contains(capturedBody, "ซึ่งถูกปิดใช้งานในเซิร์ฟเวอร์นี้โดยผู้ดูแลระบบ") {
		t.Fatalf("expected module disabled message in response, got: %s", capturedBody)
	}

	// 2. Component belonging to party module when party is disabled
	if err := store.SetModuleEnabled(ctx, guildID, "party", false); err != nil {
		t.Fatal(err)
	}
	capturedBody = ""
	partyBtn := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    "i2",
			Token: "t2",
			Type:  discordgo.InteractionMessageComponent,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "u1"},
			},
			GuildID: guildID,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "party_join_1",
			},
		},
	}
	h.Handle(s, partyBtn)
	if !strings.Contains(capturedBody, "ซึ่งถูกปิดใช้งานในเซิร์ฟเวอร์นี้โดยผู้ดูแลระบบ") {
		t.Fatalf("expected party disabled message in response, got: %s", capturedBody)
	}

	// 3. In another guild where 'fun' is not disabled, 'roll' should defer acknowledgement normally
	capturedBody = ""
	otherGuildCmd := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    "i3",
			Token: "t3",
			Type:  discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "u1"},
			},
			GuildID: "guild-456",
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "roll",
			},
		},
	}
	h.Handle(s, otherGuildCmd)
	if strings.Contains(capturedBody, "ซึ่งถูกปิดใช้งาน") {
		t.Fatalf("roll should be enabled in guild-456: %s", capturedBody)
	}
}
