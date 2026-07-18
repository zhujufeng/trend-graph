import html
import re
from datetime import UTC, datetime
from email.utils import parsedate_to_datetime
from urllib.parse import urljoin, urlparse
from xml.etree import ElementTree

from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class RSSCollector:
    source = "rss"

    def __init__(self, client: HTTPClient, feeds: list[str]) -> None:
        self.client = client
        self.feeds = [feed.strip() for feed in feeds if _http_url(feed.strip())]
        if not self.feeds:
            raise ValueError("RSS feed list must contain at least one HTTP(S) URL")
        self.failures: list[str] = []
        self._evidence: dict[str, str] = {}

    def list_candidates(self, limit: int = 20) -> list[Candidate]:
        limit = max(1, min(limit, 100))
        found: dict[str, Candidate] = {}
        successful = 0
        for feed_url in self.feeds:
            try:
                entries = _parse_feed(self.client.get_text(feed_url), feed_url)
                successful += 1
            except Exception as exc:
                self.failures.append(f"{feed_url}: {exc}")
                continue
            for candidate, evidence in entries:
                found[candidate.url] = candidate
                self._evidence[candidate.source_id] = evidence
        if successful == 0:
            raise RuntimeError(f"all RSS feeds failed: {'; '.join(self.failures)}")
        return sorted(
            found.values(),
            key=lambda item: item.updated_at or item.published_at or datetime.min.replace(tzinfo=UTC),
            reverse=True,
        )[:limit]

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        excerpt = self._evidence.get(candidate.source_id, "").strip()
        if not excerpt:
            raise ValueError(f"RSS entry {candidate.source_id} has no readable content")
        return EvidenceDetail(
            source=self.source,
            source_id=candidate.source_id,
            source_url=candidate.url,
            title=candidate.title,
            excerpt=excerpt,
            evidence_class="publisher_feed",
            requires_github_verification=False,
            published_at=candidate.published_at,
            updated_at=candidate.updated_at,
        )


def _parse_feed(document: str, feed_url: str) -> list[tuple[Candidate, str]]:
    root = ElementTree.fromstring(document)
    root_name = _local(root.tag)
    if root_name == "feed":
        nodes = [node for node in root if _local(node.tag) == "entry"]
    else:
        channel = next((node for node in root.iter() if _local(node.tag) == "channel"), root)
        nodes = [node for node in channel if _local(node.tag) == "item"]
    result: list[tuple[Candidate, str]] = []
    for node in nodes:
        title = _text(node, "title")
        link = urljoin(feed_url, _entry_link(node))
        if not title or not _http_url(link):
            continue
        content = _text(node, "encoded", "content", "summary", "description")
        entry_id = _text(node, "id", "guid") or link
        published = _date(_text(node, "published", "pubDate", "date"))
        updated = _date(_text(node, "updated", "modified"))
        source_id = f"{feed_url}#{entry_id}"
        evidence = _plain_text(content)
        result.append(
            (
                Candidate(
                    source="rss",
                    source_id=source_id,
                    title=_plain_text(title),
                    url=link,
                    discovery_url=feed_url,
                    summary=evidence,
                    score=0,
                    published_at=published,
                    updated_at=updated,
                    author=_text(node, "author", "creator", "name"),
                ),
                evidence,
            )
        )
    return result


def _local(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


def _text(node: ElementTree.Element, *names: str) -> str:
    wanted = set(names)
    for child in node.iter():
        if _local(child.tag) in wanted:
            value = "".join(child.itertext()).strip()
            if value:
                return value
    return ""


def _entry_link(node: ElementTree.Element) -> str:
    for child in node:
        if _local(child.tag) != "link":
            continue
        href = child.attrib.get("href", "").strip()
        if href and child.attrib.get("rel", "alternate") in {"", "alternate"}:
            return href
        if child.text and child.text.strip():
            return child.text.strip()
    return ""


def _plain_text(value: str) -> str:
    return " ".join(html.unescape(re.sub(r"<[^>]+>", " ", value)).split())


def _http_url(value: str) -> bool:
    parsed = urlparse(value)
    return parsed.scheme in {"http", "https"} and bool(parsed.netloc)


def _date(value: str) -> datetime | None:
    if not value:
        return None
    try:
        parsed = parsedate_to_datetime(value)
    except (TypeError, ValueError):
        try:
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        except ValueError:
            return None
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=UTC)
    return parsed
