import unittest

from signal_collector.waytoagi import WaytoAGICollector


class FakeClient:
    def __init__(self, texts: dict[str, str]) -> None:
        self.texts = texts

    def get_text(self, url: str, headers=None) -> str:
        return self.texts[url]

    def get_json(self, url: str, headers=None) -> dict:
        raise AssertionError("JSON should not be requested")


class WaytoAGICollectorTests(unittest.TestCase):
    def test_lists_curated_articles_and_extracts_detail(self) -> None:
        client = FakeClient(
            {
                WaytoAGICollector.homepage_url: """
                    <a href='/zh/blog/news-20260714'>Codex + Remotion 实战</a>
                    <a href='/sites'>不应成为候选</a>
                    <a href='/zh/blog/news-20260714'>重复</a>
                """,
                "https://www.waytoagi.com/zh/blog/news-20260714": """
                    <html><head><title>案例详情</title></head>
                    <body><h1>Codex + Remotion 实战</h1><p>这是可复现的工作流。</p></body></html>
                """,
            }
        )
        collector = WaytoAGICollector(client)

        candidates = collector.list_candidates(10)

        self.assertEqual(len(candidates), 1)
        self.assertEqual(candidates[0].title, "Codex + Remotion 实战")
        detail = collector.fetch_detail(candidates[0])
        self.assertEqual(detail.evidence_class, "documented_third_party_practice")
        self.assertIn("这是可复现的工作流。", detail.excerpt)
