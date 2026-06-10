package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/susu3304/nkmzbot/internal/client"
)

func TestCommandExpanderExpandsCommandArguments(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "world"},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "!hello literal", []string{"!hello", "literal"})
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	if len(got.Args) != 2 || got.Args[0] != "world" || got.Args[1] != "literal" {
		t.Fatalf("Args = %#v, want [world literal]", got.Args)
	}
	if got.RawArgs != "world literal" {
		t.Fatalf("RawArgs = %q, want world literal", got.RawArgs)
	}
}

func TestCommandExpanderExpandsParenthesizedCommandArguments(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "hello world"},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "(!hello) literal", nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	if len(got.Args) != 2 || got.Args[0] != "hello world" || got.Args[1] != "literal" {
		t.Fatalf("Args = %#v, want [hello world literal]", got.Args)
	}
	if got.RawArgs != `"hello world" literal` {
		t.Fatalf("RawArgs = %q, want quoted expanded arg", got.RawArgs)
	}
}

func TestCommandExpanderExpandsNestedParenthesizedCommandArguments(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"a": {Kind: "text", Response: "(!b)"},
		"b": {Kind: "text", Response: "done"},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "(!a)", nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	if len(got.Args) != 1 || got.Args[0] != "done" {
		t.Fatalf("Args = %#v, want [done]", got.Args)
	}
}

func TestCommandExpanderDoesNotExpandQuotedCommandArguments(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "world"},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, `"(!hello)" "?hello"`, nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	want := []string{"(!hello)", "?hello"}
	if len(got.Args) != len(want) || got.Args[0] != want[0] || got.Args[1] != want[1] {
		t.Fatalf("Args = %#v, want %#v", got.Args, want)
	}
}

func TestCommandExpanderExpandsBotCommandCallsInSource(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "world"},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_command("!hello")`, "", nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	if !strings.Contains(got.Source, `squeak "world"`) {
		t.Fatalf("Source = %q, want bot_command replacement", got.Source)
	}
}

func TestCommandExpanderExpandsBotCommandAPICallsInSource(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "world"},
		"howl":  {Kind: "imm", Response: `squeak "awoo"`},
	})

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `stash commands = bot_commands("all")
stash info = bot_command_info("?howl")
squeak bot_expand("!hello") + bot_command_body("!hello")`, "", nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	for _, want := range []string{
		`"prefix": "!"`,
		`"prefix": "?"`,
		`"name": "howl"`,
		`squeak "world" + "world"`,
	} {
		if !strings.Contains(got.Source, want) {
			t.Fatalf("Source = %q, missing %q", got.Source, want)
		}
	}
}

func TestCommandExpanderSkipsBotCommandInsideStringsAndComments(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"hello": {Kind: "text", Response: "world"},
	})
	source := "# bot_command(\"!hello\")\nsqueak \"bot_command(\\\"!hello\\\")\""

	got, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, source, "", nil)
	if err != nil {
		t.Fatalf("ExpandRequest() error = %v", err)
	}
	if got.Source != source {
		t.Fatalf("Source changed inside string/comment:\n%s", got.Source)
	}
}

func TestCommandExpanderStopsAtMaxDepth(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"a": {Kind: "text", Response: "!b"},
		"b": {Kind: "text", Response: "!c"},
		"c": {Kind: "text", Response: "!d"},
		"d": {Kind: "text", Response: "done"},
	})

	_, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "!a", []string{"!a"})
	if err == nil || !strings.Contains(err.Error(), "command expansion depth exceeded: !a -> !b -> !c -> !d") {
		t.Fatalf("ExpandRequest() error = %v, want depth chain", err)
	}
}

func TestCommandExpanderAllowsRecursiveReferencesUntilMaxDepth(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"a": {Kind: "text", Response: "!a"},
	})

	_, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "!a", []string{"!a"})
	if err == nil || !strings.Contains(err.Error(), "command expansion depth exceeded: !a -> !a -> !a -> !a") {
		t.Fatalf("ExpandRequest() error = %v, want depth chain", err)
	}
}

func TestSplitExpansionArgsKeepsCommandExpressionsTogether(t *testing.T) {
	got, err := SplitExpansionArgs(`(?command2 aaaaa) (?a (?b "two words")) "(!quoted)" tail`)
	if err != nil {
		t.Fatalf("SplitExpansionArgs() error = %v", err)
	}
	want := []string{"(?command2 aaaaa)", `(?a (?b "two words"))`, "(!quoted)", "tail"}
	if len(got) != len(want) {
		t.Fatalf("SplitExpansionArgs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitExpansionArgs() = %#v, want %#v", got, want)
		}
	}
}

func newExpansionTestClient(t *testing.T, commands map[string]client.BotCommandResponse) *client.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := "/bot/guilds/g1/commands/"
		if r.Method == http.MethodGet && r.URL.Path == "/bot/guilds/g1/commands" {
			records := make([]client.CommandRecord, 0, len(commands))
			for name, cmd := range commands {
				kind := cmd.Kind
				if kind == "" {
					kind = "text"
				}
				records = append(records, client.CommandRecord{
					GuildID:  "g1",
					Name:     name,
					Kind:     kind,
					Response: cmd.Response,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.CommandsListResponse{Commands: records})
			return
		}
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, prefix) {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		name, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, prefix))
		if err != nil {
			t.Fatalf("PathUnescape() error = %v", err)
		}
		cmd, ok := commands[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if cmd.Kind == "" {
			cmd.Kind = "text"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cmd)
	}))
	t.Cleanup(server.Close)
	return client.New(server.URL, "")
}
