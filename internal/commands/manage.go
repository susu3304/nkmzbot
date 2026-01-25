package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
)

func HandleAdd(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	data := i.ApplicationCommandData()
	guildID := i.GuildID

	options := data.Options
	var name, response string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		} else if opt.Name == "response" {
			response = opt.StringValue()
		}
	}

	err := cli.AddCommand(guildID, name, response)
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

func HandleRemove(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	data := i.ApplicationCommandData()
	guildID := i.GuildID

	options := data.Options
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	err := cli.RemoveCommand(guildID, name)
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

func HandleUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	data := i.ApplicationCommandData()
	guildID := i.GuildID

	options := data.Options
	var name, response string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		} else if opt.Name == "response" {
			response = opt.StringValue()
		}
	}

	err := cli.UpdateCommand(guildID, name, response)
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

func HandleAddBulk(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	data := i.ApplicationCommandData()
	guildID := i.GuildID

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

	// Convert to client bulk input
	var bulkInputs []client.BulkCommandInput
	for _, cmd := range commands {
		bulkInputs = append(bulkInputs, client.BulkCommandInput{
			Name:     cmd.Name,
			Response: cmd.Response,
		})
	}

	// Add commands using API
	err := cli.AddBulkCommands(guildID, bulkInputs)
	var content string
	if err != nil {
		content = fmt.Sprintf("❌ コマンドの追加に失敗しました。")
	} else {
		content = fmt.Sprintf("✅ コマンドを追加しました。")
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

// BulkCommand represents a command to be added in bulk
type BulkCommand struct {
	Name     string
	Response string
}

// parseBulkCommands parses a multi-line string in the format:
// !cmd1: content1
// !cmd2: content2
// !cmd3: content3
func parseBulkCommands(text string) []BulkCommand {
	var commands []BulkCommand
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
			commands = append(commands, BulkCommand{
				Name:     name,
				Response: response,
			})
		}
	}

	return commands
}
