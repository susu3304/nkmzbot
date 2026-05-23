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
	if err == nil || !strings.Contains(err.Error(), "max depth 3") {
		t.Fatalf("ExpandRequest() error = %v, want max depth", err)
	}
}

func TestCommandExpanderDetectsLoops(t *testing.T) {
	cli := newExpansionTestClient(t, map[string]client.BotCommandResponse{
		"a": {Kind: "text", Response: "!a"},
	})

	_, err := NewCommandExpander(cli, nil).ExpandRequest(context.Background(), CommandExecutionContext{GuildID: "g1"}, `squeak bot_args[0]`, "!a", []string{"!a"})
	if err == nil || !strings.Contains(err.Error(), "loop detected") {
		t.Fatalf("ExpandRequest() error = %v, want loop", err)
	}
}

func newExpansionTestClient(t *testing.T, commands map[string]client.BotCommandResponse) *client.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := "/bot/guilds/g1/commands/"
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
