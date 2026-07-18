import unittest

from signal_collector.rss import RSSCollector


RSS = """<?xml version="1.0"?>
<rss version="2.0"><channel><title>示例</title>
  <item><guid>one</guid><title>机器人进入工厂</title><link>https://example.com/robot</link>
  <description><![CDATA[<p>具身智能开始在生产线上落地。</p>]]></description>
  <pubDate>Thu, 16 Jul 2026 08:00:00 +0000</pubDate></item>
</channel></rss>"""

ATOM = """<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom"><title>Example</title>
  <entry><id>tag:example.com,2026:two</id><title>AI research update</title>
  <link href="/ai"/><summary><b>New model</b> evaluation.</summary>
  <updated>2026-07-16T09:00:00Z</updated><author><name>Editor</name></author></entry>
</feed>"""

TITLE_ONLY = """<?xml version="1.0"?><rss version="2.0"><channel>
  <item><guid>empty</guid><title>机器人标题</title><link>https://example.com/empty</link></item>
</channel></rss>"""


class FakeClient:
    def get_text(self, url: str, headers=None) -> str:
        if url.endswith("rss.xml"):
            return RSS
        if url.endswith("atom.xml"):
            return ATOM
        if url.endswith("empty.xml"):
            return TITLE_ONLY
        raise ValueError("broken feed")


class RSSCollectorTests(unittest.TestCase):
    def test_parses_rss_and_atom_and_keeps_working_when_one_feed_fails(self) -> None:
        collector = RSSCollector(
            FakeClient(),
            ["https://example.com/rss.xml", "https://example.com/broken.xml", "https://example.com/atom.xml"],
        )

        candidates = collector.list_candidates(10)
        detail = collector.fetch_detail(next(item for item in candidates if item.url.endswith("/robot")))

        self.assertEqual(len(candidates), 2)
        self.assertEqual(len(collector.failures), 1)
        self.assertEqual(detail.evidence_class, "publisher_feed")
        self.assertEqual(detail.excerpt, "具身智能开始在生产线上落地。")
        atom = next(item for item in candidates if item.url.endswith("/ai"))
        self.assertEqual(atom.summary, "New model evaluation.")
        self.assertEqual(atom.author, "Editor")
        self.assertEqual(atom.updated_at.isoformat(), "2026-07-16T09:00:00+00:00")

    def test_fails_when_every_feed_is_invalid(self) -> None:
        collector = RSSCollector(FakeClient(), ["https://example.com/broken.xml"])
        with self.assertRaisesRegex(RuntimeError, "all RSS feeds failed"):
            collector.list_candidates()

    def test_title_only_entry_is_not_accepted_as_evidence(self) -> None:
        collector = RSSCollector(FakeClient(), ["https://example.com/empty.xml"])
        candidate = collector.list_candidates()[0]
        with self.assertRaisesRegex(ValueError, "no readable content"):
            collector.fetch_detail(candidate)


if __name__ == "__main__":
    unittest.main()
