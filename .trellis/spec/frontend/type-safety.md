# Type Safety

> Type safety patterns in this project.

---

## Overview

<!--
Document your project's type safety conventions here.

Questions to answer:
- What type system do you use?
- How are types organized?
- What validation library do you use?
- How do you handle type inference?
-->

(To be filled by the team)

---

## Type Organization

<!-- Where types are defined, shared types vs local types -->

(To be filled by the team)

---

## Validation

<!-- Runtime validation patterns (Zod, Yup, io-ts, etc.) -->

(To be filled by the team)

---

## Common Patterns

<!-- Type utilities, generics, type guards -->

(To be filled by the team)

---

## Forbidden Patterns

<!-- any, type assertions, etc. -->

(To be filled by the team)

## Scenario: Radar JSONB response contract

### 1. Scope / Trigger

- Trigger: React reads source settings or structured signal analysis stored as PostgreSQL `jsonb` through the Go API.

### 2. Signatures

- `GET /api/source-configs -> { data: SourceConfig[], count: number }`
- `GET /api/radar/signals?limit=N -> { data: RadarSignal[], count: number }`
- Frontend owners: `SourceConfig`, `RadarSignal`, and `SignalAnalysis` in `src/types/index.ts`.

### 3. Contracts

- `settings` and `analysis` are JSON objects in HTTP responses, never escaped JSON strings.
- Every radar signal retains `originalUrl`; evidence retains `sourceUrl`, `evidenceClass`, and `excerpt`.
- Missing analysis is omitted and rendered as pending; the frontend must not invent an action or content opportunity.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Stored JSONB is valid | Return a JSON object using `json.RawMessage`. |
| Stored analysis is invalid | Return HTTP 500; do not return a partial or fabricated object. |
| Session is absent/expired | Return HTTP 401 and show the password page. |
| Analysis is absent | Render preserved evidence and a pending state. |

### 5. Good / Base / Bad Cases

- Good: a qualified SkillsMP signal renders Chinese analysis, evidence class, action, and the GitHub source link.
- Base: a pending WaytoAGI signal renders its evidence excerpt without an analysis block.
- Bad: the browser receives `"settings":"{\\"communities\\":[...]}"` and parses it independently.

### 6. Tests Required

- Go HTTP contract test asserts structured `analysis` and preserved evidence fields.
- React render test asserts the three dashboard outcomes and original source link.
- Run `npm test`, `npm run build`, and `npm run lint` after contract changes.

### 7. Wrong vs Correct

#### Wrong

```go
c.JSON(http.StatusOK, gin.H{"settings": config.SettingsJSON})
```

#### Correct

```go
c.JSON(http.StatusOK, gin.H{"settings": json.RawMessage(config.SettingsJSON)})
```
