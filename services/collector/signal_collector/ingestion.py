from .http import HTTPClient
from .models import Candidate, EvidenceDetail


class BackendIngestionClient:
    def __init__(self, base_url: str, secret: str, client: HTTPClient) -> None:
        self.endpoint = f"{base_url.rstrip('/')}/internal/ingest/signals"
        self.secret = secret
        self.client = client

    def ingest(self, candidate: Candidate, detail: EvidenceDetail) -> bool:
        response = self.client.post_json(
            self.endpoint,
            {
                "source": candidate.source,
                "originalUrl": candidate.url,
                "originalTitle": candidate.title,
                "author": "",
                "score": candidate.score,
                "publishedAt": _iso(candidate.published_at),
                "updatedAt": _iso(candidate.updated_at),
                "evidenceUrl": detail.source_url,
                "evidenceTitle": detail.title,
                "evidenceClass": detail.evidence_class,
                "evidenceExcerpt": detail.excerpt,
            },
            {"X-Internal-Ingest-Secret": self.secret},
        )
        created = response.get("created")
        if not isinstance(created, bool):
            raise ValueError("backend ingestion response must contain boolean created")
        return created


def _iso(value):
    return value.isoformat() if value else None
