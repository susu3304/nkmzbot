package commands

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/db"
)

// ExecuteTextCommand checks if the content is a custom text command (starting with !)
// and executes it if found. Returns true if a command was executed.
// It assumes the command name is the first word after "!" (e.g. "!hello world" -> command "hello").
func ExecuteTextCommand(ctx context.Context, s *discordgo.Session, database *db.DB, guildID int64, channelID string, content string) bool {
	content = strings.TrimSpace(content)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return false
	}

	mainCmd := parts[0]
	if !strings.HasPrefix(mainCmd, "!") || len(mainCmd) <= 1 {
		return false
	}

	cmdName := mainCmd[1:]

	cmd, err := database.GetCommand(ctx, guildID, cmdName)
	if err == nil && cmd != nil {
		s.ChannelMessageSend(channelID, cmd.Response)
		return true
	}

	return false
}
