package radar

import (
	"testing"
	"time"
)

func TestNextRadarRunsUseShanghaiCollectionAndDigestTimes(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 7, 15, 7, 35, 0, 0, location)

	runs, err := NextRadarRuns(after)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]time.Time{
		"pre_digest_collection": time.Date(2026, 7, 15, 7, 40, 0, 0, location),
		"digest":                time.Date(2026, 7, 15, 8, 0, 0, 0, location),
		"regular_collection":    time.Date(2026, 7, 15, 9, 0, 0, 0, location),
	}
	for name, expected := range want {
		if !runs[name].Equal(expected) {
			t.Fatalf("%s next = %s, want %s", name, runs[name], expected)
		}
	}
}

func TestNewCollectionCronRegistersBothCollectionSchedules(t *testing.T) {
	scheduler, err := NewCollectionCron(func() {})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(scheduler.Entries()); got != 2 {
		t.Fatalf("collection cron entries = %d, want 2", got)
	}
}

func TestNewDigestCronRegistersMorningAndEveningSchedule(t *testing.T) {
	scheduler, err := NewDigestCron(func() {})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(scheduler.Entries()); got != 1 {
		t.Fatalf("digest cron entries = %d, want 1", got)
	}
}
