# Component Guidelines

> How components are built in this project.

---

## Overview

<!--
Document your project's component conventions here.

Questions to answer:
- What component patterns do you use?
- How are props defined?
- How do you handle composition?
- What accessibility standards apply?
-->

(To be filled by the team)

## Scenario: Practice-first radar dashboard

### 1. Scope / Trigger

- Trigger: React renders qualified and pending AI radar signals for the personal daily workflow.

### 2. Signatures

- `RadarDashboard({signals, sources, onLifecycleChange, onGenerateContent, ...})`
- `PATCH /api/radar/signals/:id/lifecycle` persists `new|queued|practiced|dismissed`.

### 3. Contracts

- Each qualified signal appears in exactly one section: `new|queued` in practice work, `practiced` in content-ready work, and `dismissed` nowhere.
- Pending signals live in a collapsed queue and render source, title, link, and state only. Never render or require `evidence.excerpt`; the radar list API returns provenance metadata only.
- Source settings are collapsed low-frequency controls. Content generation is shown only for `practiced` signals.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Lifecycle update fails | Keep current signal list and show the API error. |
| Pending evidence is hundreds of KB | Do not place it in HTML; render the compact queue row. |
| Signal is `queued` | Show the practice plan and `mark practiced`, not content generation. |
| Signal is `practiced` | Move it to content-ready and show one generation action. |

### 5. Good / Base / Bad Cases

- Good: one card explains the pain point, practical use, concrete action, and original source, then persists practice progress.
- Base: no qualified signal shows a useful empty state while the pending queue stays collapsed.
- Bad: the same signal appears under news, tools, and content, or a raw README becomes card copy.

### 6. Tests Required

- Render tests assert section partitioning, one generation button for practiced work, and absence of a raw evidence marker.
- Run `npm test`, `npm run build`, and `npm run lint`; browser-test lifecycle persistence after reload.

### 7. Wrong vs Correct

#### Wrong

```tsx
<p>{signal.analysis?.whatChanged ?? signal.evidence?.excerpt}</p>
```

#### Correct

```tsx
{signal.qualification === 'pending' ? <PendingQueueRow signal={signal} /> : <PracticeCard signal={signal} />}
```

---

## Component Structure

<!-- Standard structure of a component file -->

(To be filled by the team)

---

## Props Conventions

<!-- How props should be defined and typed -->

(To be filled by the team)

---

## Styling Patterns

<!-- How styles are applied (CSS modules, styled-components, Tailwind, etc.) -->

(To be filled by the team)

---

## Accessibility

<!-- A11y requirements and patterns -->

(To be filled by the team)

---

## Common Mistakes

<!-- Component-related mistakes your team has made -->

(To be filled by the team)
