import unittest

from signal_collector.bluesky import BlueskyCollector


class FakeClient:
    def __init__(self) -> None:
        self.urls: list[str] = []

    def get_json(self, url: str, headers=None):
        self.urls.append(url)
        if "searchPosts?" in url:
            return {
                "posts": [
                    {
                        "uri": "at://did:plc:builder/app.bsky.feed.post/abc123",
                        "author": {"handle": "builder.example", "displayName": "Builder"},
                        "record": {
                            "text": "Built an MCP workflow that turns research into drafts.",
                            "createdAt": "2026-07-15T08:00:00Z",
                        },
                        "likeCount": 5,
                        "repostCount": 3,
                        "replyCount": 2,
                    }
                ]
            }
        if "getPostThread?" in url:
            return {
                "thread": {
                    "post": {
                        "record": {"text": "Built an MCP workflow that turns research into drafts."},
                        "embed": {
                            "external": {
                                "title": "Source repository",
                                "description": "Installation and examples",
                                "uri": "https://github.com/example/workflow",
                            }
                        },
                    },
                    "replies": [
                        {"post": {"record": {"text": "I reproduced it with Codex."}}}
                    ],
                }
            }
        raise AssertionError(url)


class BlueskyCollectorTests(unittest.TestCase):
    def test_searches_public_posts_and_preserves_thread_evidence(self) -> None:
        client = FakeClient()
        collector = BlueskyCollector(client)

        candidates = collector.search("MCP,Agent Skills", limit=10)
        detail = collector.fetch_detail(candidates[0])

        self.assertEqual(len(candidates), 1)
        self.assertEqual(candidates[0].source, "bluesky")
        self.assertEqual(candidates[0].author, "Builder")
        self.assertEqual(candidates[0].score, 13)
        self.assertEqual(
            candidates[0].url,
            "https://bsky.app/profile/builder.example/post/abc123",
        )
        self.assertEqual(detail.evidence_class, "community_discussion")
        self.assertIn("https://github.com/example/workflow", detail.excerpt)
        self.assertIn("I reproduced it with Codex", detail.excerpt)
        self.assertEqual(sum("searchPosts?" in url for url in client.urls), 2)


if __name__ == "__main__":
    unittest.main()
