package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/db"
)

type AddCommand struct {
	DB *db.DB
}

func (c *AddCommand) Def() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:         "add",
		Description:  "新しいコマンドを追加します",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "コマンド名",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "response",
				Description: "返答内容",
				Required:    true,
			},
		},
	}
}

func (c *AddCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	err := c.DB.AddCommand(context.Background(), guildID, name, response)
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

type RemoveCommand struct {
	DB *db.DB
}

func (c *RemoveCommand) Def() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:         "remove",
		Description:  "コマンドを削除します",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "削除するコマンド名",
				Required:    true,
			},
		},
	}
}

func (c *RemoveCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	guildID := ParseGuildID(i.GuildID)

	options := data.Options
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	err := c.DB.RemoveCommand(context.Background(), guildID, name)
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

type UpdateCommand struct {
	DB *db.DB
}

func (c *UpdateCommand) Def() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:         "update",
		Description:  "コマンドを更新します",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "更新するコマンド名",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "response",
				Description: "新しい返答内容",
				Required:    true,
			},
		},
	}
}

func (c *UpdateCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	err := c.DB.UpdateCommand(context.Background(), guildID, name, response)
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
