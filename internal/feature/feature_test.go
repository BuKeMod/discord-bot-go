package feature

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"testing"
)

func TestAuthorizationAndDuplicateNames(t *testing.T) {
	admin := int64(discordgo.PermissionAdministrator)
	called := false
	command := Command{Definition: &discordgo.ApplicationCommand{Name: "admin", DefaultMemberPermissions: &admin}, Execute: func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (string, error) {
		called = true
		return "ok", nil
	}}
	r, err := New(Module{Name: "a", Commands: []Command{command}})
	if err != nil {
		t.Fatal(err)
	}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionApplicationCommand, Member: &discordgo.Member{}, Data: discordgo.ApplicationCommandInteractionData{Name: "admin"}}}
	if _, err = r.Execute(context.Background(), nil, i); err != nil || called {
		t.Fatal("unauthorized handler called", err)
	}
	i.Member.Permissions = admin
	if _, err = r.Execute(context.Background(), nil, i); err != nil || !called {
		t.Fatal("admin rejected", err)
	}
	if _, err = New(Module{Name: "a", Commands: []Command{command}}, Module{Name: "b", Commands: []Command{command}}); err == nil {
		t.Fatal("duplicate command accepted")
	}
	if _, err = New(Module{Name: "a"}, Module{Name: "a"}); err == nil {
		t.Fatal("duplicate module accepted")
	}
}

func TestZeroPermissionsRequireAdmin(t *testing.T) {
	zero := int64(0)
	d := &discordgo.ApplicationCommand{DefaultMemberPermissions: &zero}
	if CanUse(d, &discordgo.Member{}) {
		t.Fatal("zero-permission command must be admin-only")
	}
	if !CanUse(d, &discordgo.Member{Permissions: discordgo.PermissionAdministrator}) {
		t.Fatal("admin rejected")
	}
}
