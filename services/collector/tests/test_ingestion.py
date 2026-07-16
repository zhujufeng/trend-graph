import unittest
from datetime import UTC, datetime

from signal_collector.ingestion import BackendIngestionClient
from signal_collector.models import Candidate, EvidenceDetail


class CaptureHTTPClient:
    def __init__(self) -> None:
        self.url = ""
        self.payload: dict[str, object] = {}
        self.headers: dict[str, str] = {}

    def post_json(self, url, payload, headers):
        self.url = url
        self.payload = payload
        self.headers = headers
        return {"created": True}


class IngestionContractTests(unittest.TestCase):
    def test_posts_authenticated_signal_contract(self) -> None:
        candidate = Candidate(
            source="dev",
            source_id="42",
            title="MCP Inspector",
            url="https://dev.to/builder/mcp-inspector-42",
            discovery_url="https://dev.to/api/articles/42",
            summary="Inspect MCP servers.",
            score=42,
            published_at=None,
            updated_at=datetime(2026, 7, 15, 8, 0, tzinfo=UTC),
            author="Builder",
        )
        detail = EvidenceDetail(
            source="dev",
            source_id="42",
            source_url=candidate.url,
            title=candidate.title,
            excerpt="Install with uv and run against a local MCP server.",
            evidence_class="original_documentation",
            requires_github_verification=False,
            published_at=None,
            updated_at=candidate.updated_at,
        )

        http = CaptureHTTPClient()
        created = BackendIngestionClient("https://backend.invalid", "collector-secret", http).ingest(candidate, detail)

        self.assertTrue(created)
        self.assertEqual(http.url, "https://backend.invalid/internal/ingest/signals")
        self.assertEqual(http.headers["X-Internal-Ingest-Secret"], "collector-secret")
        self.assertEqual(
            http.payload,
            {
                "source": "dev",
                "originalUrl": "https://dev.to/builder/mcp-inspector-42",
                "originalTitle": "MCP Inspector",
                "author": "Builder",
                "score": 42,
                "publishedAt": None,
                "updatedAt": "2026-07-15T08:00:00+00:00",
                "evidenceUrl": "https://dev.to/builder/mcp-inspector-42",
                "evidenceTitle": "MCP Inspector",
                "evidenceClass": "original_documentation",
                "evidenceExcerpt": "Install with uv and run against a local MCP server.",
            },
        )


if __name__ == "__main__":
    unittest.main()
