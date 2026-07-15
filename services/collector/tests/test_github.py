import base64
import unittest

from signal_collector.github import GitHubCollector


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
        if url.endswith("/repos/owner/mcp-inspector"):
            return {"full_name": "owner/mcp-inspector", "description": "Inspect MCP servers"}
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


if __name__ == "__main__":
    unittest.main()
