import base64
import binascii
from datetime import datetime
from urllib.parse import quote, urlencode, urlparse

from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class GitHubCollector:
    source = "github"
    api_url = "https://api.github.com"

    def __init__(self, client: HTTPClient, token: str = "") -> None:
        self.client = client
        self.headers = {
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }
        if token:
            self.headers["Authorization"] = f"Bearer {token}"

    def search(self, query: str, limit: int = 20) -> list[Candidate]:
        if not query.strip():
            raise ValueError("GitHub search query must not be empty")
        limit = max(1, min(limit, 100))
        params = urlencode(
            {"q": f"{query} archived:false", "sort": "updated", "order": "desc", "per_page": limit}
        )
        payload = self.client.get_json(f"{self.api_url}/search/repositories?{params}", self.headers)
        candidates: list[Candidate] = []
        for repo in payload.get("items", []):
            full_name = repo.get("full_name")
            html_url = repo.get("html_url")
            if repo.get("archived") or not isinstance(full_name, str) or not isinstance(html_url, str):
                continue
            candidates.append(
                Candidate(
                    source=self.source,
                    source_id=full_name,
                    title=full_name,
                    url=html_url,
                    discovery_url=html_url,
                    summary=str(repo.get("description") or ""),
                    score=float(repo.get("stargazers_count") or 0),
                    published_at=_parse_datetime(repo.get("created_at")),
                    updated_at=_parse_datetime(repo.get("pushed_at") or repo.get("updated_at")),
                )
            )
        return candidates

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        repo, document_url = _repository_and_document(candidate.url)
        metadata = self.client.get_json(f"{self.api_url}/repos/{repo}", self.headers)
        document_payload = self.client.get_json(document_url, self.headers)
        content = document_payload.get("content") if isinstance(document_payload, dict) else None
        try:
            documentation = base64.b64decode(content).decode("utf-8").strip() if isinstance(content, str) else ""
        except (binascii.Error, UnicodeDecodeError):
            documentation = ""
        if not documentation:
            raise ValueError(f"GitHub repository {repo} has no readable documentation")
        releases = self.client.get_json(
            f"{self.api_url}/repos/{repo}/releases?per_page=1", self.headers
        )
        release = releases[0] if isinstance(releases, list) and releases else None
        excerpt = documentation
        if isinstance(release, dict):
            excerpt += f"\n\nLatest release: {release.get('name') or release.get('tag_name') or ''}\n{release.get('body') or ''}"
        return EvidenceDetail(
            source=candidate.source,
            source_id=candidate.source_id,
            source_url=candidate.url,
            title=str(metadata.get("full_name") or candidate.title),
            excerpt=excerpt,
            evidence_class="original_documentation",
            requires_github_verification=False,
            published_at=candidate.published_at,
            updated_at=candidate.updated_at,
        )


def _repository_and_document(url: str) -> tuple[str, str]:
    parsed = urlparse(url)
    parts = [part for part in parsed.path.split("/") if part]
    if parsed.hostname not in {"github.com", "www.github.com"} or len(parts) < 2:
        raise ValueError("GitHub evidence URL must name a repository")
    repo = f"{parts[0]}/{parts[1]}"
    if len(parts) >= 5 and parts[2] in {"tree", "blob"}:
        ref = parts[3]
        path = "/".join(parts[4:])
        if parts[2] == "tree":
            path = f"{path.rstrip('/')}/SKILL.md"
        document = f"{GitHubCollector.api_url}/repos/{repo}/contents/{quote(path, safe='/')}?{urlencode({'ref': ref})}"
        return repo, document
    return repo, f"{GitHubCollector.api_url}/repos/{repo}/readme"


def _parse_datetime(value: object) -> datetime | None:
    if not isinstance(value, str):
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None
