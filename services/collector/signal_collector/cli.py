import argparse
import json
import os
from dataclasses import asdict

from .github import GitHubCollector
from .http import UrllibHTTPClient
from .ingestion import BackendIngestionClient
from .reddit import RedditCollector
from .skillsmp import SkillsMPCollector
from .waytoagi import WaytoAGICollector


DEFAULT_REDDIT_COMMUNITIES = "r/LocalLLaMA,r/ClaudeAI,r/ClaudeCode,r/AI_Agents,r/cursor,r/ChatGPTCoding"


def main() -> None:
    parser = argparse.ArgumentParser(description="Collect AI Signal Radar candidates")
    parser.add_argument("--source", choices=["waytoagi", "skillsmp", "github", "reddit"], required=True)
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--query", help="required for SkillsMP and GitHub search")
    parser.add_argument("--communities", help="comma-separated Reddit community allowlist")
    parser.add_argument("--ingest", action="store_true", help="write collected details to the Go backend")
    args = parser.parse_args()

    client = UrllibHTTPClient()
    if args.source == "waytoagi":
        collector = WaytoAGICollector(client)
        candidates = collector.list_candidates(args.limit)
    elif args.source == "skillsmp":
        if not args.query:
            parser.error("--query is required when --source=skillsmp")
        collector = SkillsMPCollector(
            client, os.getenv("SKILLSMP_API_KEY", ""), os.getenv("GITHUB_TOKEN", "")
        )
        candidates = collector.search(args.query, args.limit)
    elif args.source == "github":
        if not args.query:
            parser.error("--query is required when --source=github")
        collector = GitHubCollector(client, os.getenv("GITHUB_TOKEN", ""))
        candidates = collector.search(args.query, args.limit)
    else:
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

    if not args.ingest:
        print(json.dumps([asdict(candidate) for candidate in candidates], ensure_ascii=False, default=str))
        return

    backend_url = os.getenv("BACKEND_URL", "http://127.0.0.1:8080")
    secret = os.getenv("INTERNAL_INGEST_SECRET", "")
    if not secret:
        parser.error("INTERNAL_INGEST_SECRET is required with --ingest")
    backend = BackendIngestionClient(backend_url, secret, client)
    created = 0
    for candidate in candidates:
        if backend.ingest(candidate, collector.fetch_detail(candidate)):
            created += 1
    print(json.dumps({"collected": len(candidates), "created": created}, ensure_ascii=False))


if __name__ == "__main__":
    main()
