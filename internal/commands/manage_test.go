package commands

import (
	"testing"
)

func TestParseBulkCommands(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []BulkCommand
	}{
		{
			name:  "Single command",
			input: "!hello: Hello, world!",
			want: []BulkCommand{
				{Name: "hello", Response: "Hello, world!"},
			},
		},
		{
			name: "Multiple commands",
			input: `!hello: Hello, world!
!bye: Goodbye!
!test: This is a test`,
			want: []BulkCommand{
				{Name: "hello", Response: "Hello, world!"},
				{Name: "bye", Response: "Goodbye!"},
				{Name: "test", Response: "This is a test"},
			},
		},
		{
			name: "Commands with extra whitespace",
			input: `  !hello:   Hello, world!  
!bye:Goodbye!
  !test: This is a test  `,
			want: []BulkCommand{
				{Name: "hello", Response: "Hello, world!"},
				{Name: "bye", Response: "Goodbye!"},
				{Name: "test", Response: "This is a test"},
			},
		},
		{
			name: "Commands with empty lines",
			input: `!hello: Hello, world!

!bye: Goodbye!

!test: This is a test`,
			want: []BulkCommand{
				{Name: "hello", Response: "Hello, world!"},
				{Name: "bye", Response: "Goodbye!"},
				{Name: "test", Response: "This is a test"},
			},
		},
		{
			name: "Commands with colon in response",
			input: `!hello: Hello: world!
!time: The time is: 12:30`,
			want: []BulkCommand{
				{Name: "hello", Response: "Hello: world!"},
				{Name: "time", Response: "The time is: 12:30"},
			},
		},
		{
			name:  "Invalid: missing colon",
			input: "!hello world",
			want:  []BulkCommand{},
		},
		{
			name:  "Invalid: missing exclamation mark",
			input: "hello: world",
			want:  []BulkCommand{},
		},
		{
			name:  "Invalid: empty name",
			input: "!: world",
			want:  []BulkCommand{},
		},
		{
			name:  "Invalid: empty response",
			input: "!hello:",
			want:  []BulkCommand{},
		},
		{
			name: "Mixed valid and invalid",
			input: `!hello: Hello, world!
invalid line
!bye: Goodbye!
another invalid
!test: This is a test`,
			want: []BulkCommand{
				{Name: "hello", Response: "Hello, world!"},
				{Name: "bye", Response: "Goodbye!"},
				{Name: "test", Response: "This is a test"},
			},
		},
		{
			name:  "Empty input",
			input: "",
			want:  []BulkCommand{},
		},
		{
			name:  "Only whitespace",
			input: "   \n  \n  ",
			want:  []BulkCommand{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBulkCommands(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseBulkCommands() returned %d commands, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("parseBulkCommands()[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Response != tt.want[i].Response {
					t.Errorf("parseBulkCommands()[%d].Response = %q, want %q", i, got[i].Response, tt.want[i].Response)
				}
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	got := parseTags(" fun,info、Fun daily\tnews ")
	want := []string{"fun", "info", "daily", "news"}

	if len(got) != len(want) {
		t.Fatalf("parseTags() returned %d tags, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTags()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
