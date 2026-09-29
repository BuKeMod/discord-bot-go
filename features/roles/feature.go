package roles

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log/slog"
	"personal-discord-bot/internal/feature"
)

type handler struct{ feature.Dependencies }

func New(d feature.Dependencies) feature.Module {
	m := feature.Module{Name: "roles"}
	if !d.Config.RolesEnabled {
		return m
	}
	h := &handler{d}
	admin := int64(discordgo.PermissionAdministrator)
	role := func() *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionRole, Name: "role", Description: "Role", Required: true}
	}
	m.Commands = []feature.Command{
		{Definition: &discordgo.ApplicationCommand{Name: "role", Description: "Add or remove an approved role for yourself", Options: []*discordgo.ApplicationCommandOption{feature.Action("add", "remove"), role()}}, Execute: h.change},
		{Definition: &discordgo.ApplicationCommand{Name: "roleconfig", Description: "Administrator: allow or deny a self-service role", DefaultMemberPermissions: &admin, Options: []*discordgo.ApplicationCommandOption{feature.Action("allow", "deny"), role()}}, Execute: h.configure},
	}
	return m
}
func (h *handler) change(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	opts := feature.Options(i)

	if opts["action"] != "add" && opts["action"] != "remove" {
		return "Action ไม่ถูกต้อง", nil
	}
	allowed, err := h.Store.Allowed(ctx, i.GuildID, opts["role"])
	if err != nil {
		return "", err
	}
	if !allowed {
		return "Role นี้ไม่ได้รับอนุญาต", nil
	}
	if err := h.checkRole(ctx, s, i.GuildID, opts["role"]); err != nil {
		return err.Error(), nil
	}
	if opts["action"] == "add" {
		err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, opts["role"], discordgo.WithContext(ctx))
	} else {
		err = s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, opts["role"], discordgo.WithContext(ctx))
	}
	if err != nil {
		return "", err
	}
	slog.Info("self role changed", "guild", i.GuildID, "role", opts["role"], "actor", i.Member.User.ID, "action", opts["action"])
	return "อัปเดต role แล้ว", nil
}
func (h *handler) configure(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	opts := feature.Options(i)
	if opts["action"] == "deny" {
		return "นำ role ออกจากรายการที่อนุญาตแล้ว", h.Store.Allow(ctx, i.GuildID, opts["role"], false)
	}
	if opts["action"] != "allow" {
		return "Action ไม่ถูกต้อง", nil
	}
	if err := h.checkRole(ctx, s, i.GuildID, opts["role"]); err != nil {
		return err.Error(), nil
	}
	slog.Info("role allowlist updated", "guild", i.GuildID, "role", opts["role"], "actor", i.Member.User.ID)
	return "อนุญาต role แล้ว", h.Store.Allow(ctx, i.GuildID, opts["role"], true)

}

// Zero-permission roles fail closed even when Discord introduces new permission bits.
func SafeRole(role *discordgo.Role, guild string, botPosition int) bool {
	return role != nil && role.ID != guild && !role.Managed && role.Permissions == 0 && role.Position < botPosition
}
func (h *handler) checkRole(ctx context.Context, s *discordgo.Session, guild, id string) error {
	roles, err := s.GuildRoles(guild, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("อ่าน role ไม่สำเร็จ")
	}
	bot, err := s.GuildMember(guild, h.BotID, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("อ่านสิทธิ์บอทไม่สำเร็จ")
	}
	pos := 0
	permissions := int64(0)
	var target *discordgo.Role
	for _, r := range roles {
		if r.ID == id {
			target = r
		}
		owned := r.ID == guild
		for _, id := range bot.Roles {
			if id == r.ID {
				owned = true
			}
		}
		if owned {
			permissions |= r.Permissions
			if r.Position > pos {
				pos = r.Position
			}
		}
	}
	if permissions&(discordgo.PermissionManageRoles|discordgo.PermissionAdministrator) == 0 {
		return fmt.Errorf("บอทต้องมี Manage Roles")
	}
	if !SafeRole(target, guild, pos) {
		return fmt.Errorf("ต้องเป็น role ที่ไม่มีสิทธิ์ระดับเซิร์ฟเวอร์ ไม่ใช่ managed/@everyone และอยู่ต่ำกว่า role บอท")
	}
	return nil
}
