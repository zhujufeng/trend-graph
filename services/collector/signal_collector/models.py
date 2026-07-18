from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class Candidate:
    source: str
    source_id: str
    title: str
    url: str
    discovery_url: str
    summary: str
    score: float
    published_at: datetime | None
    updated_at: datetime | None
    author: str = ""
    subscribed: bool = False


@dataclass(frozen=True)
class EvidenceDetail:
    source: str
    source_id: str
    source_url: str
    title: str
    excerpt: str
    evidence_class: str
    requires_github_verification: bool
    published_at: datetime | None
    updated_at: datetime | None
