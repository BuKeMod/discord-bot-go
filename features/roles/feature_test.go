package roles

import (
	"github.com/bwmarrin/discordgo"
	"testing"
)

func TestSafeRole(t *testing.T) {
	cases := []struct {
		name string
		role *discordgo.Role
		want bool
	}{
		{"cosmetic", &discordgo.Role{ID: "role", Position: 1}, true},
		{"everyone", &discordgo.Role{ID: "guild"}, false},
		{"managed", &discordgo.Role{ID: "role", Managed: true}, false},
		{"administrator", &discordgo.Role{ID: "role", Permissions: discordgo.PermissionAdministrator}, false},
		{"unknown permission", &discordgo.Role{ID: "role", Permissions: 1 << 60}, false},
		{"equal", &discordgo.Role{ID: "role", Position: 5}, false},
		{"higher", &discordgo.Role{ID: "role", Position: 6}, false},
		{"missing", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SafeRole(c.role, "guild", 5); got != c.want {
				t.Fatalf("got %v want %v", got, c.want)
			}
		})
	}
}
