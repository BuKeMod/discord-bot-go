package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Token, GuildID, DBPath string
	WelcomeEnabled         bool
	RolesEnabled           bool
	GlobalCommands         bool
	MultiGuild             bool
	BotStatus              string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("read .env: %w", err)
	}
	if os.Getenv("DISCORD_TOKEN") == "" {
		_ = godotenv.Load("/etc/discord-bot.env")
	}
	c := Config{Token: strings.TrimSpace(os.Getenv("DISCORD_TOKEN")), GuildID: strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID")), DBPath: os.Getenv("DB_PATH")}
	var err error
	if value := os.Getenv("MULTI_GUILD"); value != "" {
		c.MultiGuild, err = strconv.ParseBool(value)
		if err != nil {
			return c, fmt.Errorf("MULTI_GUILD must be true or false")
		}
	}
	if c.Token == "" {
		return c, fmt.Errorf("DISCORD_TOKEN is required")
	}
	if !c.MultiGuild && !Snowflake(c.GuildID) {
		return c, fmt.Errorf("DISCORD_TOKEN and a valid DISCORD_GUILD_ID are required")
	}
	if c.DBPath == "" {
		c.DBPath = "data/bot.db"
	}
	if value := os.Getenv("WELCOME_ENABLED"); value != "" {
		c.WelcomeEnabled, err = strconv.ParseBool(value)
	}
	if err != nil {
		return c, fmt.Errorf("WELCOME_ENABLED must be true or false")
	}
	if value := os.Getenv("ROLES_ENABLED"); value != "" {
		c.RolesEnabled, err = strconv.ParseBool(value)
		if err != nil {
			return c, fmt.Errorf("ROLES_ENABLED must be true or false")
		}
	}
	if value := os.Getenv("GLOBAL_COMMANDS"); value != "" {
		c.GlobalCommands, err = strconv.ParseBool(value)
		if err != nil {
			return c, fmt.Errorf("GLOBAL_COMMANDS must be true or false")
		}
	} else if c.MultiGuild {
		c.GlobalCommands = true
	}
	c.BotStatus = strings.TrimSpace(os.Getenv("BOT_STATUS"))
	if c.BotStatus == "" {
		c.BotStatus = "/help"
	}
	return c, nil
}

func Snowflake(s string) bool {
	if len(s) < 17 || len(s) > 20 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0
}
