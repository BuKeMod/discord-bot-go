package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	t.Setenv("DISCORD_GUILD_ID", "123456789012345678")
	t.Setenv("DB_PATH", "test.db")
	t.Setenv("WELCOME_ENABLED", "false")
	c, err := Load()
	if err != nil || c.WelcomeEnabled {
		t.Fatal(c, err)
	}
	t.Setenv("WELCOME_ENABLED", "invalid")
	if _, err = Load(); err == nil {
		t.Fatal("invalid boolean accepted")
	}
	t.Setenv("WELCOME_ENABLED", "true")
	c, err = Load()
	if err != nil || !c.WelcomeEnabled {
		t.Fatal(c, err)
	}
	t.Setenv("DISCORD_GUILD_ID", "not-an-id")
	if _, err = Load(); err == nil {
		t.Fatal("invalid guild accepted")
	}
	t.Setenv("DISCORD_GUILD_ID", "123456789012345678")
	t.Setenv("DISCORD_TOKEN", "")
	if _, err = Load(); err == nil {
		t.Fatal("missing token accepted")
	}
	t.Setenv("DISCORD_TOKEN", "test-token")
	t.Setenv("GLOBAL_COMMANDS", "true")
	t.Setenv("BOT_STATUS", "Playing game")
	c, err = Load()
	if err != nil || !c.GlobalCommands || c.BotStatus != "Playing game" {
		t.Fatalf("unexpected global/status: %+v %v", c, err)
	}

	t.Setenv("MULTI_GUILD", "true")
	t.Setenv("DISCORD_GUILD_ID", "")
	c, err = Load()
	if err != nil || !c.MultiGuild || !c.GlobalCommands {
		t.Fatalf("expected multi-guild to succeed without guild ID, got: %+v %v", c, err)
	}
}
