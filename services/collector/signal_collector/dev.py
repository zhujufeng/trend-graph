import re
from datetime import UTC, datetime
from urllib.parse import urlencode

from .github import _parse_datetime
from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class DEVCollector:
    source = "dev"
    api_url = "https://dev.to/api"

    def __init__(self, client: HTTPClient) -> None:
        self.client = client
        self.headers = {"Accept": "application/vnd.forem.api-v1+json"}

    def search(self, query: str, limit: int = 20) -> list[Candidate]:
        tags = [tag for tag in re.split(r"[,\s]+", query.strip().lower()) if tag]
        if not tags:
            raise ValueError("DEV tags must not be empty")
        limit = max(1, min(limit, 100))
        found: dict[str, Candidate] = {}
        for tag in tags:
            params = urlencode({"tag": tag, "state": "fresh", "per_page": limit})
            payload = self.client.get_json(f"{self.api_url}/articles?{params}", self.headers)
            if not isinstance(payload, list):
                raise ValueError("DEV articles response must be a list")
            for article in payload:
                candidate = _candidate(article)
                if candidate is not None:
                    found[candidate.source_id] = candidate
        return sorted(
            found.values(),
            key=lambda item: item.published_at or datetime.min.replace(tzinfo=UTC),
            reverse=True,
        )[:limit]

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        payload = self.client.get_json(
            f"{self.api_url}/articles/{candidate.source_id}", self.headers
        )
        excerpt = str(payload.get("body_markdown") or payload.get("description") or "").strip()
        if not excerpt:
            raise ValueError(f"DEV article {candidate.source_id} has no readable body")
        return EvidenceDetail(
            source=self.source,
            source_id=candidate.source_id,
            source_url=candidate.url,
            title=str(payload.get("title") or candidate.title),
            excerpt=excerpt,
            evidence_class="documented_third_party_practice",
            requires_github_verification=False,
            published_at=candidate.published_at,
            updated_at=candidate.updated_at,
        )


def _candidate(article: object) -> Candidate | None:
    if not isinstance(article, dict):
        return None
    article_id = article.get("id")
    title = article.get("title")
    url = article.get("url")
    if not isinstance(article_id, int) or not isinstance(title, str) or not isinstance(url, str):
        return None
    user = article.get("user") if isinstance(article.get("user"), dict) else {}
    return Candidate(
        source=DEVCollector.source,
        source_id=str(article_id),
        title=title,
        url=url,
        discovery_url=f"{DEVCollector.api_url}/articles/{article_id}",
        summary=str(article.get("description") or ""),
        score=float(article.get("positive_reactions_count") or 0)
        + float(article.get("comments_count") or 0),
        published_at=_parse_datetime(article.get("published_at")),
        updated_at=_parse_datetime(article.get("edited_at")),
        author=str(user.get("name") or user.get("username") or ""),
    )
