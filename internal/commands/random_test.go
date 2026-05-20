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
