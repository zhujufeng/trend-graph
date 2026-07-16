# AI Signal Radar Contract

## Scenario: Qualify, analyze, and digest evidence-backed AI signals

### 1. Scope / Trigger

- Trigger: a collected signal crosses persistence, deterministic qualification, DeepSeek analysis, dashboard, and Feishu digest boundaries.
- Trigger: a backend process with collection enabled starts and must not leave a new dashboard empty until the next cron tick.
- Preserve original evidence and reject non-actionable catalog entries before spending model quota.

### 2. Signatures

- `radar.Qualify(signal store.Signal, evidence store.EvidenceSnapshot, now time.Time) QualificationDecision`
- `(*radar.AnalysisRunner).Run(ctx context.Context, now time.Time) (AnalysisRunResult, error)`
- `radar.NewCollectionRunner(repo, collectorDir, backendURL, secret).Run(ctx) error`
- `(*store.SourceConfigRepo).RecordCollectionRun(store.CollectionRun) error`
- `radar.NewCollectionCron(job func()) (*cron.Cron, error)`
- `(*analyzer.Analyzer).AnalyzeSignal(ctx, analyzer.SignalInput, analyzer.EvidenceInput) (analyzer.SignalAnalysisOutput, error)`
- `(*analyzer.Analyzer).GenerateContentPackage(ctx, analyzer.SignalInput, analyzer.EvidenceInput, analysisJSON) (analyzer.ContentPackageDraft, error)`
- `radar.BuildDigest(items []store.RadarSignal, now time.Time) (Digest, error)`
- `(*radar.DeliveryService).SendDigest(ctx, now) error`
- `(*radar.DeliveryService).SendMajorAlerts(ctx, now) error`
- `notify.FeishuNotifier.Notify(ctx, notify.FeishuPost) error`
- Content API: `POST /api/radar/signals/:id/content-packages`, `GET|PUT /api/content-packages/:id`, `POST /api/content-packages/:id/approve`.
- Practice API: `PATCH /api/radar/signals/:id/lifecycle` with `{state:"new"|"queued"|"practiced"|"dismissed"}`.
- `(*store.SignalRepo).UpdateLifecycleState(id int64, state string) error` updates qualified radar signals only.
- Collector command: `uv run --no-sync python -m signal_collector.cli --source {dev,github,reddit,bluesky}`; DEV/GitHub/Bluesky also require `--query`.
- Runtime environment: `COLLECTOR_DIR` defaults to `../services/collector`; the Go runner supplies `BACKEND_URL` and `INTERNAL_INGEST_SECRET` directly to the child process.
- Runtime environment: `BACKGROUND_JOBS_ENABLED` defaults to `true`. When `false`, the server still loads DeepSeek and serves user-triggered content generation, but creates no collection/digest cron and runs no initial collection or pending analysis.
- Python seams: `search()` or `list_candidates()` returns `Candidate`; `fetch_detail(Candidate)` returns `EvidenceDetail`.
- Python deterministic shortlist: `qualification.shortlist(candidates, now)` runs before detail fetch and ingestion.
- DB transition: `signals.qualification=pending -> qualified|rejected`; qualified writes `signal_analyses` and the signal state in one transaction.

### 3. Contracts

- Accepted sources are `dev`, `github`, `reddit`, and `bluesky`; X is deferred and Linux.do, WaytoAGI, SkillsMP, and Bilibili are excluded.
- `types.RadarSources()` is the persistence-query allowlist. Dashboard, pending-analysis, digest/alert, and content-package lookups must filter through it so retired source rows can remain for audit without re-entering the product.
- Recency is 30 days, preferring source update time, then publish time, then creation time.
- DEV search splits a comma/whitespace-separated query into tags, deduplicates article IDs across tag results, and fetches `body_markdown` from the article-detail endpoint. Successful evidence is `documented_third_party_practice`.
- GitHub search uses repository metadata; detail uses the JSON Contents API and decodes its Base64 `content`, then appends the latest release when present.
- GitHub detail reuses the search candidate title and must not refetch repository metadata. One search plus README and release detail is at most `1 + 2N` GitHub API requests for `N` candidates.
- GitHub requires `original_documentation` plus a usable setup/run indicator; Reddit and Bluesky require `community_discussion`.
- Reddit requires `REDDIT_CLIENT_ID` and `REDDIT_CLIENT_SECRET`, requests application-only OAuth, calls `oauth.reddit.com`, and only reads `REDDIT_COMMUNITIES` or the explicit `--communities` allowlist. `r/all` is forbidden at both configuration and collector boundaries.
- Bluesky uses the public official AppView endpoints at `https://api.bsky.app/xrpc`: `app.bsky.feed.searchPosts` for discovery and `app.bsky.feed.getPostThread` for detail. The `public.api.bsky.app` alias is not used because it can be blocked by intermediary CDNs even when the official `api.bsky.app` endpoint succeeds.
- `GITHUB_TOKEN` is optional; DEV and Bluesky require no key. A missing Reddit credential is a degraded-source error, never permission to scrape logged-out pages.
- `DEEPSEEK_MODEL` defaults to `deepseek-v4-pro`; the daily Asia/Shanghai model-analysis quota is 30 newly analyzed signals.
- Structured JSON preserves `evidenceClass` and includes facts with source URLs, `whatChanged`, audience, practical use, prerequisites, pain point, action, content opportunity, uncertainty, and alert decision.
- Signal analysis sends at most 12,000 evidence runes (first 8,000 and last 4,000) so README setup and appended release notes survive without modifying stored evidence. Structured output allows 2,400 tokens and rejects `finish_reason=length` before JSON parsing.
- GitHub analysis requires evidence-backed `toolType`, `compatibleClients`, and `installation`; never translate “universal” into an unsupported one-click claim.
- `alertEligible=true` requires `alertReason` and exactly one category: `major_release`, `material_efficiency_gain`, `corroborated_pain_point`, or `source_backed_content_opportunity`.
- Schedules use Asia/Shanghai: collection `0 */3 * * *`, pre-digest refresh `40 7,17 * * *`, digest `0 8,18 * * *`.
- Go owns scheduling and source health; Python owns source-specific collection and authenticated ingestion. Disabled sources never start a process.
- A successful collector command writes exactly one JSON object to stdout, including `failed` and `failures` for skipped candidates. Partial-failure diagnostics must not use stderr because Go reads child output with `CombinedOutput`; if every shortlisted candidate fails, the command exits non-zero.
- A failed source records a failed `collection_runs` row and updates `source_configs.last_failure`, but does not prevent later enabled sources from running. Success updates `last_success_at` and clears `last_failure` in the same transaction as its audit row.
- The collection runner must prevent overlapping whole rounds. Cron wrappers alone are insufficient because the regular and pre-digest entries have separate overlap locks.
- When collection is configured, backend startup binds the HTTP listener first and then starts one asynchronous collection round. This guarantees the Python child can call the internal ingestion route; scheduled rounds remain unchanged.
- Pending signals may appear only in a clearly labeled `最新采集（待分析）` section. They must never be promoted into `今日必读`, tools, content opportunities, digests, or alerts before qualification.
- A digest contains at most 8 qualified analyzed signals and 3 content opportunities, preserving each original URL. Feishu uses `msg_type=post` with `zh_cn` rich-post content.
- Digest delivery uses one idempotency key per Shanghai date/hour. Major alerts use one key per signal and are capped at three successful sends per Shanghai day. Failed sends and stale `running` deliveries older than 15 minutes are retryable.
- `DIGEST_ENABLED` and `MAJOR_ALERTS_ENABLED` independently disable those jobs without deleting stored history.
- Content generation is user-triggered only. It freezes `evidenceSnapshotId`, stores separate strategy/Xiaohongshu/WeChat/X/visual JSON artifacts, and never renders images or publishes externally.
- Optional model wiring must pass a true nil interface when DeepSeek is absent. A typed nil `*analyzer.Analyzer` inside `contentPackageGenerator` is non-nil and will panic when invoked.
- `BACKGROUND_JOBS_ENABLED=false` does not disable authentication, the internal ingestion route, or content APIs; it disables only automatic collection, batch analysis, alerts, and digest schedules.
- Practice state reuses `signals.lifecycle_state`: `new` is untriaged, `queued` is selected for practice, `practiced` is user-confirmed, and `dismissed` is hidden. Content creation requires both `qualification=qualified` and `lifecycle_state=practiced`.
- Each platform keeps server-supplied source links; X has separate Chinese and English drafts. Non-`user_verified` packages reject generated or edited first-person test claims.
- Approval saves the latest edits before setting `status=approved`; approved packages are immutable through the edit endpoint.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Unsupported or non-AI source material | Reject before model invocation with a stable reason. |
| A retired source still has historical `signals` rows | Preserve the rows, but exclude them from dashboard, pending analysis, delivery, and content-package lookup. |
| `ai` appears only inside another word, such as `maintainer` | Do not count it as the AI topic token. |
| Missing evidence or wrong evidence class | Reject; do not create `signal_analyses`. |
| GitHub README or referenced `SKILL.md` is missing | Treat the candidate as unusable; do not downgrade to catalog evidence. |
| DEV article detail has no body or description | Fail that candidate; title-only evidence is forbidden. |
| Bluesky search/thread response has no post text | Fail that candidate; do not synthesize discussion evidence from metadata. |
| Reddit credentials are missing or OAuth has no access token | Fail the source run explicitly; do not fall back to `reddit.com/*.json` or `r/all`. |
| One enabled collector exits non-zero or returns invalid JSON | Record that source as failed, continue later sources, and return an aggregate error after the round. |
| A second schedule fires while any collection round is active | Skip the entire overlapping round without launching another Python process. |
| Backend cannot bind its HTTP listener | Exit without starting the initial collector; never launch a child that cannot ingest. |
| `INTERNAL_INGEST_SECRET` is empty | Do not register ingestion, collection schedules, or the initial collection round. |
| `BACKGROUND_JOBS_ENABLED=false` | Keep manual content generation available; report zero schedulers and do not launch background work. |
| Content generation has no configured model | Return HTTP 503 JSON `content model is not configured`; never panic or return an empty 500. |
| Reddit allowlist contains `r/all`, duplicates, or invalid names | Drop forbidden/invalid values and deduplicate before any request. |
| Daily count is already 30 | Return zero remaining and do not list candidates or call the model. |
| Model returns invalid JSON, omits core fields, or changes `evidenceClass` | Return an error; do not mark the signal qualified. |
| Model returns `finish_reason=length` | Return a truncation error before parsing; leave the signal pending. |
| One shortlisted collector candidate fails detail fetch or ingestion | Continue later candidates and report the failure inside the final JSON object. |
| Every shortlisted collector candidate fails | Exit non-zero so the source health row records a failure. |
| Digest analysis JSON is invalid | Return an error; do not send a partial digest. |
| Feishu returns non-200 | Return an error so delivery can be retried and recorded. |
| Feishu returns HTTP 200 with non-zero business `code` | Treat it as failure and record a retryable delivery. |
| Alert lacks a supported category or reason | Reject the model output; do not send. |
| Content creation targets pending/rejected/missing-evidence signal | Return HTTP 409 without calling the model. |
| Lifecycle payload has an unsupported state | Return HTTP 400 without updating the signal. |
| Lifecycle target is pending, rejected, retired, or missing | Return HTTP 404 without updating it. |
| Content creation targets a qualified but unpracticed signal | Return HTTP 409 without calling the model. |
| Content edit removes source links or adds first-person testing to third-party evidence | Return HTTP 400 and keep the stored draft. |
| Default `go test ./...` reaches a live source | Gate the misclassified test behind `RUN_LIVE_TESTS=1`. |

### 5. Good / Base / Bad Cases

- Good: a recent GitHub README with install/use evidence is analyzed once, marked qualified transactionally, and appears with its original link and concrete action.
- Good: a recent DEV implementation article is shortlisted by tag, its full Markdown body is preserved, and it is labeled documented third-party practice.
- Good: a recent Bluesky post preserves its original profile/post URL and thread context while remaining community discussion.
- Base: a GitHub repository without README is omitted while other candidates can continue; it is not model-analyzed.
- Base: one GitHub candidate fails while another succeeds; stdout remains valid JSON and the source round succeeds with `failed=1`.
- Base: Reddit is degraded because credentials are missing; DEV, GitHub, and Bluesky still run and receive their own audit rows.
- Base: historical WaytoAGI/SkillsMP signals remain in PostgreSQL but disappear from every current product query.
- Good: a fresh backend binds its port, serves health/login, and immediately starts one source-config-driven collection round.
- Good: the user selects one qualified signal, edits three platform drafts and image prompts, saves, then explicitly approves it.
- Good: a qualified signal moves from `new` to `queued` to `practiced`, survives refresh, and only then exposes content generation.
- Good: local HTTP starts with DeepSeek configured and `BACKGROUND_JOBS_ENABLED=false`; health reports zero schedulers and one manual request creates a draft content package.
- Good: a failed 08:00 webhook attempt is recorded and can retry without duplicating a successful digest.
- Base: model analysis is not configured; collected pending signals remain visible only under `最新采集（待分析）`.
- Bad: sending pending/rejected items as “今日必读”, consuming a model call before qualification, or claiming first-person verification for third-party evidence.
- Bad: treating a DEV title or Bluesky search hit as sufficient evidence, or silently using an unauthenticated Reddit scraper.
- Bad: generating packages for every candidate, approving unsaved browser edits, or letting third-party evidence become “我实测”.

### 6. Tests Required

- Table-test qualification reasons, 30-day timestamps, evidence classes, and AI token boundaries.
- Test quota exhaustion without store candidate reads or model calls; test successful analysis persistence fields and token usage.
- Test analyzer prompts/structured response with a fake AI client; never call DeepSeek in default tests.
- Test over-limit multibyte evidence preserves both ends, uses the bounded prompt, and rejects a fake `finish_reason=length` response.
- Test schedule next-runs from an Asia/Shanghai timestamp and digest caps/link preservation.
- Test Feishu payload through an in-memory `http.RoundTripper`; do not bind ports or call a webhook.
- Test HTTP-200 Feishu business errors, digest idempotency, the three-alert cap, and explicit alert categories.
- Test content creation requires qualified frozen evidence, links survive every platform payload, and third-party evidence cannot become first-person testing.
- Test `BACKGROUND_JOBS_ENABLED` defaults true and parses explicit false. Live manual-generation smoke must assert health `schedulers=0`, missing-model HTTP 503, and configured-model HTTP 201.
- Test lifecycle state validation and qualified-only persistence; test that queued content creation returns conflict while practiced creation succeeds.
- Test that the radar list response keeps evidence provenance but never serializes `excerpt`; list queries must select only evidence metadata while single-signal reads retain the frozen body.
- Frontend render tests must assert rejected signals never enter outcome sections.
- Frontend render tests must assert pending signals are visible in `最新采集（待分析）` while remaining absent from qualified outcome sections.
- Collector contract tests must cover DEV tag deduplication/full-body detail, GitHub README/release decoding, Reddit OAuth/allowlist/detail parsing, Bluesky search/thread parsing, author preservation, and authenticated ingestion without network access.
- GitHub tests must assert no redundant repository metadata request; CLI tests must assert one failed candidate does not block the next, successful stdout remains parseable JSON, and all-candidate failure is non-zero.
- Runner tests must inject the process boundary and assert disabled-source skipping, Reddit allowlist arguments, per-source success/failure audit records, continuation after failure, and whole-round overlap prevention.
- Store/API tests must assert that retired-source rows cannot enter dashboard, analysis, delivery, or content generation.
- Live smoke tests are read-only and separate: a current DEV article body, a known public GitHub README/release, and a current Bluesky thread. Reddit live success requires real eligible OAuth credentials.
- Legacy crawler network tests run only with `RUN_LIVE_TESTS=1`; the default Go suite is deterministic and offline.

### 7. Wrong vs Correct

#### Wrong

```go
output, _ := model.AnalyzeSignal(ctx, signal, evidence) // spends quota before qualification
notifier.Notify(ctx, signal.OriginalTitle)              // loses evidence and original link
```

#### Correct

```go
decision := radar.Qualify(signal, evidence, now)
if decision.Eligible {
    output, err := model.AnalyzeSignal(ctx, analyzer.SignalInput{/* ... */}, analyzer.EvidenceInput{/* ... */})
    // Persist output plus qualification in one transaction, then build a capped rich digest.
}
```

```python
# Wrong: list titles or all-site scraping become evidence.
detail = EvidenceDetail(excerpt=candidate.title, evidence_class="documented_third_party_practice")
client.get_json("https://www.reddit.com/r/all/new.json")

# Correct: fetch source detail and require OAuth allowlist access.
detail = dev.fetch_detail(dev_candidate)
thread = bluesky.fetch_detail(bluesky_candidate)
reddit = RedditCollector(client, client_id, client_secret, approved_communities)
```

```python
# Wrong: stderr corrupts the successful JSON read through CombinedOutput.
print(f"skipped: {exc}", file=sys.stderr)

# Correct: partial failures stay in the single stdout JSON payload.
print(json.dumps({"created": created, "failed": failed, "failures": failures}))
```

```go
// Wrong: separate cron entry locks can still overlap with each other.
scheduler.AddFunc(regularSpec, runner.Run)
scheduler.AddFunc(preDigestSpec, runner.Run)

// Correct: CollectionRunner.Run owns a shared non-blocking round mutex and
// each source result is recorded before processing continues.
if !runner.runMu.TryLock() {
    return nil
}
defer runner.runMu.Unlock()
```

```go
// Wrong: qualification alone bypasses the user's practice step.
if signal.Qualification == "qualified" { generateContent() }

// Correct: content generation follows explicit personal practice.
if signal.Qualification == "qualified" && signal.LifecycleState == "practiced" { generateContent() }
```

```go
// Wrong: converting a nil pointer to an interface makes the interface non-nil.
api.NewContentPackageHandler(signalRepo, an).Register(privateAPI)

// Correct: pass literal nil unless the concrete analyzer exists.
contentHandler := api.NewContentPackageHandler(signalRepo, nil)
if an != nil {
    contentHandler = api.NewContentPackageHandler(signalRepo, an)
}
```

```go
// Wrong: the dashboard pays to load and serialize source documents it never displays.
db.First(&evidence)
response.Evidence = &evidence

// Correct: lists select provenance metadata; full evidence is read only for one signal's analysis/content flow.
db.Select("id", "signal_id", "source_url", "evidence_class", "title", "captured_at").First(&evidence)
```

```go
// Wrong: a boolean lets the model invent its own definition of “major”.
if analysis.AlertEligible { sendAlert() }

// Correct: validate a closed category set, then enforce DB-backed caps/idempotency.
if analyzer.ValidAlertCategory(analysis.AlertCategory) {
    delivery.SendMajorAlerts(ctx, now)
}
```

```tsx
// Wrong: approval loses unsaved editor state.
await approveContentPackage(draft.id)

// Correct: persist the evidence-linked artifacts, then approve that saved version.
const saved = await updateContentPackage(draft)
await approveContentPackage(saved.id)
```

```go
// Wrong: start collection before the ingestion endpoint is listening.
go collectionRunner.Run(context.Background())
return r.Run(addr)

// Correct: bind first, then collect asynchronously through the live endpoint.
listener, err := net.Listen("tcp", addr)
if err != nil { return err }
go collectionRunner.Run(context.Background())
return r.RunListener(listener)
```
