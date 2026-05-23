package commands

import (
	"testing"

	"github.com/susu3304/nkmzbot/internal/client"
)

func TestCommandPrefix(t *testing.T) {
	tests := []struct {
		name string
		cmd  client.CommandRecord
		want string
	}{
		{name: "text", cmd: client.CommandRecord{Kind: "text"}, want: "!"},
		{name: "legacy", cmd: client.CommandRecord{}, want: "!"},
		{name: "imm", cmd: client.CommandRecord{Kind: "imm"}, want: "?"},
		{name: "imm uppercase", cmd: client.CommandRecord{Kind: "IMM"}, want: "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandPrefix(tt.cmd); got != tt.want {
				t.Fatalf("commandPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
