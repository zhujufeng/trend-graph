# Personal Information Radar Contract

## Scope

This contract covers the active product path: topic and source settings, collection, evidence qualification, model analysis, ranking, lifecycle, dashboard responses, and Feishu delivery. Legacy hot-item and graph routes are outside this path.

## Public seams

- `radar.Qualify(signal, evidence, topics, now) QualificationDecision`
- `radar.RankSignals(items, now) []RankedSignal`
- `radar.SelectDigestSignals(items, now, limit) []RankedSignal`
- `radar.BuildDigest(items, now) (Digest, error)`
- `radar.NewCollectionRunner(sourceStore, topicStore, collectorDir, backendURL, secret)`
- `radar.NewAnalysisRunner(signalStore, topicStore, analyzer, modelName)`
- `PATCH /api/radar/signals/:id/lifecycle`
- Collector CLI sources: `dev|github|reddit|bluesky|rss`

## Topics and sources

- A fresh database gets one active `AI` topic. `EnsureDefault` uses an unscoped count so deleting every topic is a durable user choice.
- At most 10 topics may be active. Collection and deterministic qualification read the same active topic rows.
- With no active topics, search-based sources are skipped. An explicit GitHub repository watchlist may still collect Releases.
- Current sources are `dev`, `github`, `reddit`, `bluesky`, and `rss`. `types.RadarSources()` is the allowlist for every active-product signal query.
- Source-specific settings are stored in `source_configs.settings_json`:
  - Reddit: `communities`, excluding `r/all`;
  - GitHub: `repositories`, normalized `owner/repo`, maximum 20;
  - RSS: absolute HTTP(S) `feeds`, maximum 20.
- One source failure records its own failed run and does not stop later sources. A whole collection round cannot overlap another round.

## Collection evidence

- DEV stores full article material as `documented_third_party_practice`.
- GitHub topic search merges repository results across topics. README and release material are `original_documentation`.
- Each explicitly watched GitHub Release uses the release HTML URL as its canonical candidate URL so a later version is a new signal.
- Reddit and Bluesky require `community_discussion` evidence.
- RSS 2.0 and Atom use the entry URL, title, summary/content, author, and publication/update time. Evidence class is `publisher_feed`.
- Candidate failures are reported in the single JSON stdout payload. If all candidates fail, the collector exits non-zero.

## Qualification and analysis

- Recency is 30 days, preferring source update time, then publish time, then creation time.
- Topic `AI` uses the maintained aliases; other topics use case-insensitive direct matching. `ai` must be a token and must not match inside `maintainer`.
- A watched GitHub Release may qualify without active topics, but still requires original documentation.
- The model receives the active topics and may return at most three `matchedTopics`, all from that list. When topics exist, at least one match is required unless the signal came from an explicit repository subscription.
- `valueScore` is an integer from 1 through 5.
- Core evidence, sourced facts, action, and alert-category validation remain mandatory. Tool compatibility and installation fields are required only when the evidence actually describes a tool.
- Daily analysis quota is 30 per Asia/Shanghai day. Qualification happens before spending quota.

## Ranking and lifecycle

- Shared rank formula: `valueScore*10 + recencyBonus + alertBonus`.
- Recency bonus is 8 within 24 hours, 5 within 7 days, 1 within 30 days, otherwise 0. A valid alert adds 20.
- Old analysis JSON without a valid value score defaults to 3 for compatibility.
- Sort by rank descending, then source timestamp descending, then ID descending. After sorting, remove duplicate canonical URLs and normalized titles so the highest-ranked copy wins.
- Lifecycle values are `inbox|saved|done|dismissed`. Startup migrates `new→inbox`, `queued→saved`, and `practiced→done`.
- Dismissed rows are excluded by the shared active query. Lifecycle updates target qualified current-source signals only.
- Content-package creation is optional and requires `qualification=qualified`, `lifecycle_state=done`, frozen evidence, and analysis.

## Delivery

- Digest selection uses the shared ranker, accepts only `inbox` or `saved`, skips `last_delivered_at`, caps each first matched topic at two, and caps the digest at eight.
- Major alerts use the same ranked working set, skip delivered and done signals, and remain capped at three successful alerts per Shanghai day.
- A successful webhook is followed by one SQL transaction that marks the delivery run sent and sets `signals.last_delivered_at` for exactly the delivered IDs.
- A failed webhook calls `Finish(..., failed, ...)` and never marks signals, so the stable delivery key remains retryable.
- The webhook and SQL transaction cannot be atomic. Keep stable idempotency keys and do not add a distributed delivery protocol unless multiple replicas require it.

## Required tests

- Topic matching, AI token boundary, watched Release exception, RSS evidence, recency, and quota short-circuit.
- Analyzer topic-subset and value-score validation.
- Rank formula, highest-copy dedupe, per-topic digest cap, and old JSON fallback.
- Morning-to-evening and alert-to-digest non-repetition; failed send retries without marking signals.
- Source setting normalization and runner argument propagation.
- RSS/Atom parsing, partial failure, GitHub multi-topic merge and watched Release identity.
- Radar API never serializes evidence excerpts.
