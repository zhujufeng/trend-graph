from datetime import UTC, datetime
from urllib.parse import urlencode

from .github import _parse_datetime
from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class BlueskyCollector:
    source = "bluesky"
    api_url = "https://api.bsky.app/xrpc"

    def __init__(self, client: HTTPClient) -> None:
        self.client = client

    def search(self, query: str, limit: int = 20) -> list[Candidate]:
        queries = [item.strip() for item in query.split(",") if item.strip()]
        if not queries:
            raise ValueError("Bluesky search query must not be empty")
        limit = max(1, min(limit, 100))
        found: dict[str, Candidate] = {}
        for term in queries:
            params = urlencode({"q": term, "limit": limit, "sort": "latest"})
            payload = self.client.get_json(
                f"{self.api_url}/app.bsky.feed.searchPosts?{params}"
            )
            posts = payload.get("posts") if isinstance(payload, dict) else None
            if not isinstance(posts, list):
                raise ValueError("Bluesky search response must contain posts")
            for post in posts:
                candidate = _candidate(post)
                if candidate is not None:
                    found[candidate.source_id] = candidate
        return sorted(
            found.values(),
            key=lambda item: item.published_at or datetime.min.replace(tzinfo=UTC),
            reverse=True,
        )[:limit]

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        params = urlencode({"uri": candidate.source_id, "depth": 1, "parentHeight": 0})
        payload = self.client.get_json(
            f"{self.api_url}/app.bsky.feed.getPostThread?{params}"
        )
        thread = payload.get("thread") if isinstance(payload, dict) else None
        if not isinstance(thread, dict):
            raise ValueError(f"Bluesky post {candidate.source_id} returned invalid detail")
        root = _post_excerpt(thread.get("post"))
        replies = thread.get("replies") if isinstance(thread.get("replies"), list) else []
        reply_text = [_post_excerpt(reply.get("post")) for reply in replies if isinstance(reply, dict)]
        excerpt = "\n\n".join(part for part in [root, *reply_text[:10]] if part)
        if not excerpt:
            raise ValueError(f"Bluesky post {candidate.source_id} has no readable text")
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


def _candidate(post: object) -> Candidate | None:
    if not isinstance(post, dict):
        return None
    uri = post.get("uri")
    record = post.get("record")
    author = post.get("author")
    if not isinstance(uri, str) or not isinstance(record, dict) or not isinstance(author, dict):
        return None
    text = record.get("text")
    handle = author.get("handle")
    if not isinstance(text, str) or not text.strip() or not isinstance(handle, str):
        return None
    rkey = uri.rsplit("/", maxsplit=1)[-1]
    url = f"https://bsky.app/profile/{handle}/post/{rkey}"
    return Candidate(
        source=BlueskyCollector.source,
        source_id=uri,
        title=text.strip().splitlines()[0][:200],
        url=url,
        discovery_url=url,
        summary=text.strip(),
        score=float(post.get("likeCount") or 0)
        + 2 * float(post.get("repostCount") or 0)
        + float(post.get("replyCount") or 0),
        published_at=_parse_datetime(record.get("createdAt") or post.get("indexedAt")),
        updated_at=None,
        author=str(author.get("displayName") or handle),
    )


def _post_excerpt(post: object) -> str:
    if not isinstance(post, dict):
        return ""
    record = post.get("record")
    text = str(record.get("text") or "").strip() if isinstance(record, dict) else ""
    embed = post.get("embed")
    external = embed.get("external") if isinstance(embed, dict) else None
    if isinstance(external, dict):
        link = "\n".join(
            str(external.get(key) or "").strip() for key in ("title", "description", "uri")
        ).strip()
        if link:
            text = f"{text}\n{link}".strip()
    return text
