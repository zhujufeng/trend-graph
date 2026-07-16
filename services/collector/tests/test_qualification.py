import unittest
from datetime import UTC, datetime, timedelta

from signal_collector.models import Candidate
from signal_collector.qualification import shortlist


class QualificationTests(unittest.TestCase):
    def test_shortlists_recent_ai_candidates_without_matching_maintainer(self) -> None:
        now = datetime(2026, 7, 16, tzinfo=UTC)

        def candidate(title: str, age: int) -> Candidate:
            return Candidate("github", title, title, "https://example.com", "https://example.com", "", 1, now - timedelta(days=age), None)

        result = shortlist([
            candidate("AI agent workflow", 1),
            candidate("Maintainer handbook", 1),
            candidate("MCP server", 31),
        ], now)

        self.assertEqual([item.title for item in result], ["AI agent workflow"])


if __name__ == "__main__":
    unittest.main()
