# AI Signal Radar — Implementation Plan

## Ordered delivery

1. **Safety foundation**
   - Add environment configuration for admin access, internal collector authentication, Feishu webhook, `deepseek-v4-pro`, and source credentials.
   - Add password login/session middleware; protect all existing and new costly routes.
   - Make test database selection safe and isolate destructive integration tests.

2. **Signal-domain migration**
   - Add migrations and repositories for source configuration, collection runs, signals, evidence snapshots, analyses, content packages, delivery runs, and sessions.
   - Add canonical URL/content-hash deduplication and an idempotent internal ingestion contract.
   - Preserve the existing table for read-only historical compatibility; do not mutate production history during migration.

3. **Python collector service**
   - Create an independently testable `uv`-managed Python package with normalized adapter and detail-fetch contracts.
   - Implement WaytoAGI detail collection, SkillsMP discovery with GitHub-source verification, GitHub metadata/README/release collection, and the editable Reddit allowlist.
   - Add collection run reporting, bounded retries/backoff, and observe-only mode. Defer X keyword-search crawling to a later task.

4. **Qualification and analysis**
   - Implement hard filters for source/community allowlists, four topic tracks, recency, GitHub usability, canonical URLs, and daily quota.
   - Call `deepseek-v4-pro` only after evidence is stored; persist structured analysis with evidence pointers and uncertainty.
   - Implement dashboard ranking and explicit major-alert eligibility without calling an LLM result “fact checking”.

5. **Private dashboard and Feishu delivery**
   - Replace the generic hot-list first screen with must-read signals, usable tools, content opportunities, and source health.
   - Add editable source/community settings and signal-selection flow.
   - Implement Asia/Shanghai schedules, idempotent rich-text digests, strict 6–8 / 2–3 limits, and capped major alerts.

6. **Content factory**
   - Build the selected-signal evidence view and editable Chinese-first content package.
   - Produce Xiaohongshu, WeChat Official Account, and X drafts, plus X English adaptation.
   - Save platform visual plans and image prompts; explicitly exclude image rendering and automatic publishing.

7. **Migration and production verification**
   - Run collectors in observe-only mode and inspect source health/deduplication.
   - Enable analysis and dashboard, then scheduled digests, then major alerts.
   - Verify production credentials are rotated and no unauthenticated costly route remains.

## Validation gates

- Unit tests for URL canonicalization, eligibility, daily quota, evidence-class preservation, session access, digest sizing, alert caps, and content-package trigger rules.
- Contract tests for collector ingestion and each source adapter using saved fixtures.
- Migration tests from current schema without deleting existing data.
- Go formatting, targeted tests, frontend type-check/build after dependencies are installed, and Python `uv` tests/lint for the collector package.
- Manual staging verification of login, source health, evidence links, 08:00/18:00 rendering, and Feishu webhook delivery to a test bot.

## High-risk checkpoints and rollback

- Do not point tests at production; inspect configuration before every database test.
- Disable a source on rate limit, schema change, or unauthorized response; show degraded status instead of retry storms.
- Disable scheduled delivery or major alerts through configuration without deleting signals.
- Roll back dashboard/API deployment independently of additive migrations; retain evidence and delivery audit rows.
