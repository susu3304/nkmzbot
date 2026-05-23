package commands

import (
	"math/rand/v2"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
)

func HandleRandom(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	guildID := i.GuildID
	tag := randomTagOption(i.ApplicationCommandData().Options)
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

	commands = filterRandomEligibleCommands(commands)
	commands = filterCommandsByTag(commands, tag)
	command, ok := pickRandomCommand(commands)
	if !ok {
		content := "コマンドは登録されていません。"
		if tag != "" {
			content = "指定されたタグのコマンドは登録されていません。"
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
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

func filterRandomEligibleCommands(commands []client.CommandRecord) []client.CommandRecord {
	filtered := make([]client.CommandRecord, 0, len(commands))
	for _, cmd := range commands {
		if strings.EqualFold(strings.TrimSpace(cmd.Kind), "imm") {
			continue
		}
		filtered = append(filtered, cmd)
	}
	return filtered
}

func pickRandomCommand(commands []client.CommandRecord) (client.CommandRecord, bool) {
	if len(commands) == 0 {
		return client.CommandRecord{}, false
	}
	return commands[rand.IntN(len(commands))], true
}

func randomTagOption(options []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, opt := range options {
		if opt.Name == "tag" {
			return strings.TrimSpace(opt.StringValue())
		}
	}
	return ""
}

func filterCommandsByTag(commands []client.CommandRecord, tag string) []client.CommandRecord {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return commands
	}

	filtered := make([]client.CommandRecord, 0, len(commands))
	for _, cmd := range commands {
		for _, cmdTag := range cmd.Tags {
			if strings.EqualFold(strings.TrimSpace(cmdTag), tag) {
				filtered = append(filtered, cmd)
				break
			}
		}
	}
	return filtered
}
