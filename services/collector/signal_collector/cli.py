import argparse
import json
import os
from dataclasses import asdict

from .bluesky import BlueskyCollector
from .dev import DEVCollector
from .github import GitHubCollector
from .http import UrllibHTTPClient
from .ingestion import BackendIngestionClient
from .qualification import shortlist
from .reddit import RedditCollector
from .rss import RSSCollector


DEFAULT_REDDIT_COMMUNITIES = "r/LocalLLaMA,r/ClaudeAI,r/ClaudeCode,r/AI_Agents,r/cursor,r/ChatGPTCoding"


def _split(value: str | None) -> list[str]:
    return [item.strip() for item in (value or "").split(",") if item.strip()]


def main() -> None:
    parser = argparse.ArgumentParser(description="Collect AI Signal Radar candidates")
    parser.add_argument("--source", choices=["dev", "github", "reddit", "bluesky", "rss"], required=True)
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--query", help="required for DEV, GitHub, and Bluesky search")
    parser.add_argument("--topics", help="comma-separated active topics used for candidate filtering")
    parser.add_argument("--communities", help="comma-separated Reddit community allowlist")
    parser.add_argument("--repositories", help="comma-separated GitHub repositories to watch for releases")
    parser.add_argument("--feeds", help="comma-separated RSS/Atom feed URLs")
    parser.add_argument("--ingest", action="store_true", help="write collected details to the Go backend")
    args = parser.parse_args()

    client = UrllibHTTPClient()
    if args.source == "dev":
        if not args.query:
            parser.error("--query is required when --source=dev")
        collector = DEVCollector(client)
        candidates = collector.search(args.query, args.limit)
    elif args.source == "github":
        collector = GitHubCollector(client, os.getenv("GITHUB_TOKEN", ""))
        candidates = collector.search(args.query, args.limit) if args.query else []
        repositories = _split(args.repositories)
        if repositories:
            candidates.extend(collector.list_releases(repositories, args.limit))
        candidates = list({candidate.url: candidate for candidate in candidates}.values())[: args.limit]
        if not candidates and not args.query and not repositories:
            parser.error("--query or --repositories is required when --source=github")
    elif args.source == "bluesky":
        if not args.query:
            parser.error("--query is required when --source=bluesky")
        collector = BlueskyCollector(client)
        candidates = collector.search(args.query, args.limit)
    elif args.source == "reddit":
        communities = (args.communities or os.getenv("REDDIT_COMMUNITIES", DEFAULT_REDDIT_COMMUNITIES)).split(",")
        client_id = os.getenv("REDDIT_CLIENT_ID", "")
        client_secret = os.getenv("REDDIT_CLIENT_SECRET", "")
        if not client_id or not client_secret:
            parser.error("REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET are required when --source=reddit")
        collector = RedditCollector(
            client,
            client_id,
            client_secret,
            communities,
        )
        candidates = collector.list_candidates(args.limit)
    else:
        feeds = _split(args.feeds)
        if not feeds:
            parser.error("--feeds is required when --source=rss")
        collector = RSSCollector(client, feeds)
        candidates = collector.list_candidates(args.limit)

    if not args.ingest:
        print(json.dumps([asdict(candidate) for candidate in candidates], ensure_ascii=False, default=str))
        return

    backend_url = os.getenv("BACKEND_URL", "http://127.0.0.1:8080")
    secret = os.getenv("INTERNAL_INGEST_SECRET", "")
    if not secret:
        parser.error("INTERNAL_INGEST_SECRET is required with --ingest")
    backend = BackendIngestionClient(backend_url, secret, client)
    created = 0
    failed = 0
    processed = 0
    failures: list[str] = list(getattr(collector, "failures", []))
    topics = _split(args.topics) or ["AI"]
    shortlisted = shortlist(candidates, topics)
    for candidate in shortlisted:
        try:
            if backend.ingest(candidate, collector.fetch_detail(candidate)):
                created += 1
            processed += 1
        except Exception as exc:
            failed += 1
            failures.append(f"{candidate.source_id}: {exc}")
    if shortlisted and processed == 0:
        raise RuntimeError(f"all shortlisted candidates failed: {'; '.join(failures)}")
    print(json.dumps({"collected": len(candidates), "shortlisted": len(shortlisted), "created": created, "failed": failed, "failures": failures}, ensure_ascii=False))


if __name__ == "__main__":
    main()
