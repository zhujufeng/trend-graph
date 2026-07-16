# AI Signal Radar — Technical Design

## Boundary

Keep the existing React application as the private dashboard and retain the Go service as the system of record for the API, PostgreSQL domain model, authentication, content factory, Feishu delivery, and scheduled digest decisions. Add a Python collector package managed by `uv` for source-specific acquisition and normalization. This introduces Python where it has the strongest fit without creating two competing owners of the data model.

The source design supports DEV Community, GitHub, the approved Reddit community allowlist, Bluesky, and a future X adapter. Linux.do, WaytoAGI, SkillsMP, Bilibili, and broad generic feeds are excluded. X remains important, but its intended keyword-search crawling is deferred from the current delivery and must not block the core source pipeline.

## Components and data flow

```text
Python collector (uv)
  list fetch -> deterministic qualification -> detail fetch -> signed internal ingestion
                                                               |
React dashboard <- authenticated Go API <- PostgreSQL <- signal analysis / content factory
                                       |
                                  Feishu rich-text delivery
```

1. The collector records a per-source run and gathers lightweight list candidates.
2. It canonicalizes URLs, applies the source allowlist, topic tracks, recency, and GitHub usability gates.
3. Shortlisted candidates receive detail collection: DEV article body, GitHub README/release metadata, Reddit post/discussion detail, or Bluesky thread context.
4. The Go ingestion boundary transactionally deduplicates and stores the signal plus evidence snapshot.
5. A deterministic rank selects at most 30 newly eligible candidates daily for `deepseek-v4-pro` structured analysis.
6. The API ranks analyzed signals for dashboard, digest, major-alert review, and a user-initiated content package.
7. At 08:00 and 18:00 Asia/Shanghai, Go renders and sends a 6–8 signal / 2–3 content-opportunity Feishu rich-text digest. A fresh run occurs at 07:40 and 17:40; regular source collection runs every three hours.

## Data model

New tables are preferred over overloading the current `hot_items` schema:

- `source_configs`: source enablement, editable Reddit allowlist, collection parameters, last successful run.
- `collection_runs`: source, timing, result counts, status, and failure reason.
- `signals`: canonical URL identity, source, source timestamps, score, qualification state, and lifecycle state.
- `evidence_snapshots`: source URL, extracted text/excerpt, capture time, content hash, and evidence class.
- `signal_analyses`: structured `deepseek-v4-pro` output, fact/judgment/action fields, uncertainty, model/version, and token/use metadata.
- `content_packages`: selected signal, evidence version, strategy, three platform drafts, visual plans/prompts, editable status, and user approval state.
- `delivery_runs`: digest/alert idempotency, selected signal IDs, response status, and error details.
- `admin_sessions`: hashed session tokens, expiry, and revocation metadata.

`hot_items` remains readable during migration but no longer drives the new dashboard. A one-off import may create unqualified historical signals only after canonical URL deduplication; it must not trigger model analysis or notifications.

## Source adapter contracts

Each Python adapter returns a normalized list candidate and, only when shortlisted, a detail payload. It cannot write to PostgreSQL directly. The Go internal ingestion endpoint validates an internal shared secret, schema, canonical URL, source configuration, and content size before storing it.

- DEV Community: official Forem API tag discovery followed by the full article body. Qualified evidence is documented third-party practice, never a user-verified claim.
- GitHub: search/repository metadata plus README and release information; use a server-side token when available, while gracefully reporting rate-limit/degraded state.
- Reddit: only configured communities; production collection should use an approved API credential path. Without valid credentials, mark the source degraded rather than silently falling back to an unreliable all-site scraper.
- Bluesky: official public AppView keyword search followed by thread retrieval. Preserve post/thread links and classify the result as community discussion.
- X: adapter interface only in the current delivery. A later phase may evaluate a read-only keyword-search crawler; it must never perform account interactions and must expose degraded status when its session or acquisition method fails.

## Analysis and content contracts

Analysis is JSON, not free-form truth claims. Required fields include evidence class, factual statements with source pointers, interpretation, uncertainty, audience, prerequisites, concrete action, content opportunity, and alert eligibility. The model may summarize evidence but cannot upgrade community discussion to official fact or use first-person test claims.

Content generation runs only from an explicitly selected signal and its frozen evidence snapshot. It produces Chinese-first Xiaohongshu, WeChat Official Account, and X drafts; X includes an adapted English counterpart. Every image proposal stores an editable visual plan and a reproducible prompt. No image generation or external publishing occurs in the first release.

## Security and operations

- Replace public high-cost endpoints with authenticated admin routes protected by a password-derived server-side session; all secrets stay in environment variables.
- Rotate the exposed database password before deployment, limit PostgreSQL network access, and use TLS/private connectivity where available.
- Split destructive integration tests from default test targets and reject a production-looking database URL in test setup.
- Make all collection, analysis, and delivery operations idempotent with canonical URL, content-hash, and delivery-run keys.
- Expose source health, last success, failures, and model-analysis quota in the dashboard.

## Rollout and rollback

Deploy schema migrations first, with new tables additive and old `hot_items` untouched. Deploy authentication before enabling any crawl/analysis route. Run collectors in observe-only mode, then enable analysis, dashboard replacement, scheduled digests, and finally major alerts. Each feature flag can be disabled independently; database rollback is additive and does not delete historical data.
