# AI 信号雷达改造

## Goal

Turn trend-graph from a generic multi-source hot-list into a personal AI signal radar that helps the user avoid missing actionable AI developments and turn qualified signals into practical experiments or Xiaohongshu content ideas.

The product is for one user first. It is not a general news reader, brand-monitoring product, or broad public-opinion system.

Its creator positioning is an evidence-led, practical AI efficiency practitioner: turn relevant releases and documented community practice into reproducible Chinese workflows and content rather than reposting hype.

## Confirmed Facts

- The user follows X/Twitter, GitHub, and Reddit every day for AI developments, practical skills, agent techniques, and real user discussions. Linux.do discussions are judged too noisy and low-value for this workflow.
- The user wants to evaluate WaytoAGI and SkillsMP as replacements for Linux.do because they surface curated AI knowledge/tools, agent skills, and learnable Vibe Coding work.
- The user also creates Xiaohongshu content about AI applications, skills, and agent usage, and wants qualified signals to become source material for posts.
- Hacker News, Bing, and broad general-news aggregation are noise for this workflow and should not be part of the first product experience.
- The user wants two scheduled Feishu Webhook digests at 08:00 and 18:00 Asia/Shanghai, plus immediate alerts only for genuinely major signals.
- A signal that cannot be tried, learned from, used in a workflow, or translated into a credible content angle is not useful enough to surface.
- The current deployed database has 32 non-deleted hot items: 31 from HN and one from Weibo. Nineteen records duplicate the same `(source, URL)`. It has no active keyword configuration or crawl-run history.
- The current public API has no authentication, and the publicly reachable PostgreSQL credentials supplied in conversation must be rotated and network-restricted before implementation touches production.
- The user wants the product to turn qualified AI signals into a complete, editable content package, including Chinese copy and image-generation prompts, rather than stopping at Xiaohongshu topic suggestions.
- The user wants eventual multi-platform content support and permits Python for future collection/processing services, managed with the local `uv` workflow. The redesign is not constrained to the existing Go implementation.
- The first content platforms are Xiaohongshu, WeChat Official Account, and X. The product must create reviewable packages for all three, but need not automate external publishing in the first version.
- `jimliu/baoyu-skills` was reviewed as a workflow reference. Its staged, artifact-preserving creation flow and explicit human confirmation before rendering or publishing are useful patterns; its files, credentials, and unofficial browser/cookie integrations are not to be copied into this product by default.

## Requirements

### R1 — AI-only source experience

The first version must focus on AI-related content only, with sources serving distinct roles:

- WaytoAGI: Chinese curated AI knowledge, tools, cases, and learnable Vibe Coding work. Its items require the same evidence and actionability qualification as any other source.
- SkillsMP: discovery index for public `SKILL.md` workflows. Its catalog entry is a discovery lead, not a quality endorsement; the system must return to the linked GitHub repository to inspect the actual skill, maintenance, setup, and permissions before qualification.
- X/Twitter: an important future source for first-party announcements, developer/researcher signals, and documented real-world practice. The intended acquisition mode is keyword-search crawling rather than author-only tracking, but X collection is deferred from the current delivery. When introduced, posts must preserve their original links and evidence class rather than being treated as user-verified truth.
- GitHub: directly usable agent, skill, and MCP tooling. Popularity alone is insufficient: selected items need clear installation/use instructions, a concrete use case, and evidence of current maintenance.
- Reddit: real user discussion, pain points, adoption patterns, and validation.

Reddit must use an editable community allowlist rather than `r/all`. The initial list is `r/LocalLLaMA`, `r/ClaudeAI`, `r/ClaudeCode`, `r/AI_Agents`, `r/cursor`, and `r/ChatGPTCoding`.

Linux.do is explicitly excluded from all collection, scheduling, dashboard, and notification paths.

The homepage must not lead with generic hot-listing or unrelated sources.

The default qualification tracks are: major AI model/platform releases; directly usable agents, skills, MCP servers, and automation tools; AI coding and personal-efficiency practice; and AI content creation or Xiaohongshu operations methods. These are defaults, not a claim that every item in a source is relevant.

### R2 — Actionability-first qualification

Each surfaced item must receive an explainable qualification result rather than a generic AI summary. The product must identify, at minimum:

- what changed and its original source;
- who can use it and a concrete practical use;
- effort or prerequisites to try it;
- whether it is evidence of a real user pain point or merely an announcement;
- a potential Xiaohongshu angle when appropriate.

Items without a plausible action, learning value, or content value must be down-ranked or omitted.

The evidence package must classify practice evidence, at minimum, as original release/documentation, documented third-party practice, community discussion, or user-verified. Generated content must preserve that distinction and must not imply personal testing when the source is third-party practice.

Original release/documentation and documented third-party practice that contains concrete steps, code/screenshots, or observed results are eligible to create a pending-review draft. Only user-verified evidence may support first-person claims such as “I tested this” or “my results improved”.

Tool discovery must be toolchain-agnostic: Codex skills/plugins, Claude Code skills, and MCP servers belong in one primary discovery experience. Item details must disclose the actual compatible clients and installation/configuration method; “universal” must never mean a misleading one-click install claim.

GitHub qualification must favor both newly published and recently maintained projects, but recency or popularity alone is not enough. A repository must also provide a clear README, executable setup/configuration path, and concrete use case before it enters the daily candidate pool.

Dashboard cards, Feishu digests, actionability explanations, and content opportunities must be Chinese-first. English-source items must retain their original title, source URL, and important technical terms so the user can verify the interpretation and search further.

### R3 — Three delivery modes

- Send an 08:00 daily Feishu digest in Asia/Shanghai.
- Send an 18:00 daily Feishu digest in Asia/Shanghai.
- Send immediate Feishu alerts only for high-confidence major AI signals; routine updates must wait for the next digest.

A major-signal alert must be capped at 1–3 items per day and meet at least one explicit criterion: a major model/platform/core-tool release; a usable agent, skill, or open-source project with material efficiency gains; corroborated practitioner discussion revealing a real pain point; or a source-backed Xiaohongshu content opportunity. The system must prefer silence to low-confidence alerts.

Messages must contain source links and clearly distinguish fact, interpretation, and suggested action.

Each routine digest must contain at most 6–8 qualified AI signals and 2–3 content opportunities. Additional qualified items remain available in the dashboard; they must not inflate the digest.

Before using the text model, collection must apply deterministic source allowlists, canonical-URL deduplication, recency checks, and GitHub usability gates. Automated `deepseek-v4-pro` analysis is capped at 30 newly collected candidates per day. User-initiated content-package generation is separate from that cap.

### R4 — Personal dashboard

The dashboard must prioritize three outcomes over a raw stream:

- today's must-read AI signals;
- usable GitHub projects, skills, and agent workflows;
- Xiaohongshu-ready content opportunities.

The graph must not be a primary navigation destination in the first version. It may be retained only when it explains a signal's evidence and relationships.

### R5 — Reliable collection and operations

- Collect the enabled source set every three hours. Run an additional collection at 07:40 and 17:40 Asia/Shanghai so the 08:00 and 18:00 digests are based on fresh data.
- Never create duplicate items for the same source and canonical URL.
- Capture per-source crawl status, duration, item count, and failure reason so unavailable or blocked sources are visible.
- Do not spend AI calls on duplicate or unqualified items.
- The user must be able to see which sources are enabled and their latest successful collection time.
- List-page titles alone are insufficient for evidence-backed interpretation. Collection must support a two-stage policy: fetch source lists first, then fetch and preserve the primary article, GitHub README/release, source project, or discussion details only for shortlisted candidates.
- Qualified candidates must proceed to second-stage detail collection before any model analysis. This additional source traffic is intentional because title-only interpretation is not an acceptable substitute for evidence.

### R6 — Safety before production use

- Rotate the currently exposed database password and restrict PostgreSQL network access to trusted IPs or private networking before deployment.
- Require a single-administrator password login and server-side session for the personal dashboard. All crawl, AI analysis, and content-generation routes must require that authenticated session; secrets must remain in server environment variables.
- Replace or isolate destructive integration tests so they cannot delete production data through a production `DATABASE_URL`.

### R7 — Evidence-backed content factory

For a selected signal, the system must produce an editable, evidence-backed content package rather than a single opaque AI answer. The package must preserve separate artifacts for:

- source snapshot and original links;
- Chinese analysis, factual claims, uncertainty, and actionability rationale;
- content strategy and platform-specific outline/draft;
- visual plan and one reproducible image-generation prompt per proposed asset;
- final platform-ready package, with image rendering deliberately left to the user's chosen tool in the first version.

The system must support different structures and aspect ratios for different platforms. It must never silently turn an unverified source into a fact claim. It may recommend or generate drafts and image prompts, but external publishing must require a final user confirmation.

The first version is prompt-first: it saves an editable visual plan and reproducible prompt for every proposed asset, but does not call an image-generation provider or store rendered images. This keeps model credentials, image cost, and image-quality review outside the initial delivery while retaining a later integration seam.

The content factory starts only when the user selects a qualified signal. It may display a suggested angle beforehand, but must not automatically generate full three-platform packages for every high-scoring opportunity.

The first platform adapters are Xiaohongshu, WeChat Official Account, and X. They must derive their draft, length, structure, visual plan, and metadata from one approved evidence package while retaining platform-specific constraints. Publishing automation is deliberately deferred; the first version exports ready-to-review packages rather than posting to accounts.

X packages are Chinese-first and include an English counterpart. The Chinese and English versions must both retain the evidence links and should be adapted for their audiences rather than mechanically concatenated or word-for-word translated.

### R8 — Evolvable implementation boundary

The redesign may introduce Python services managed by `uv`, especially for source collection and AI/content processing. The technical design must assign responsibilities by reliability and operational need rather than preserve Go for its own sake. Python dependencies must be reproducible and isolated; the existing Go/React code may be retained, replaced, or bridged only after the design is reviewed.

The default text model is `deepseek-v4-pro`, using DeepSeek's OpenAI-compatible API with all provider URL, model name, and credentials supplied only through server-side environment variables. The legacy `deepseek-chat` default must be removed.

## Out of Scope for the First Version

- General news and entertainment trending topics.
- Brand monitoring, broad public-opinion monitoring, and marketing budget analysis.
- Claims that an LLM has fact-checked a story solely from its title or URL.
- Large-scale multi-user SaaS features.
- Automatic external publishing, X interaction automation, and X keyword-search crawling in the current delivery.

## Acceptance Criteria

- [ ] The first screen is recognizably an AI signal radar, not a generic nine-source hot-list.
- [ ] Linux.do is absent from all enabled collection sources, dashboard filters, scheduled jobs, and notification output.
- [ ] The approved replacement source set, GitHub, and Reddit are enabled collection sources in the primary experience. X is presented as a deferred future source rather than silently substituted with another source.
- [ ] A surfaced item explains why it is actionable and includes the original source link.
- [ ] Feishu receives correctly timed 08:00 and 18:00 Asia/Shanghai digests in a testable configuration.
- [ ] All enabled collection sources run every three hours, with fresh pre-digest runs at 07:40 and 17:40 Asia/Shanghai.
- [ ] Immediate alerts are subject to explicit major-signal criteria and routine updates do not generate them.
- [ ] Each routine Feishu digest stays within the approved 6–8 signal and 2–3 content-opportunity limits.
- [ ] Automatic text-model analysis never exceeds 30 new candidates in a day and runs only after deterministic qualification.
- [ ] GitHub projects/skills and Reddit pain-point discussions can be distinguished from announcements.
- [ ] Reddit collection uses the approved editable community allowlist rather than an all-site feed.
- [ ] The GitHub primary view prioritizes directly usable agent, skill, and MCP tooling rather than general AI frameworks or popular repositories.
- [ ] Agent skills, plugins, and MCP servers are discovered in one unified view, with accurate compatibility and setup information.
- [ ] English-source items have Chinese explanations while retaining their original title, source link, and key technical terms.
- [ ] Generated content distinguishes user-verified practice from third-party documented practice and links to the supporting source.
- [ ] Duplicate `(source, canonical URL)` records are prevented, and source health is observable.
- [ ] Any model-derived fact or action recommendation is traceable to a preserved source excerpt or repository documentation, not merely a list title.
- [ ] Production routes and integration tests cannot accidentally expose AI spend or delete production records.
- [ ] The dashboard and all crawl, analysis, and content-generation operations reject unauthenticated access.
- [ ] A selected qualified signal can produce a Chinese, source-linked content package with editable analysis, platform draft, visual plan, and saved image prompts.
- [ ] No full content package is generated unless the user explicitly selects the signal.
- [ ] Platform-specific copy and visual assets are only marked ready for publishing after the user reviews them.
- [ ] Any Python collector/processor introduced by the redesign is managed through `uv` with reproducible dependencies.
- [ ] The deployed text-model configuration uses `deepseek-v4-pro`, never exposes credentials to the browser, and does not retain the legacy `deepseek-chat` default.
- [ ] One approved signal can produce distinct, reviewable packages for Xiaohongshu, WeChat Official Account, and X without losing source evidence.
- [ ] Each X package includes Chinese-first copy and an evidence-preserving English counterpart.

## Open Product Decision

- Decide whether WaytoAGI and SkillsMP should both be enabled in the first collection release, or whether to launch one first as a quality experiment.
