package store

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"trend-graph/internal/types"
)

func TestActiveRadarSignalsFiltersRetiredSources(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=test dbname=test sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	var signals []Signal
	statement := activeRadarSignals(db).Find(&signals).Statement
	if !strings.Contains(statement.SQL.String(), "source IN") {
		t.Fatalf("query = %q", statement.SQL.String())
	}
	if !reflect.DeepEqual(statement.Vars, []any{
		types.SourceDEV, types.SourceGitHub, types.SourceReddit, types.SourceBluesky,
	}) {
		t.Fatalf("query vars = %#v", statement.Vars)
	}
}

func TestCanonicalURLRemovesFragmentAndTracking(t *testing.T) {
	got, err := CanonicalURL("HTTPS://GitHub.com/owner/repo/?utm_source=radar&ref=readme#install")
	if err != nil {
		t.Fatalf("CanonicalURL returned error: %v", err)
	}
	if want := "https://github.com/owner/repo?ref=readme"; got != want {
		t.Fatalf("CanonicalURL = %q, want %q", got, want)
	}
}

func TestNormalizedAllowlist(t *testing.T) {
	got := NormalizedAllowlist([]string{" ClaudeAI ", "claudeai", "", "LocalLLaMA"})
	want := []string{"claudeai", "localllama"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizedAllowlist = %#v, want %#v", got, want)
	}
}
