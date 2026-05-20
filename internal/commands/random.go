package commands

import (
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
)

func HandleRandom(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	guildID := i.GuildID
	commands, err := cli.ListCommands(guildID)
	if err != nil {
		content := "ランダムコマンドの取得に失敗しました。"
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

	command, ok := pickRandomCommand(commands)
	if !ok {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "コマンドは登録されていません。",
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: command.Response,
		},
	})
}

func pickRandomCommand(commands []client.CommandRecord) (client.CommandRecord, bool) {
	if len(commands) == 0 {
		return client.CommandRecord{}, false
	}
	return commands[rand.IntN(len(commands))], true
}
