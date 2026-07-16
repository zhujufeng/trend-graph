import unittest

from signal_collector.dev import DEVCollector


class FakeClient:
    def __init__(self) -> None:
        self.urls: list[str] = []

    def get_json(self, url: str, headers=None):
        self.urls.append(url)
        if "/articles?" in url:
            return [
                {
                    "id": 42,
                    "title": "I built an MCP workflow with Claude Code",
                    "url": "https://dev.to/builder/mcp-workflow-42",
                    "description": "Exact setup and production failures.",
                    "positive_reactions_count": 12,
                    "comments_count": 3,
                    "published_at": "2026-07-15T08:00:00Z",
                    "edited_at": "2026-07-15T09:00:00Z",
                    "user": {"name": "Builder", "username": "builder"},
                }
            ]
        if url.endswith("/articles/42"):
            return {
                "title": "I built an MCP workflow with Claude Code",
                "body_markdown": "# Setup\n\nInstall the MCP server, run it, and verify the output.",
            }
        raise AssertionError(url)


class DEVCollectorTests(unittest.TestCase):
    def test_collects_tagged_articles_once_and_preserves_full_body(self) -> None:
        client = FakeClient()
        collector = DEVCollector(client)

        candidates = collector.search("mcp,claudecode", limit=10)
        detail = collector.fetch_detail(candidates[0])

        self.assertEqual(len(candidates), 1)
        self.assertEqual(candidates[0].source, "dev")
        self.assertEqual(candidates[0].author, "Builder")
        self.assertEqual(candidates[0].score, 15)
        self.assertEqual(detail.evidence_class, "documented_third_party_practice")
        self.assertIn("verify the output", detail.excerpt)
        self.assertEqual(sum("/articles?" in url for url in client.urls), 2)


if __name__ == "__main__":
    unittest.main()
