package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestModalInputValue(t *testing.T) {
	data := discordgo.ModalSubmitInteractionData{
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "command_name", Value: "  repeat  "},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "imm_args", Value: ` a "hello world" `},
				},
			},
		},
	}

	if got := modalInputValue(data, "command_name"); got != "repeat" {
		t.Fatalf("modalInputValue(command_name) = %q, want repeat", got)
	}
	if got := modalInputValue(data, "imm_args"); got != `a "hello world"` {
		t.Fatalf("modalInputValue(imm_args) = %q", got)
	}
	if got := modalInputValue(data, "missing"); got != "" {
		t.Fatalf("modalInputValue(missing) = %q, want empty", got)
	}
}

func TestGetCommandsIncludesIMMContextCommands(t *testing.T) {
	got := map[string]discordgo.ApplicationCommandType{}
	for _, cmd := range GetCommands() {
		got[cmd.Name] = cmd.Type
	}

	for _, name := range []string{"Run as IMM", "Register as IMM"} {
		if got[name] != discordgo.MessageApplicationCommand {
			t.Fatalf("%s type = %v, want MessageApplicationCommand", name, got[name])
		}
	}
}
