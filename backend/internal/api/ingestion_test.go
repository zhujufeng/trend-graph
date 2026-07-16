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
		Source: "dev", OriginalURL: "https://dev.to/example/mcp-workflow", OriginalTitle: "Example",
		EvidenceURL: "https://dev.to/example/mcp-workflow", EvidenceClass: "documented_third_party_practice", EvidenceExcerpt: "A documented workflow",
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
		"source":"dev",
		"originalUrl":"https://dev.to/owner/mcp-workflow",
		"originalTitle":"MCP Inspector",
		"score":42,
		"evidenceUrl":"https://dev.to/owner/mcp-workflow",
		"evidenceTitle":"MCP Inspector",
		"evidenceClass":"documented_third_party_practice",
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
	if signals.signal.Source != "dev" || signals.signal.OriginalTitle != "MCP Inspector" {
		t.Fatalf("signal = %#v", signals.signal)
	}
	if signals.evidence.EvidenceClass != "documented_third_party_practice" || signals.evidence.Excerpt != "Inspect MCP servers." {
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
