package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
)

func TestIngestSignalValidation(t *testing.T) {
	valid := ingestSignalRequest{
		Source: "skillsmp", OriginalURL: "https://github.com/example/skill", OriginalTitle: "Example",
		EvidenceURL: "https://skillsmp.com/skill", EvidenceClass: "catalog_discovery", EvidenceExcerpt: "A discovery record",
	}
	if got := valid.validate(); got != "" {
		t.Fatalf("valid request error = %q", got)
	}
	valid.Source = "linuxdo"
	if got := valid.validate(); got != "unsupported source" {
		t.Fatalf("source validation = %q", got)
	}
}

func TestIngestSignalHTTPReturnsRepositoryCreationResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	signals := &fakeSignalIngestor{created: false}
	router := gin.New()
	NewIngestionHandler("collector-secret", signals).Register(router)

	payload := []byte(`{
		"source":"skillsmp",
		"originalUrl":"https://github.com/owner/repo/tree/main/skills/mcp",
		"originalTitle":"MCP Inspector",
		"score":42,
		"evidenceUrl":"https://skillsmp.com/skills/owner-repo-skill",
		"evidenceTitle":"MCP Inspector",
		"evidenceClass":"catalog_discovery",
		"evidenceExcerpt":"Inspect MCP servers."
	}`)
	request := httptest.NewRequest(http.MethodPost, "/internal/ingest/signals", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(internalIngestSecretHeader, "collector-secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"created":false}` {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if signals.signal.Source != "skillsmp" || signals.signal.OriginalTitle != "MCP Inspector" {
		t.Fatalf("signal = %#v", signals.signal)
	}
	if signals.evidence.EvidenceClass != "catalog_discovery" || signals.evidence.Excerpt != "Inspect MCP servers." {
		t.Fatalf("evidence = %#v", signals.evidence)
	}
}

func TestIngestSignalHTTPRequiresInternalSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	signals := &fakeSignalIngestor{}
	router := gin.New()
	NewIngestionHandler("collector-secret", signals).Register(router)

	request := httptest.NewRequest(http.MethodPost, "/internal/ingest/signals", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if signals.signal.Source != "" {
		t.Fatalf("repository should not be called: %#v", signals.signal)
	}
}

type fakeSignalIngestor struct {
	created  bool
	signal   store.Signal
	evidence store.EvidenceSnapshot
}

func (s *fakeSignalIngestor) IngestIfNew(signal store.Signal, evidence store.EvidenceSnapshot) (bool, error) {
	s.signal = signal
	s.evidence = evidence
	return s.created, nil
}
