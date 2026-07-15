# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

<!--
Document your project's database conventions here.

Questions to answer:
- What ORM/query library do you use?
- How are migrations managed?
- What are the naming conventions for tables/columns?
- How do you handle transactions?
-->

(To be filled by the team)

---

## Query Patterns

<!-- How should queries be written? Batch operations? -->

(To be filled by the team)

---

## Migrations

<!-- How to create and run migrations -->

(To be filled by the team)

---

## Naming Conventions

<!-- Table names, column names, index names -->

(To be filled by the team)

---

## Common Mistakes

<!-- Database-related mistakes your team has made -->

(To be filled by the team)

## Scenario: Collector ingestion and production-safe tests

### 1. Scope / Trigger

- Trigger: a Python collector writes source material through Go into PostgreSQL.
- The contract prevents duplicate model work and prevents tests from deleting production data.

### 2. Signatures

- `(*SignalRepo).IngestIfNew(signal Signal, evidence EvidenceSnapshot) (bool, error)`
- PostgreSQL identity: `signals(source, canonical_url)` is unique.
- Test-only environment key: `TEST_DATABASE_URL`.

### 3. Contracts

- `Signal.OriginalURL` is canonicalized before persistence; fragments and known tracking parameters are removed.
- An evidence snapshot is written only when a signal is newly inserted.
- `TEST_DATABASE_URL` must name a dedicated database containing `trend_graph_test`; `DATABASE_URL` is never a test fallback.

### 4. Validation & Error Matrix

| Condition | Required behavior |
| --- | --- |
| Duplicate `(source, canonical URL)` | Return `created=false`; do not create a second snapshot. |
| Invalid URL | Return an error; write nothing. |
| Missing `TEST_DATABASE_URL` | Skip database integration tests. |
| Test URL not containing `trend_graph_test` | Fail before connecting. |

### 5. Good / Base / Bad Cases

- Good: a new SkillsMP discovery with evidence creates one signal and one snapshot.
- Base: a repeated fetch of the same GitHub URL returns `created=false`.
- Bad: a test reads `DATABASE_URL` or defaults to a production-looking DSN.

### 6. Tests Required

- Unit-test URL canonicalization and source allowlist normalization.
- Test the internal ingestion request validation without a database.
- Run destructive integration tests only with an explicitly configured `TEST_DATABASE_URL`.

### 7. Wrong vs Correct

#### Wrong

```go
dsn := os.Getenv("DATABASE_URL")
db.Unscoped().Where("1 = 1").Delete(&HotItem{})
```

#### Correct

```go
dsn := testDatabaseURL(t) // requires TEST_DATABASE_URL containing trend_graph_test
db.Unscoped().Where("1 = 1").Delete(&HotItem{})
```
