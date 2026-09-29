package commands

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"personal-discord-bot/internal/feature"
)

// Register replaces this application's guild commands with the enabled definitions.
// Global cleanup is explicit because it affects every server using this application.
func Register(s *discordgo.Session, appID, guild string, registry *feature.Registry, clearGlobal bool) error {
	created, err := s.ApplicationCommandBulkOverwrite(appID, guild, registry.Definitions())
	if err != nil {
		return fmt.Errorf("replace guild commands: %w", err)
	}
	if clearGlobal {
		if _, err := s.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{}); err != nil {
			return fmt.Errorf("guild commands updated, but global cleanup failed: %w", err)
		}
	}
	logCommandMentions(created, false)
	return nil
}

// RegisterGlobal replaces this application's global commands across all servers and clears guild-specific commands.
func RegisterGlobal(s *discordgo.Session, appID, guild string, registry *feature.Registry) error {
	created, err := s.ApplicationCommandBulkOverwrite(appID, "", registry.Definitions())
	if err != nil {
		return fmt.Errorf("replace global commands: %w", err)
	}
	if guild != "" {
		if _, err := s.ApplicationCommandBulkOverwrite(appID, guild, []*discordgo.ApplicationCommand{}); err != nil {
			return fmt.Errorf("global commands updated, but guild cleanup failed: %w", err)
		}
	}
	logCommandMentions(created, true)
	return nil
}

// ClearGlobal deletes all global application commands across all servers.
func ClearGlobal(s *discordgo.Session, appID string) error {
	if _, err := s.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{}); err != nil {
		return fmt.Errorf("clear global commands: %w", err)
	}
	slog.Info("all global commands cleared successfully")
	return nil
}

// ClearAll deletes all global and guild-specific application commands.
func ClearAll(s *discordgo.Session, appID, guildID string) error {
	if err := ClearGlobal(s, appID); err != nil {
		return err
	}
	if guildID != "" {
		if _, err := s.ApplicationCommandBulkOverwrite(appID, guildID, []*discordgo.ApplicationCommand{}); err != nil {
			return fmt.Errorf("clear guild commands: %w", err)
		}
		slog.Info("guild commands cleared successfully", "guild", guildID)
	}
	return nil
}

// ListCommands fetches and prints all active global and guild commands from Discord.
func ListCommands(s *discordgo.Session, appID, guildID string) error {
	globalCmds, err := s.ApplicationCommands(appID, "")
	if err != nil {
		return fmt.Errorf("fetch global commands: %w", err)
	}
	fmt.Printf("\n=== Global Commands on Discord (%d) ===\n", len(globalCmds))
	if len(globalCmds) == 0 {
		fmt.Println("  (none)")
	}
	for _, cmd := range globalCmds {
		fmt.Printf("• /%-16s (ID: %s) - %s\n", cmd.Name, cmd.ID, cmd.Description)
	}

	if guildID != "" {
		guildCmds, err := s.ApplicationCommands(appID, guildID)
		if err != nil {
			return fmt.Errorf("fetch guild commands: %w", err)
		}
		fmt.Printf("\n=== Guild Commands on Discord (%d, Guild: %s) ===\n", len(guildCmds), guildID)
		if len(guildCmds) == 0 {
			fmt.Println("  (none)")
		}
		for _, cmd := range guildCmds {
			fmt.Printf("• /%-16s (ID: %s) - %s\n", cmd.Name, cmd.ID, cmd.Description)
		}
	}
	fmt.Println()
	return nil
}

func logCommandMentions(commands []*discordgo.ApplicationCommand, isGlobal bool) {
	scope := "guild"
	if isGlobal {
		scope = "global"
	}
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		mention := fmt.Sprintf("</%s:%s>", cmd.Name, cmd.ID)
		slog.Info("slash command registered",
			"scope", scope,
			"name", cmd.Name,
			"id", cmd.ID,
			"mention", mention,
		)
	}
}
