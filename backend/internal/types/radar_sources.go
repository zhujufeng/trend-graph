package types

const (
	SourceDEV     = "dev"
	SourceGitHub  = "github"
	SourceReddit  = "reddit"
	SourceBluesky = "bluesky"
	SourceRSS     = "rss"
)

// RadarSources returns the active collection set for persistence queries.
// Returning a new slice keeps callers from mutating shared package state.
func RadarSources() []string {
	return []string{SourceDEV, SourceGitHub, SourceReddit, SourceBluesky, SourceRSS}
}

// IsRadarSource is the single source-of-truth for collectors and API routes
// that are part of the first AI signal radar release. X is intentionally not
// accepted until its keyword-search collector has been designed and tested.
func IsRadarSource(source string) bool {
	switch source {
	case SourceDEV, SourceGitHub, SourceReddit, SourceBluesky, SourceRSS:
		return true
	default:
		return false
	}
}
