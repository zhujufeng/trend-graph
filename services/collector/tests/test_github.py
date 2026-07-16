import base64
import io
import json
import os
import sys
import unittest
from contextlib import redirect_stderr, redirect_stdout
from datetime import datetime, timezone
from unittest.mock import patch

from signal_collector import cli
from signal_collector.github import GitHubCollector
from signal_collector.models import Candidate, EvidenceDetail


class FakeClient:
    def __init__(self) -> None:
        self.urls: list[str] = []

    def get_json(self, url: str, headers=None) -> dict:
        self.urls.append(url)
        if "/search/repositories?" in url:
            return {
                "items": [
                    {
                        "full_name": "owner/mcp-inspector",
                        "html_url": "https://github.com/owner/mcp-inspector",
                        "description": "Inspect MCP servers",
                        "stargazers_count": 42,
                        "created_at": "2026-06-01T00:00:00Z",
                        "pushed_at": "2026-07-15T07:00:00Z",
                        "archived": False,
                    }
                ]
            }
        if url.endswith("/repos/owner/mcp-inspector/readme"):
            content = "# MCP Inspector\n\nInstall with uv. Run against a local MCP server."
            return {"encoding": "base64", "content": base64.b64encode(content.encode()).decode()}
        if "/releases?" in url:
            return [{"name": "v1.2", "body": "Adds local inspection", "published_at": "2026-07-14T00:00:00Z"}]
        raise AssertionError(url)

    def get_text(self, url: str, headers=None) -> str:
        raise AssertionError("GitHub documentation should use the JSON contents API")


class GitHubCollectorTests(unittest.TestCase):
    def test_searches_recent_repositories_and_preserves_readme_release_evidence(self) -> None:
        client = FakeClient()
        collector = GitHubCollector(client, token="github-token")

        candidates = collector.search("mcp skill agent", limit=5)
        detail = collector.fetch_detail(candidates[0])

        self.assertEqual(candidates[0].source, "github")
        self.assertEqual(candidates[0].source_id, "owner/mcp-inspector")
        self.assertEqual(candidates[0].updated_at.isoformat(), "2026-07-15T07:00:00+00:00")
        self.assertEqual(detail.evidence_class, "original_documentation")
        self.assertFalse(detail.requires_github_verification)
        self.assertIn("Install with uv", detail.excerpt)
        self.assertIn("v1.2", detail.excerpt)
        self.assertTrue(any("sort=updated" in url and "per_page=5" in url for url in client.urls))
        self.assertFalse(any(url.endswith("/repos/owner/mcp-inspector") for url in client.urls))

    def test_cli_continues_after_one_detail_failure(self) -> None:
        first = _candidate("missing")
        second = _candidate("working")
        collector = _BatchCollector([first, second], {"missing"})
        backend = _Backend()
        stdout = io.StringIO()
        stderr = io.StringIO()

        with (
            patch.object(sys, "argv", ["collector", "--source", "github", "--query", "agent", "--ingest"]),
            patch.dict(os.environ, {"INTERNAL_INGEST_SECRET": "test-secret"}),
            patch.object(cli, "UrllibHTTPClient", return_value=object()),
            patch.object(cli, "GitHubCollector", return_value=collector),
            patch.object(cli, "BackendIngestionClient", return_value=backend),
            patch.object(cli, "shortlist", side_effect=lambda candidates: candidates),
            redirect_stdout(stdout),
            redirect_stderr(stderr),
        ):
            cli.main()

        self.assertEqual([item.source_id for item in backend.items], ["working"])
        output = json.loads(stdout.getvalue())
        self.assertEqual(output["failed"], 1)
        self.assertIn("missing", output["failures"][0])
        self.assertEqual(stderr.getvalue(), "")

    def test_cli_fails_when_every_detail_fails(self) -> None:
        candidate = _candidate("missing")
        with (
            patch.object(sys, "argv", ["collector", "--source", "github", "--query", "agent", "--ingest"]),
            patch.dict(os.environ, {"INTERNAL_INGEST_SECRET": "test-secret"}),
            patch.object(cli, "UrllibHTTPClient", return_value=object()),
            patch.object(cli, "GitHubCollector", return_value=_BatchCollector([candidate], {"missing"})),
            patch.object(cli, "BackendIngestionClient", return_value=_Backend()),
            patch.object(cli, "shortlist", side_effect=lambda candidates: candidates),
            redirect_stdout(io.StringIO()),
            redirect_stderr(io.StringIO()),
        ):
            with self.assertRaisesRegex(RuntimeError, "all shortlisted candidates failed"):
                cli.main()


def _candidate(source_id: str) -> Candidate:
    now = datetime.now(timezone.utc)
    return Candidate("github", source_id, source_id, f"https://github.com/o/{source_id}", "", "", 1, now, now)


class _BatchCollector:
    def __init__(self, candidates: list[Candidate], failures: set[str]) -> None:
        self.candidates = candidates
        self.failures = failures

    def search(self, query: str, limit: int) -> list[Candidate]:
        return self.candidates

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        if candidate.source_id in self.failures:
            raise ValueError(f"{candidate.source_id} has no README")
        return EvidenceDetail(
            candidate.source,
            candidate.source_id,
            candidate.url,
            candidate.title,
            "Install and run.",
            "original_documentation",
            False,
            candidate.published_at,
            candidate.updated_at,
        )


class _Backend:
    def __init__(self) -> None:
        self.items: list[Candidate] = []

    def ingest(self, candidate: Candidate, detail: EvidenceDetail) -> bool:
        self.items.append(candidate)
        return True


if __name__ == "__main__":
    unittest.main()
