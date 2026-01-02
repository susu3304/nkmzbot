package bot

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/commands"
)

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("%s is connected!", event.User.Username)

	// Register global commands once on startup
	if err := b.registerGlobalCommands(); err != nil {
		log.Printf("Failed to register global commands: %v", err)
	}
}

func (b *Bot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	log.Printf("Guild available/joined: %s (id=%s)", event.Name, event.ID)
}

func (b *Bot) registerGlobalCommands() error {
	cmds := b.registry.GetDefinitions()
	// Register Global Commands by passing empty string as guildID
	_, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, "", cmds)
	if err != nil {
		return err
	}

	log.Printf("Registered global application commands")
	return nil
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	b.handleTextCommand(s, m)
}

func (b *Bot) handleTextCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.GuildID == "" {
		return
	}
	guildID := commands.ParseGuildID(m.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	commands.ExecuteTextCommand(ctx, s, b.db, guildID, m.ChannelID, m.Content)
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.registry.Handle(s, i)
}
