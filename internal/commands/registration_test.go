package commands

import (
	"context"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"io"
	"net/http"
	"personal-discord-bot/features"
	"personal-discord-bot/internal/config"
	"personal-discord-bot/internal/feature"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRegisterScopes(t *testing.T) {
	for _, clear := range []bool{false, true} {
		s, _ := discordgo.New("Bot test")
		paths := []string{}
		s.Client = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
			paths = append(paths, r.URL.Path)
			if r.Method != "PUT" {
				t.Fatal("expected bulk replacement")
			}
			b, _ := io.ReadAll(r.Body)
			var defs []*discordgo.ApplicationCommand
			if err := json.Unmarshal(b, &defs); err != nil {
				t.Fatal(err)
			}
			for _, def := range defs {
				if def.Name == "role" || def.Name == "roleconfig" {
					t.Fatal("disabled role registered")
				}
			}

			if len(paths) == 2 && string(b) != "[]" {
				t.Fatal("expected global clear")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("[]"))}, nil
		})}
		registry, buildErr := features.New(feature.Dependencies{})
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		if err := Register(s, "app", "guild", registry, clear); err != nil {
			t.Fatal(err)
		}
		want := 1
		if clear {
			want = 2
		}
		if len(paths) != want {
			t.Fatal(paths)
		}
		if !strings.HasSuffix(paths[0], "/applications/app/guilds/guild/commands") {
			t.Fatal(paths)
		}
		if clear && !strings.HasSuffix(paths[1], "/applications/app/commands") {
			t.Fatal(paths)
		}
	}
}
func TestDisabledRolesFailClosed(t *testing.T) {
	registry, err := features.New(feature.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{Registry: registry}
	for _, name := range []string{"role", "roleconfig"} {
		i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionApplicationCommand, Data: discordgo.ApplicationCommandInteractionData{Name: name}}}
		msg, err := h.execute(context.Background(), nil, i)
		if err != nil || msg != "คำสั่งนี้ไม่ได้เปิดใช้งาน" {
			t.Fatal(msg, err)
		}
	}
	enabled, err := features.New(feature.Dependencies{Config: config.Config{RolesEnabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled.Definitions()) != len(registry.Definitions())+2 {
		t.Fatal("unexpected command count")
	}
}

func TestRegisterGlobal(t *testing.T) {
	s, _ := discordgo.New("Bot test")
	paths := []string{}
	s.Client = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("[]"))}, nil
	})}
	registry, err := features.New(feature.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterGlobal(s, "app", "guild", registry); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 calls (global update and guild cleanup), got %d: %v", len(paths), paths)
	}
	if !strings.HasSuffix(paths[0], "/applications/app/commands") {
		t.Fatalf("expected global path, got: %s", paths[0])
	}
	if !strings.HasSuffix(paths[1], "/applications/app/guilds/guild/commands") {
		t.Fatalf("expected guild cleanup path, got: %s", paths[1])
	}
}

func TestClearAllAndList(t *testing.T) {
	s, _ := discordgo.New("Bot test")
	paths := []string{}
	s.Client = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("[]"))}, nil
	})}
	if err := ClearAll(s, "app", "guild"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(paths), paths)
	}
	paths = nil
	if err := ListCommands(s, "app", "guild"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 get calls, got %d: %v", len(paths), paths)
	}
}
