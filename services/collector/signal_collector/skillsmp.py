from datetime import UTC, datetime
from urllib.parse import urlencode

from .github import GitHubCollector
from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class SkillsMPCollector:
    source = "skillsmp"
    search_url = "https://skillsmp.com/api/v1/skills/search"

    def __init__(self, client: HTTPClient, api_key: str = "", github_token: str = "") -> None:
        self.client = client
        self.api_key = api_key
        self.github = GitHubCollector(client, github_token)

    def search(self, query: str, limit: int = 20) -> list[Candidate]:
        if not query.strip():
            raise ValueError("SkillsMP search query must not be empty")
        params = urlencode({"q": query, "limit": max(1, min(limit, 100)), "sortBy": "recent"})
        headers = {"Authorization": f"Bearer {self.api_key}"} if self.api_key else None
        payload = self.client.get_json(f"{self.search_url}?{params}", headers)
        if not payload.get("success"):
            raise ValueError("SkillsMP returned an unsuccessful response")
        skills = payload.get("data", {}).get("skills", [])
        candidates: list[Candidate] = []
        for skill in skills:
            github_url = skill.get("githubUrl")
            skill_id = skill.get("id")
            name = skill.get("name")
            if not isinstance(github_url, str) or not isinstance(skill_id, str) or not isinstance(name, str):
                continue
            updated_at = _parse_unix_timestamp(skill.get("updatedAt"))
            candidates.append(
                Candidate(
                    source=self.source,
                    source_id=skill_id,
                    title=name,
                    url=github_url,
                    discovery_url=str(skill.get("skillUrl") or github_url),
                    summary=str(skill.get("description") or ""),
                    score=float(skill.get("stars") or 0),
                    published_at=None,
                    updated_at=updated_at,
                )
            )
        return candidates

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        return self.github.fetch_detail(candidate)


def _parse_unix_timestamp(value: object) -> datetime | None:
    if not isinstance(value, (str, int)):
        return None
    try:
        return datetime.fromtimestamp(int(value), tz=UTC)
    except (OverflowError, OSError, ValueError):
        return None
