package bot

import (
	"testing"

	"github.com/susu3304/nkmzbot/internal/client"
)

func TestCommandKindMatchesDefaultsToText(t *testing.T) {
	if !commandKindMatches(&client.BotCommandResponse{}, textCommandKind) {
		t.Fatal("empty command kind should match text")
	}
	if commandKindMatches(&client.BotCommandResponse{}, immCommandKind) {
		t.Fatal("empty command kind should not match imm")
	}
	if !commandKindMatches(&client.BotCommandResponse{Kind: "IMM"}, immCommandKind) {
		t.Fatal("IMM command kind should match case-insensitively")
	}
}
