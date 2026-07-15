package store

import (
	"reflect"
	"testing"
)

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
