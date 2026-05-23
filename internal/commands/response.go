package commands

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
	"github.com/susu3304/nkmzbot/internal/imm"
)

const (
	modalRegisterResponsePrefix   = "reg_resp:"
	modalRunMessageAsIMMPrefix    = "imm_run_msg:"
	modalRegisterMessageIMMPrefix = "imm_reg_msg:"
)

func HandleRegisterAsResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	// Get the message from the interaction
	message := selectedResolvedMessage(data)
	if message == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "メッセージが見つかりませんでした。",
			},
		})
		return
	}

	// Show modal to get command name
	customID := modalRegisterResponsePrefix + message.ID
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    "コマンド名を入力",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "command_name",
							Label:       "コマンド名",
							Style:       discordgo.TextInputShort,
							Placeholder: "例: hello",
							Required:    true,
							MaxLength:   50,
						},
					},
				},
			},
		},
	})

	if err != nil {
		log.Printf("Failed to create modal: %v", err)
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner) {
	data := i.ModalSubmitData()
	switch {
	case strings.HasPrefix(data.CustomID, modalRegisterResponsePrefix):
		handleRegisterAsResponseModalSubmit(s, i, cli, data)
	case strings.HasPrefix(data.CustomID, modalRunMessageAsIMMPrefix):
		HandleRunMessageAsIMMModalSubmit(s, i, runner, data)
	case strings.HasPrefix(data.CustomID, modalRegisterMessageIMMPrefix):
		HandleRegisterMessageAsIMMModalSubmit(s, i, cli, runner, data)
	}
}

func handleRegisterAsResponseModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, data discordgo.ModalSubmitInteractionData) {
	guildID := i.GuildID
	messageID := strings.TrimPrefix(data.CustomID, modalRegisterResponsePrefix)

	commandName := modalInputValue(data, "command_name")
	if commandName == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "コマンド名が入力されていません。",
			},
		})
		return
	}

	// Fetch the original message
	message, err := s.ChannelMessage(i.ChannelID, messageID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "メッセージの取得に失敗しました。",
			},
		})
		return
	}

	// Build response content (message content + attachment URLs)
	responseContent := message.Content
	for _, attachment := range message.Attachments {
		if responseContent != "" {
			responseContent += "\n"
		}
		responseContent += attachment.URL
	}

	// Add command to using API
	err = cli.AddCommand(guildID, commandName, responseContent)
	var content string
	if err != nil {
		if client.IsConnectionError(err) {
			content = apiConnectionErrorMessage
		} else {
			content = "登録に失敗しました。同じ名前のコマンドが既に存在するかもしれません。"
		}
	} else {
		content = "メッセージの内容をコマンド '" + commandName + "' の返答として登録しました！"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func selectedResolvedMessage(data discordgo.ApplicationCommandInteractionData) *discordgo.Message {
	if data.Resolved == nil || len(data.Resolved.Messages) == 0 {
		return nil
	}
	for _, msg := range data.Resolved.Messages {
		return msg
	}
	return nil
}

func modalInputValue(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, component := range data.Components {
		actionRow, ok := asActionsRow(component)
		if !ok {
			continue
		}
		for _, c := range actionRow.Components {
			input, ok := asTextInput(c)
			if ok && input.CustomID == customID {
				return strings.TrimSpace(input.Value)
			}
		}
	}
	return ""
}

func asActionsRow(component discordgo.MessageComponent) (*discordgo.ActionsRow, bool) {
	switch v := component.(type) {
	case *discordgo.ActionsRow:
		return v, true
	case discordgo.ActionsRow:
		return &v, true
	default:
		return nil, false
	}
}

func asTextInput(component discordgo.MessageComponent) (*discordgo.TextInput, bool) {
	switch v := component.(type) {
	case *discordgo.TextInput:
		return v, true
	case discordgo.TextInput:
		return &v, true
	default:
		return nil, false
	}
}
