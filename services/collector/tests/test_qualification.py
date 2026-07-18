import unittest
from datetime import UTC, datetime, timedelta

from signal_collector.models import Candidate
from signal_collector.qualification import shortlist


class QualificationTests(unittest.TestCase):
    def test_shortlists_recent_ai_candidates_without_matching_maintainer(self) -> None:
        now = datetime(2026, 7, 16, tzinfo=UTC)

        def candidate(title: str, age: int) -> Candidate:
            return Candidate("github", title, title, "https://example.com", "https://example.com", "", 1, now - timedelta(days=age), None)

        result = shortlist(
            [
                candidate("AI agent workflow", 1),
                candidate("Maintainer handbook", 1),
                candidate("MCP server", 31),
            ],
            ["AI"],
            now,
        )

        self.assertEqual([item.title for item in result], ["AI agent workflow"])

    def test_supports_custom_topics_and_explicit_subscriptions(self) -> None:
        now = datetime(2026, 7, 16, tzinfo=UTC)
        robotics = Candidate("rss", "1", "机器人产业周报", "https://example.com/1", "", "具身智能进展", 0, now, None)
        release = Candidate("github", "2", "boring release", "https://github.com/o/r/releases/tag/v1", "", "", 0, now, None, subscribed=True)
        unrelated = Candidate("rss", "3", "天气", "https://example.com/3", "", "晴天", 0, now, None)

        result = shortlist([robotics, release, unrelated], ["机器人"], now)

        self.assertEqual([item.source_id for item in result], ["1", "2"])


if __name__ == "__main__":
    unittest.main()
