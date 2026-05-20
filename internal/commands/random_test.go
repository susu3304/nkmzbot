package commands

import (
	"testing"

	"github.com/susu3304/nkmzbot/internal/client"
)

func TestPickRandomCommandEmpty(t *testing.T) {
	_, ok := pickRandomCommand(nil)
	if ok {
		t.Fatal("pickRandomCommand(nil) ok = true, want false")
	}
}

func TestPickRandomCommandReturnsRegisteredCommand(t *testing.T) {
	commands := []client.CommandRecord{
		{Name: "hello", Response: "Hello"},
		{Name: "bye", Response: "Bye"},
	}
	allowed := map[string]bool{
		"hello": true,
		"bye":   true,
	}

	for range 20 {
		got, ok := pickRandomCommand(commands)
		if !ok {
			t.Fatal("pickRandomCommand(commands) ok = false, want true")
		}
		if !allowed[got.Name] {
			t.Fatalf("pickRandomCommand(commands).Name = %q, want one of registered commands", got.Name)
		}
	}
}

func TestFilterCommandsByTag(t *testing.T) {
	commands := []client.CommandRecord{
		{Name: "hello", Response: "Hello", Tags: []string{"greeting", "daily"}},
		{Name: "bye", Response: "Bye", Tags: []string{"Greeting"}},
		{Name: "weather", Response: "Sunny", Tags: []string{"info"}},
		{Name: "empty", Response: "No tags"},
	}

	got := filterCommandsByTag(commands, " greeting ")
	if len(got) != 2 {
		t.Fatalf("filterCommandsByTag() returned %d commands, want 2", len(got))
	}
	if got[0].Name != "hello" || got[1].Name != "bye" {
		t.Fatalf("filterCommandsByTag() names = [%q, %q], want [hello, bye]", got[0].Name, got[1].Name)
	}
}

func TestFilterCommandsByTagEmptyTagKeepsAllCommands(t *testing.T) {
	commands := []client.CommandRecord{
		{Name: "hello", Response: "Hello", Tags: []string{"greeting"}},
		{Name: "weather", Response: "Sunny", Tags: []string{"info"}},
	}

	got := filterCommandsByTag(commands, "")
	if len(got) != len(commands) {
		t.Fatalf("filterCommandsByTag(empty) returned %d commands, want %d", len(got), len(commands))
	}
}
