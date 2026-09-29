package feature

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/config"
	"personal-discord-bot/internal/storage"
	"sort"
)

type Dependencies struct {
	Config config.Config
	Store  *storage.Store
	BotID  string
}
type Execute func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (string, error)
type Command struct {
	Definition *discordgo.ApplicationCommand
	Execute    Execute
}
type Module struct {
	Start            func(context.Context, *discordgo.Session) error
	Stop             func()
	Name             string
	Commands         []Command
	Intents          discordgo.Intent
	MemberAdd        func(*discordgo.Session, *discordgo.GuildMemberAdd)
	MemberRemove     func(*discordgo.Session, *discordgo.GuildMemberRemove)
	MemberUpdate     func(*discordgo.Session, *discordgo.GuildMemberUpdate)
	VoiceStateUpdate func(*discordgo.Session, *discordgo.VoiceStateUpdate)
	MessageCreate    func(*discordgo.Session, *discordgo.MessageCreate)
	MessageUpdate    func(*discordgo.Session, *discordgo.MessageUpdate)
	MessageDelete    func(*discordgo.Session, *discordgo.MessageDelete)
	GuildBanAdd      func(*discordgo.Session, *discordgo.GuildBanAdd)
	GuildBanRemove   func(*discordgo.Session, *discordgo.GuildBanRemove)
	ComponentHandler func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (bool, error)
}
type Registry struct {
	cancel   context.CancelFunc
	stops    []func()
	commands map[string]Command
	modules  []Module
}

func New(modules ...Module) (*Registry, error) {
	r := &Registry{commands: map[string]Command{}, modules: modules}
	names := map[string]bool{}
	for _, m := range modules {
		if m.Name == "" || names[m.Name] {
			return nil, fmt.Errorf("empty or duplicate feature: %q", m.Name)
		}
		names[m.Name] = true
		for _, c := range m.Commands {
			if c.Definition == nil || c.Definition.Name == "" || c.Execute == nil {
				return nil, fmt.Errorf("invalid command in feature %s", m.Name)
			}
			if _, exists := r.commands[c.Definition.Name]; exists {
				return nil, fmt.Errorf("duplicate command: %s", c.Definition.Name)
			}
			r.commands[c.Definition.Name] = c
		}
	}
	return r, nil
}
func (r *Registry) Definitions() []*discordgo.ApplicationCommand {
	defs := make([]*discordgo.ApplicationCommand, 0, len(r.commands))
	for _, c := range r.commands {
		defs = append(defs, c.Definition)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}
func (r *Registry) Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	c, ok := r.commands[i.ApplicationCommandData().Name]
	if !ok {
		return "คำสั่งนี้ไม่ได้เปิดใช้งาน", nil
	}
	if !CanUse(c.Definition, i.Member) {
		return "คุณไม่มีสิทธิ์ใช้คำสั่งนี้", nil
	}

	return c.Execute(ctx, s, i)
}
func (r *Registry) Intents() discordgo.Intent {
	intents := discordgo.IntentsGuilds
	for _, m := range r.modules {
		intents |= m.Intents
	}
	return intents
}
func (r *Registry) MemberAdd(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	for _, m := range r.modules {
		if m.MemberAdd != nil {
			m.MemberAdd(s, e)
		}
	}
}
func (r *Registry) MemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	for _, m := range r.modules {
		if m.MemberRemove != nil {
			m.MemberRemove(s, e)
		}
	}
}
func (r *Registry) MemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	for _, m := range r.modules {
		if m.MemberUpdate != nil {
			m.MemberUpdate(s, e)
		}
	}
}
func (r *Registry) VoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	for _, m := range r.modules {
		if m.VoiceStateUpdate != nil {
			m.VoiceStateUpdate(s, e)
		}
	}
}
func (r *Registry) MessageCreate(s *discordgo.Session, e *discordgo.MessageCreate) {
	for _, m := range r.modules {
		if m.MessageCreate != nil {
			m.MessageCreate(s, e)
		}
	}
}
func (r *Registry) MessageUpdate(s *discordgo.Session, e *discordgo.MessageUpdate) {
	for _, m := range r.modules {
		if m.MessageUpdate != nil {
			m.MessageUpdate(s, e)
		}
	}
}
func (r *Registry) MessageDelete(s *discordgo.Session, e *discordgo.MessageDelete) {
	for _, m := range r.modules {
		if m.MessageDelete != nil {
			m.MessageDelete(s, e)
		}
	}
}
func (r *Registry) GuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	for _, m := range r.modules {
		if m.GuildBanAdd != nil {
			m.GuildBanAdd(s, e)
		}
	}
}
func (r *Registry) GuildBanRemove(s *discordgo.Session, e *discordgo.GuildBanRemove) {
	for _, m := range r.modules {
		if m.GuildBanRemove != nil {
			m.GuildBanRemove(s, e)
		}
	}
}
func (r *Registry) ExecuteComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	for _, m := range r.modules {
		if m.ComponentHandler != nil {
			handled, err := m.ComponentHandler(ctx, s, i)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
		}
	}
	return nil
}
func Options(i *discordgo.InteractionCreate) map[string]string {
	result := map[string]string{}
	for _, o := range i.ApplicationCommandData().Options {
		if v, ok := o.Value.(string); ok {
			result[o.Name] = v
		}
	}
	return result
}
func Action(values ...string) *discordgo.ApplicationCommandOption {
	o := &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "Action", Required: true}
	for _, v := range values {
		o.Choices = append(o.Choices, &discordgo.ApplicationCommandOptionChoice{Name: v, Value: v})
	}
	return o
}

// Zero default permissions means administrator-only, matching Discord's defaults.
func CanUse(d *discordgo.ApplicationCommand, m *discordgo.Member) bool {
	if d.DefaultMemberPermissions == nil {
		return true
	}
	if m == nil {
		return false
	}
	if m.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	required := *d.DefaultMemberPermissions
	return required != 0 && m.Permissions&required == required
}
