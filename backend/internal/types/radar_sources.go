package types

const (
	SourceWaytoAGI = "waytoagi"
	SourceSkillsMP = "skillsmp"
	SourceGitHub   = "github"
	SourceReddit   = "reddit"
)

// IsRadarSource is the single source-of-truth for collectors and API routes
// that are part of the first AI signal radar release. X is intentionally not
// accepted until its keyword-search collector has been designed and tested.
func IsRadarSource(source string) bool {
	switch source {
	case SourceWaytoAGI, SourceSkillsMP, SourceGitHub, SourceReddit:
		return true
	default:
		return false
	}
}
