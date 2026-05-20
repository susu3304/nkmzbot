package commands

import (
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
