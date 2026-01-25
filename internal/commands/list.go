package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
)

func HandleList(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	guildID := i.GuildID
	commands, err := cli.ListCommands(guildID)
	if err != nil {
		content := "コマンド一覧の取得に失敗しました。"
		if client.IsConnectionError(err) {
			content = apiConnectionErrorMessage
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})
		return
	}
	if len(commands) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "コマンドは登録されていません。",
			},
		})
		return
	}

	// Build command list
	var entries []string
	for _, cmd := range commands {
		entries = append(entries, fmt.Sprintf("!%s: %s", cmd.Name, cmd.Response))
	}

	// Split into 2000 character chunks
	var buffer strings.Builder
	for _, entry := range entries {
		if buffer.Len()+len(entry)+1 > 2000 {
			s.ChannelMessageSend(i.ChannelID, buffer.String())
			buffer.Reset()
		}
		if buffer.Len() > 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(entry)
	}

	if buffer.Len() > 0 {
		s.ChannelMessageSend(i.ChannelID, buffer.String())
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "コマンド一覧を送信しました。",
		},
	})
}
