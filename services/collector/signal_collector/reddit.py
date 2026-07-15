import base64
import re
from datetime import UTC, datetime
from urllib.parse import urlencode

from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class RedditCollector:
    source = "reddit"
    oauth_url = "https://oauth.reddit.com"
    token_url = "https://www.reddit.com/api/v1/access_token"

    def __init__(
        self,
        client: HTTPClient,
        client_id: str,
        client_secret: str,
        communities: list[str],
    ) -> None:
        if not client_id or not client_secret:
            raise ValueError("Reddit OAuth client credentials are required")
        self.client = client
        self.client_id = client_id
        self.client_secret = client_secret
        self.communities = _normalize_communities(communities)
        if not self.communities:
            raise ValueError("Reddit community allowlist must not be empty or contain r/all")
        self._access_token = ""

    def list_candidates(self, limit: int) -> list[Candidate]:
        limit = max(1, min(limit, 100))
        headers = self._oauth_headers()
        candidates: list[Candidate] = []
        for community in self.communities:
            params = urlencode({"limit": limit, "raw_json": 1})
            payload = self.client.get_json(
                f"{self.oauth_url}/r/{community}/new?{params}", headers
            )
            for child in _listing_children(payload):
                post = child.get("data", {})
                post_id = post.get("id")
                title = post.get("title")
                permalink = post.get("permalink")
                if not all(isinstance(value, str) and value for value in (post_id, title, permalink)):
                    continue
                candidates.append(
                    Candidate(
                        source=self.source,
                        source_id=post_id,
                        title=title,
                        url=f"https://www.reddit.com{permalink}",
                        discovery_url=f"{self.oauth_url}/comments/{post_id}?raw_json=1&limit=20",
                        summary=str(post.get("selftext") or ""),
                        score=float(post.get("score") or 0),
                        published_at=_from_unix(post.get("created_utc")),
                        updated_at=None,
                    )
                )
        candidates.sort(key=lambda item: item.published_at or datetime.min.replace(tzinfo=UTC), reverse=True)
        return candidates[:limit]

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        payload = self.client.get_json(candidate.discovery_url, self._oauth_headers())
        if not isinstance(payload, list) or len(payload) < 2:
            raise ValueError(f"Reddit discussion {candidate.source_id} returned invalid detail")
        posts = _listing_children(payload[0])
        selftext = str(posts[0].get("data", {}).get("selftext") or "") if posts else ""
        comments = [
            str(child.get("data", {}).get("body") or "").strip()
            for child in _listing_children(payload[1])
        ]
        excerpt = "\n\n".join(
            part for part in (candidate.title, selftext, "\n".join(filter(None, comments[:10]))) if part
        )
        return EvidenceDetail(
            source=self.source,
            source_id=candidate.source_id,
            source_url=candidate.url,
            title=candidate.title,
            excerpt=excerpt,
            evidence_class="community_discussion",
            requires_github_verification=False,
            published_at=candidate.published_at,
            updated_at=candidate.updated_at,
        )

    def _oauth_headers(self) -> dict[str, str]:
        if not self._access_token:
            credentials = base64.b64encode(
                f"{self.client_id}:{self.client_secret}".encode("utf-8")
            ).decode("ascii")
            response = self.client.post_form(
                self.token_url,
                {"grant_type": "client_credentials"},
                {"Authorization": f"Basic {credentials}"},
            )
            token = response.get("access_token")
            if not isinstance(token, str) or not token:
                raise ValueError("Reddit OAuth response did not contain an access token")
            self._access_token = token
        return {"Authorization": f"Bearer {self._access_token}"}


def _normalize_communities(values: list[str]) -> list[str]:
    communities: list[str] = []
    for value in values:
        community = value.strip().lower().removeprefix("r/")
        if community == "all" or not re.fullmatch(r"[a-z0-9_]+", community):
            continue
        if community not in communities:
            communities.append(community)
    return communities


def _listing_children(payload: object) -> list[dict]:
    if not isinstance(payload, dict):
        return []
    data = payload.get("data")
    if not isinstance(data, dict):
        return []
    children = data.get("children")
    return [child for child in children if isinstance(child, dict)] if isinstance(children, list) else []


def _from_unix(value: object) -> datetime | None:
    if not isinstance(value, (int, float)):
        return None
    try:
        return datetime.fromtimestamp(value, tz=UTC)
    except (OverflowError, OSError, ValueError):
        return None
