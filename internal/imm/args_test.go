package imm

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	got, err := SplitArgs(`a 3 "two words" 'and more'`)
	if err != nil {
		t.Fatalf("SplitArgs() error = %v", err)
	}
	want := []string{"a", "3", "two words", "and more"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitArgs() = %#v, want %#v", got, want)
	}
}

func TestRunnerRunWithConfiguredBinary(t *testing.T) {
	binary := os.Getenv("IMM_TEST_BINARY")
	if binary == "" {
		t.Skip("IMM_TEST_BINARY is not set")
	}

	runner := NewRunner(binary)
	result, err := runner.Run(context.Background(), Request{
		Source:  `squeak bot_args[0] + bot_args[0] + bot_args[0]`,
		Args:    []string{"a"},
		RawArgs: "a",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("Run() exit = %d stderr = %s", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "aaa" {
		t.Fatalf("Run() stdout = %q, want aaa", result.Stdout)
	}
}

func TestBuildSourceWrapsSnippetAndInjectsBotArgs(t *testing.T) {
	source := BuildSource(Request{
		Source:  `squeak bot_args[0]`,
		Args:    []string{"a"},
		RawArgs: "a",
		UserID:  "u1",
	})

	for _, want := range []string{
		`stash bot_args = ["a"]`,
		`stash bot_raw = "a"`,
		`stash bot_user_id = "u1"`,
		"marmot main {",
		"    squeak bot_args[0]",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("BuildSource() missing %q in:\n%s", want, source)
		}
	}
}
