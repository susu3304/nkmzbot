package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/db"
)

func HandleAdd(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB) {
	data := i.ApplicationCommandData()
	guildID := ParseGuildID(i.GuildID)

	options := data.Options
	var name, response string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		} else if opt.Name == "response" {
			response = opt.StringValue()
		}
	}

	err := db.AddCommand(context.Background(), guildID, name, response)
	var content string
	if err != nil {
		content = "追加に失敗しました。"
	} else {
		content = fmt.Sprintf("コマンド '%s' を追加しました。", name)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func HandleRemove(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB) {
	data := i.ApplicationCommandData()
	guildID := ParseGuildID(i.GuildID)

	options := data.Options
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	err := db.RemoveCommand(context.Background(), guildID, name)
	var content string
	if err != nil {
		content = "そのコマンドは存在しません。"
	} else {
		content = fmt.Sprintf("コマンド '%s' を削除しました。", name)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func HandleUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB) {
	data := i.ApplicationCommandData()
	guildID := ParseGuildID(i.GuildID)

	options := data.Options
	var name, response string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		} else if opt.Name == "response" {
			response = opt.StringValue()
		}
	}

	err := db.UpdateCommand(context.Background(), guildID, name, response)
	var content string
	if err != nil {
		content = "そのコマンドは存在しません。"
	} else {
		content = fmt.Sprintf("コマンド '%s' を更新しました。", name)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func HandleAddBulk(s *discordgo.Session, i *discordgo.InteractionCreate, db *db.DB) {
	data := i.ApplicationCommandData()
	guildID := ParseGuildID(i.GuildID)

	options := data.Options
	var commandsText string
	for _, opt := range options {
		if opt.Name == "commands" {
			commandsText = opt.StringValue()
		}
	}

	// Parse the commands text
	commands := parseBulkCommands(commandsText)

	if len(commands) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "有効なコマンドが見つかりませんでした。形式: !コマンド名: 内容",
			},
		})
		return
	}

	// Add commands to database
	successCount := 0
	var errors []string
	for _, cmd := range commands {
		err := db.AddCommand(context.Background(), guildID, cmd.Name, cmd.Response)
		if err != nil {
			errors = append(errors, fmt.Sprintf("'%s': 追加失敗", cmd.Name))
		} else {
			successCount++
		}
	}

	// Build response message
	var content string
	if successCount > 0 && len(errors) == 0 {
		content = fmt.Sprintf("✅ %d件のコマンドを追加しました。", successCount)
	} else if successCount > 0 {
		content = fmt.Sprintf("✅ %d件のコマンドを追加しました。\n❌ エラー: %s", successCount, strings.Join(errors, ", "))
	} else {
		content = fmt.Sprintf("❌ コマンドの追加に失敗しました: %s", strings.Join(errors, ", "))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

// parseBulkCommands parses a multi-line string in the format:
// !cmd1: content1
// !cmd2: content2
// !cmd3: content3
func parseBulkCommands(text string) []struct{ Name, Response string } {
	var commands []struct{ Name, Response string }
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line starts with !
		if !strings.HasPrefix(line, "!") {
			continue
		}

		// Remove the ! prefix
		line = line[1:]

		// Find the first colon to split command name and response
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		name := strings.TrimSpace(line[:colonIndex])
		response := strings.TrimSpace(line[colonIndex+1:])

		if name != "" && response != "" {
			commands = append(commands, struct{ Name, Response string }{
				Name:     name,
				Response: response,
			})
		}
	}

	return commands
}
