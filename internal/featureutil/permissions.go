package featureutil

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// BotHasChannelPermissions checks whether the bot has the required permissions in the specified channel.
// It directly inspects bot roles and channel overwrites, properly recognizing Administrator and avoiding DiscordGo's UserChannelPermissions bug.
func BotHasChannelPermissions(ctx context.Context, s *discordgo.Session, guildID, botID string, ch *discordgo.Channel, needed int64) (bool, error) {
	// 1. Fetch bot member to inspect assigned roles
	bot, err := s.GuildMember(guildID, botID, discordgo.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("อ่านข้อมูลบอทไม่สำเร็จ: %w", err)
	}

	// 2. Fetch guild roles
	roles, err := s.GuildRoles(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("อ่าน role ไม่สำเร็จ: %w", err)
	}

	roleMap := make(map[string]*discordgo.Role, len(roles))
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	// 3. Compute base permissions
	var basePerms int64
	// @everyone role permissions (role ID == guild ID)
	if everyone, ok := roleMap[guildID]; ok {
		basePerms |= everyone.Permissions
	}
	// Add permissions of bot's assigned roles
	for _, rID := range bot.Roles {
		if r, ok := roleMap[rID]; ok {
			basePerms |= r.Permissions
		}
	}

	// Administrator bypasses ALL channel overwrites and grants all permissions!
	if basePerms&discordgo.PermissionAdministrator != 0 {
		return true, nil
	}

	// 4. Apply channel permission overwrites
	if ch != nil {
		// A. Apply @everyone overwrite
		for _, ow := range ch.PermissionOverwrites {
			if ow.ID == guildID {
				basePerms &= ^ow.Deny
				basePerms |= ow.Allow
				break
			}
		}

		// B. Apply role overwrites
		var roleDeny, roleAllow int64
		for _, ow := range ch.PermissionOverwrites {
			if ow.Type == discordgo.PermissionOverwriteTypeRole {
				for _, rID := range bot.Roles {
					if ow.ID == rID {
						roleDeny |= ow.Deny
						roleAllow |= ow.Allow
						break
					}
				}
			}
		}
		basePerms &= ^roleDeny
		basePerms |= roleAllow

		// C. Apply member-specific overwrite for the bot
		for _, ow := range ch.PermissionOverwrites {
			if ow.Type == discordgo.PermissionOverwriteTypeMember && ow.ID == botID {
				basePerms &= ^ow.Deny
				basePerms |= ow.Allow
				break
			}
		}
	}

	return (basePerms & needed) == needed, nil
}
