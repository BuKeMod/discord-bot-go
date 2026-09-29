package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/features"
	"personal-discord-bot/internal/commands"
	"personal-discord-bot/internal/config"
	"personal-discord-bot/internal/feature"
	"personal-discord-bot/internal/storage"
)

// gate stops accepting work before waiting, avoiding Add/Wait races at shutdown.
type gate struct {
	mu      sync.Mutex
	closing bool
	wg      sync.WaitGroup
}

func (g *gate) run(fn func()) {
	g.mu.Lock()
	if g.closing {
		g.mu.Unlock()
		return
	}
	g.wg.Add(1)
	g.mu.Unlock()
	defer g.wg.Done()
	fn()
}
func (g *gate) stop() { g.mu.Lock(); g.closing = true; g.mu.Unlock(); g.wg.Wait() }

func run() error {
	register := flag.Bool("register", false, "Replace this app's guild commands with enabled commands and exit")
	clearGlobal := flag.Bool("clear-global", false, "Delete this app's global commands across all servers and exit (or combine with -register)")
	clearAll := flag.Bool("clear-all", false, "Delete ALL commands (both global and guild) and exit")
	global := flag.Bool("global", false, "With -register: register commands globally across Discord (shows in bot profile 'Commands' section) and clear guild duplicates")
	listCmds := flag.Bool("list-commands", false, "List all active commands on Discord (both global and guild) and exit")
	flag.Parse()
	c, err := config.Load()
	if err != nil {
		return err
	}
	s, err := discordgo.New("Bot " + c.Token)
	if err != nil {
		return err
	}
	s.Client = &http.Client{Timeout: 15 * time.Second}
	s.MaxRestRetries = 1
	s.State.MaxMessageCount = 0
	s.State.TrackMembers = false
	s.State.TrackPresences = false
	s.State.TrackVoice = false
	me, err := s.User("@me")
	if err != nil {
		return err
	}
	slog.Info("authenticated as", "bot_username", me.Username, "bot_id", me.ID)
	if *listCmds {
		return commands.ListCommands(s, me.ID, c.GuildID)
	}
	if *clearAll {
		return commands.ClearAll(s, me.ID, c.GuildID)
	}
	if *clearGlobal && !*register {
		return commands.ClearGlobal(s, me.ID)
	}
	if *register {
		registry, err := features.New(feature.Dependencies{Config: c, BotID: me.ID})
		if err != nil {
			return err
		}
		if *global || c.GlobalCommands {
			err = commands.RegisterGlobal(s, me.ID, c.GuildID, registry)
			if err == nil {
				slog.Info("global commands registered (visible on bot profile)", "count", len(registry.Definitions()))
			}
			return err
		}
		err = commands.Register(s, me.ID, c.GuildID, registry, *clearGlobal)
		if err == nil {
			slog.Info("commands registered", "count", len(registry.Definitions()), "global_commands_cleared", *clearGlobal)
		}
		return err
	}
	store, err := storage.Open(c.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()
	registry, err := features.New(feature.Dependencies{Config: c, Store: store, BotID: me.ID})
	if err != nil {
		return err
	}
	if err = registry.Start(context.Background(), s); err != nil {
		return err
	}
	defer registry.Stop()
	s.Identify.Intents = registry.Intents()
	h := &commands.Handler{Registry: registry, Config: c, Store: store}
	var work gate
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) { work.run(func() { h.Handle(s, i) }) })
	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
		if e == nil || e.Member == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MemberAdd(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
		if e == nil || e.Member == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MemberRemove(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
		if e == nil || e.Member == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MemberUpdate(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
		if e == nil || e.VoiceState == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.VoiceStateUpdate(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.MessageCreate) {
		if e == nil || e.Message == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MessageCreate(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.MessageUpdate) {
		if e == nil || e.Message == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MessageUpdate(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.MessageDelete) {
		if e == nil || e.Message == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.MessageDelete(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildBanAdd) {
		if e == nil || e.User == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.GuildBanAdd(s, e) })
	})
	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildBanRemove) {
		if e == nil || e.User == nil || (!c.MultiGuild && e.GuildID != c.GuildID) {
			return
		}
		work.run(func() { registry.GuildBanRemove(s, e) })
	})

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("gateway ready", "bot_username", r.User.Username, "bot_id", r.User.ID)
		if c.BotStatus != "" {
			if err := s.UpdateListeningStatus(c.BotStatus); err != nil {
				slog.Warn("failed to set bot listening status", "error", err)
			}
		}
	})
	s.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		slog.Warn("gateway disconnected; automatic reconnect enabled")
	})
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	if err = s.Open(); err != nil {
		return err
	}
	slog.Info("bot started", "multi_guild", c.MultiGuild, "guild", c.GuildID, "welcome_enabled", c.WelcomeEnabled)
	<-stop
	slog.Info("shutting down")
	work.stop()
	registry.Stop()
	return s.Close()
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("bot stopped", "error", err)
		os.Exit(1)
	}
}
