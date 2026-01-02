package bot

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/commands"
)

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("%s is connected!", event.User.Username)

	// Register commands for all guilds
	for _, guild := range event.Guilds {
		if err := b.registerGuildCommands(guild.ID); err != nil {
			log.Printf("Failed to register commands for guild %s: %v", guild.ID, err)
		}
	}
}

func (b *Bot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	log.Printf("Guild available/joined: %s (id=%s) — ensuring commands", event.Name, event.ID)
	if err := b.registerGuildCommands(event.ID); err != nil {
		log.Printf("Failed to register commands for guild %s: %v", event.ID, err)
	}
}

func (b *Bot) registerGuildCommands(guildID string) error {
	cmds := b.registry.GetDefinitions()
	// Delete existing commands and register new ones
	_, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, guildID, cmds)
	if err != nil {
		return err
	}

	log.Printf("Registered application commands for guild %s", guildID)
	return nil
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	content := strings.TrimSpace(m.Content)
	if strings.HasPrefix(content, "!") && len(content) > 1 {
		cmdName := content[1:]
		if m.GuildID != "" {
			guildID := commands.ParseGuildID(m.GuildID)
			cmd, err := b.db.GetCommand(context.Background(), guildID, cmdName)
			if err == nil && cmd != nil {
				s.ChannelMessageSend(m.ChannelID, cmd.Response)
			}
		}
	}
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.registry.Handle(s, i)
}
