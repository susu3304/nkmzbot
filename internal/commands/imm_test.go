package commands

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNormalizeImmCommandNameTrimsPrefixes(t *testing.T) {
	for input, want := range map[string]string{
		"repeat":    "repeat",
		"?repeat":   "repeat",
		"!repeat":   "repeat",
		"  ?repeat": "repeat",
	} {
		if got := normalizeImmCommandName(input); got != want {
			t.Fatalf("normalizeImmCommandName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatImmCommandSourceFitsDiscordLimit(t *testing.T) {
	source := strings.Repeat("squeak \"hello\"\n", 300)
	got := formatImmCommandSource("?long", source)

	if runeLen(got) > discordMessageLimit {
		t.Fatalf("formatted source length = %d, want <= %d", runeLen(got), discordMessageLimit)
	}
	if !strings.Contains(got, "IMMコマンド `?long` の中身:") {
		t.Fatalf("formatted source is missing command header: %q", got[:80])
	}
	if !strings.Contains(got, "```imm\n") || !strings.HasSuffix(got, "\n```") {
		t.Fatalf("formatted source should be an imm code block: %q", got)
	}
	if !strings.Contains(got, "...(truncated)") {
		t.Fatalf("long source should include truncation marker")
	}
}

func TestFormatImmCommandSourceEscapesCodeFence(t *testing.T) {
	got := formatImmCommandSource("fence", "squeak \"before\"\n```\nsqueak \"after\"")
	if strings.Contains(got, "\n```\nsqueak \"after\"") {
		t.Fatalf("embedded code fence was not escaped: %q", got)
	}
}

func TestTruncateDiscordMessageFitsDiscordLimit(t *testing.T) {
	got := truncateDiscordMessage(strings.Repeat("あ", discordMessageLimit+100))
	if runeLen(got) > discordMessageLimit {
		t.Fatalf("truncated message length = %d, want <= %d", runeLen(got), discordMessageLimit)
	}
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Fatalf("truncated message should include marker")
	}
}

func TestGetCommandsIncludesIMMCommandShow(t *testing.T) {
	for _, cmd := range GetCommands() {
		if cmd.Name != "imm" {
			continue
		}
		for _, top := range cmd.Options {
			if top.Name != "command" {
				continue
			}
			if top.Type != discordgo.ApplicationCommandOptionSubCommandGroup {
				t.Fatalf("imm command option type = %v, want subcommand group", top.Type)
			}
			for _, sub := range top.Options {
				if sub.Name == "show" {
					if sub.Type != discordgo.ApplicationCommandOptionSubCommand {
						t.Fatalf("show option type = %v, want subcommand", sub.Type)
					}
					return
				}
			}
		}
	}
	t.Fatal("/imm command show was not registered")
}
