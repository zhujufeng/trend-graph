import unittest

from signal_collector.reddit import RedditCollector


class FakeClient:
    def __init__(self) -> None:
        self.urls: list[str] = []
        self.token_headers = None

    def post_form(self, url: str, payload: dict, headers=None) -> dict:
        self.urls.append(url)
        self.token_headers = headers
        if payload != {"grant_type": "client_credentials"}:
            raise AssertionError(payload)
        return {"access_token": "reddit-token"}

    def get_json(self, url: str, headers=None):
        self.urls.append(url)
        if "/r/claudeai/new?" in url:
            return {
                "data": {
                    "children": [
                        {
                            "data": {
                                "id": "abc123",
                                "title": "Claude Code MCP setup pain point",
                                "selftext": "The setup fails without a clear error.",
                                "permalink": "/r/ClaudeAI/comments/abc123/claude_code_mcp_setup/",
                                "score": 55,
                                "created_utc": 1784088000,
                            }
                        }
                    ]
                }
            }
        if "/comments/abc123?" in url:
            return [
                {"data": {"children": [{"data": {"selftext": "The setup fails without a clear error."}}]}},
                {"data": {"children": [{"data": {"body": "Pin the MCP package version, then retry."}}]}},
            ]
        raise AssertionError(url)


class RedditCollectorTests(unittest.TestCase):
    def test_collects_only_oauth_allowlist_and_preserves_discussion_evidence(self) -> None:
        client = FakeClient()
        collector = RedditCollector(
            client,
            client_id="client-id",
            client_secret="client-secret",
            communities=["r/ClaudeAI", "r/all"],
        )

        candidates = collector.list_candidates(limit=10)
        detail = collector.fetch_detail(candidates[0])

        self.assertEqual(candidates[0].source, "reddit")
        self.assertEqual(candidates[0].url, "https://www.reddit.com/r/ClaudeAI/comments/abc123/claude_code_mcp_setup/")
        self.assertEqual(detail.evidence_class, "community_discussion")
        self.assertIn("Pin the MCP package version", detail.excerpt)
        self.assertTrue(any("/r/claudeai/new?" in url for url in client.urls))
        self.assertFalse(any("/r/all/" in url for url in client.urls))
        self.assertEqual(client.token_headers["Authorization"], "Basic Y2xpZW50LWlkOmNsaWVudC1zZWNyZXQ=")


if __name__ == "__main__":
    unittest.main()
