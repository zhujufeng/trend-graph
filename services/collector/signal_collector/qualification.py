import re
from datetime import UTC, datetime, timedelta

from .models import Candidate


AI_TERMS = (
    "llm", "agent", "mcp", "skill", "vibe coding", "deepseek", "claude", "openai",
    "chatgpt", "gemini", "人工智能", "大模型", "智能体", "工作流", "提示词", "自动化",
)
AI_TOKEN = re.compile(r"(?<![a-z])ai(?![a-z])", re.IGNORECASE)


def shortlist(candidates: list[Candidate], now: datetime | None = None) -> list[Candidate]:
    now = now or datetime.now(UTC)
    cutoff = now - timedelta(days=30)
    return [candidate for candidate in candidates if _recent(candidate, cutoff) and _ai_related(candidate)]


def _recent(candidate: Candidate, cutoff: datetime) -> bool:
    timestamp = candidate.updated_at or candidate.published_at
    if timestamp is not None and timestamp.tzinfo is None:
        timestamp = timestamp.replace(tzinfo=UTC)
    return timestamp is None or timestamp >= cutoff


def _ai_related(candidate: Candidate) -> bool:
    if candidate.source == "reddit":
        return True
    text = f"{candidate.title} {candidate.summary}".lower()
    return bool(AI_TOKEN.search(text)) or any(term in text for term in AI_TERMS)
