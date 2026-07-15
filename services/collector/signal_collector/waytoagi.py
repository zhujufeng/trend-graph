from html import unescape
from html.parser import HTMLParser

from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class _LinkParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.href: str | None = None
        self.text: list[str] = []
        self.links: list[tuple[str, str]] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag == "a":
            self.href = dict(attrs).get("href")
            self.text = []

    def handle_data(self, data: str) -> None:
        if self.href:
            text = unescape(data).strip()
            if text:
                self.text.append(text)

    def handle_endtag(self, tag: str) -> None:
        if tag == "a" and self.href:
            text = " ".join(self.text).strip()
            if text:
                self.links.append((self.href, text))
            self.href = None
            self.text = []


class _ArticleTextParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.title = ""
        self._in_title = False
        self.parts: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag in {"h1", "title"} and not self.title:
            self._in_title = True

    def handle_data(self, data: str) -> None:
        text = unescape(data).strip()
        if not text:
            return
        if self._in_title and not self.title:
            self.title = text
        self.parts.append(text)

    def handle_endtag(self, tag: str) -> None:
        if tag in {"h1", "title"}:
            self._in_title = False


class WaytoAGICollector:
    source = "waytoagi"
    homepage_url = "https://www.waytoagi.com/zh"

    def __init__(self, client: HTTPClient) -> None:
        self.client = client

    def list_candidates(self, limit: int) -> list[Candidate]:
        parser = _LinkParser()
        parser.feed(self.client.get_text(self.homepage_url))
        candidates: list[Candidate] = []
        seen: set[str] = set()
        for href, text in parser.links:
            if "/zh/blog/" not in href or href in seen:
                continue
            seen.add(href)
            url = href if href.startswith("http") else f"https://www.waytoagi.com{href}"
            candidates.append(
                Candidate(
                    source=self.source,
                    source_id=url.rsplit("/", maxsplit=1)[-1],
                    title=text,
                    url=url,
                    discovery_url=url,
                    summary="",
                    score=float(limit - len(candidates)),
                    published_at=None,
                    updated_at=None,
                )
            )
            if len(candidates) >= limit:
                break
        return candidates

    def fetch_detail(self, candidate: Candidate) -> EvidenceDetail:
        parser = _ArticleTextParser()
        parser.feed(self.client.get_text(candidate.url))
        excerpt = "\n".join(parser.parts[:80]).strip()
        if not excerpt:
            raise ValueError(f"WaytoAGI page {candidate.url} has no readable text")
        return EvidenceDetail(
            source=self.source,
            source_id=candidate.source_id,
            source_url=candidate.url,
            title=parser.title or candidate.title,
            excerpt=excerpt,
            evidence_class="documented_third_party_practice",
            requires_github_verification=False,
            published_at=candidate.published_at,
            updated_at=candidate.updated_at,
        )
