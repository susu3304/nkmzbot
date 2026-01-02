package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// Command defines the interface that all application commands must implement.
type Command interface {
	// Def returns the Discord application command definition.
	Def() *discordgo.ApplicationCommand
	// Handler is the function called when the command is invoked.
	Handler(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// AutocompleteCommand is an optional interface for commands that support autocomplete.
type AutocompleteCommand interface {
	Command
	// AutocompleteHandler is called for autocomplete interactions.
	AutocompleteHandler(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// ModalHandlerCommand is an optional interface for commands that handle modal submissions.
type ModalHandlerCommand interface {
	Command
	// ModalID returns the prefix or exact ID of the modal this command handles.
	ModalID() string
	// ModalHandler is called when a modal with the matching ID is submitted.
	ModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// Registry manages the registration and dispatching of commands.
type Registry struct {
	commands      map[string]Command
	modalHandlers map[string]Command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands:      make(map[string]Command),
		modalHandlers: make(map[string]Command),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd Command) {
	def := cmd.Def()
	r.commands[def.Name] = cmd
	log.Printf("Registered command: %s", def.Name)

	if mh, ok := cmd.(ModalHandlerCommand); ok {
		r.modalHandlers[mh.ModalID()] = cmd
		log.Printf("Registered modal handler for: %s", mh.ModalID())
	}
}

// GetDefinitions returns a list of all registered command definitions.
func (r *Registry) GetDefinitions() []*discordgo.ApplicationCommand {
	defs := make([]*discordgo.ApplicationCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		defs = append(defs, cmd.Def())
	}
	return defs
}

// Handle dispatches an interaction to the appropriate command handler.
func (r *Registry) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		name := i.ApplicationCommandData().Name
		if cmd, ok := r.commands[name]; ok {
			cmd.Handler(s, i)
		} else {
			log.Printf("Unknown command: %s", name)
		}
	} else if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		name := i.ApplicationCommandData().Name
		if cmd, ok := r.commands[name]; ok {
			if ac, ok := cmd.(AutocompleteCommand); ok {
				ac.AutocompleteHandler(s, i)
			}
		}
	} else if i.Type == discordgo.InteractionModalSubmit {
		data := i.ModalSubmitData()
		// Simple prefix matching for modal IDs
		for id, cmd := range r.modalHandlers {
			if len(data.CustomID) >= len(id) && data.CustomID[:len(id)] == id {
				if mh, ok := cmd.(ModalHandlerCommand); ok {
					mh.ModalHandler(s, i)
					return
				}
			}
		}
		// Try generic fallback if needed, or logging
		// HandleModalSubmit legacy function could be called here if we wanted to support both during migration,
		// but we aim to replace it.
	}
}
