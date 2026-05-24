package commands

import (
	"reflect"
	"testing"
	"time"
)

func TestNextDailyOccurrenceFuture(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, 5, 20, 18, 0, 0, 0, time.UTC)

	got := nextDailyOccurrence(scheduled, now)
	if !got.Equal(scheduled) {
		t.Fatalf("nextDailyOccurrence() = %s, want %s", got, scheduled)
	}
}

func TestNextDailyOccurrencePast(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, 5, 18, 18, 0, 0, 0, time.UTC)
	want := time.Date(2026, 5, 20, 18, 0, 0, 0, time.UTC)

	got := nextDailyOccurrence(scheduled, now)
	if !got.Equal(want) {
		t.Fatalf("nextDailyOccurrence() = %s, want %s", got, want)
	}
}

func TestNextDailyOccurrenceSameTimeAdvancesOneDay(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	want := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)

	got := nextDailyOccurrence(now, now)
	if !got.Equal(want) {
		t.Fatalf("nextDailyOccurrence() = %s, want %s", got, want)
	}
}

func TestParseScheduledImmCommand(t *testing.T) {
	got, ok, err := parseScheduledImmCommand(`?repeat marmot "two words" 3`)
	if err != nil {
		t.Fatalf("parseScheduledImmCommand() error = %v", err)
	}
	if !ok {
		t.Fatal("parseScheduledImmCommand() ok = false, want true")
	}
	if got.Name != "repeat" {
		t.Fatalf("Name = %q, want repeat", got.Name)
	}
	if got.RawArgs != `marmot "two words" 3` {
		t.Fatalf("RawArgs = %q, want quoted raw args", got.RawArgs)
	}
	if want := []string{"marmot", "two words", "3"}; !reflect.DeepEqual(got.Args, want) {
		t.Fatalf("Args = %#v, want %#v", got.Args, want)
	}
}

func TestParseScheduledImmCommandIgnoresTextCommand(t *testing.T) {
	_, ok, err := parseScheduledImmCommand("!repeat marmot 3")
	if err != nil {
		t.Fatalf("parseScheduledImmCommand() error = %v", err)
	}
	if ok {
		t.Fatal("parseScheduledImmCommand() ok = true, want false")
	}
}
