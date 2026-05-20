package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateWalletTransferUsesDiscordIDPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/wallet/transfers" {
			t.Fatalf("path = %s, want /wallet/transfers", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("X-Discord-User-ID"); got != "actor-discord-id" {
			t.Fatalf("X-Discord-User-ID = %q, want %q", got, "actor-discord-id")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["toDiscordId"]; got != "target-discord-id" {
			t.Fatalf("toDiscordId = %v, want %q", got, "target-discord-id")
		}
		if _, exists := body["toUserId"]; exists {
			t.Fatalf("toUserId must not be sent")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"event-1","kind":"transfer","fromUserId":"wallet-user-a","toUserId":"wallet-user-b","amount":1500,"settledAmount":0,"remainingAmount":1500,"status":"open","note":"lunch","createdByUserId":"wallet-user-a","createdAt":"2026-03-09T09:30:00.000Z","updatedAt":"2026-03-09T09:30:00.000Z"}`))
	}))
	defer server.Close()

	cli := New(server.URL, "test-token")
	event, err := cli.CreateWalletTransfer("actor-discord-id", "target-discord-id", 1500, "lunch")
	if err != nil {
		t.Fatalf("CreateWalletTransfer() error = %v", err)
	}
	if event == nil {
		t.Fatal("CreateWalletTransfer() returned nil event")
	}
	if event.Kind != "transfer" {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, "transfer")
	}
}

func TestCreateWalletRequestUsesDiscordIDPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/wallet/requests" {
			t.Fatalf("path = %s, want /wallet/requests", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("X-Discord-User-ID"); got != "actor-discord-id" {
			t.Fatalf("X-Discord-User-ID = %q, want %q", got, "actor-discord-id")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["fromDiscordId"]; got != "target-discord-id" {
			t.Fatalf("fromDiscordId = %v, want %q", got, "target-discord-id")
		}
		if _, exists := body["fromUserId"]; exists {
			t.Fatalf("fromUserId must not be sent")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"event-2","kind":"request","fromUserId":"wallet-user-b","toUserId":"wallet-user-a","amount":1500,"settledAmount":0,"remainingAmount":1500,"status":"open","note":"lunch","createdByUserId":"wallet-user-a","createdAt":"2026-03-09T10:00:00.000Z","updatedAt":"2026-03-09T10:00:00.000Z"}`))
	}))
	defer server.Close()

	cli := New(server.URL, "test-token")
	event, err := cli.CreateWalletRequest("actor-discord-id", "target-discord-id", 1500, "lunch")
	if err != nil {
		t.Fatalf("CreateWalletRequest() error = %v", err)
	}
	if event == nil {
		t.Fatal("CreateWalletRequest() returned nil event")
	}
	if event.Kind != "request" {
		t.Fatalf("event.Kind = %q, want %q", event.Kind, "request")
	}
}

func TestAddCommandSendsOptionalTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/bot/guilds/guild-1/commands/add" {
			t.Fatalf("path = %s, want /bot/guilds/guild-1/commands/add", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["name"]; got != "hello" {
			t.Fatalf("name = %v, want hello", got)
		}
		tags, ok := body["tags"].([]any)
		if !ok {
			t.Fatalf("tags = %#v, want array", body["tags"])
		}
		if len(tags) != 2 || tags[0] != "fun" || tags[1] != "daily" {
			t.Fatalf("tags = %#v, want [fun daily]", tags)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cli := New(server.URL, "")
	if err := cli.AddCommand("guild-1", "hello", "Hello", " fun ", "daily", "FUN"); err != nil {
		t.Fatalf("AddCommand() error = %v", err)
	}
}

func TestRecordCommandUsagePostsUsageEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/bot/guilds/guild-1/commands/hello/usage" {
			t.Fatalf("path = %s, want /bot/guilds/guild-1/commands/hello/usage", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["source"]; got != "bot-message" {
			t.Fatalf("source = %q, want %q", got, "bot-message")
		}
		if got := body["actorId"]; got != "user-1" {
			t.Fatalf("actorId = %q, want %q", got, "user-1")
		}
		if got := body["channelId"]; got != "channel-1" {
			t.Fatalf("channelId = %q, want %q", got, "channel-1")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cli := New(server.URL, "")
	if err := cli.RecordCommandUsage("guild-1", "hello", CommandUsageInput{
		ActorID:   "user-1",
		ChannelID: "channel-1",
		Source:    "bot-message",
	}); err != nil {
		t.Fatalf("RecordCommandUsage() error = %v", err)
	}
}
