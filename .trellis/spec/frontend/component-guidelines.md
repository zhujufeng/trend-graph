# Component Guidelines

## Personal inbox dashboard

`App` owns server state for signals, topics, source configuration, and the optional content package. `RadarDashboard` owns only presentation state such as the selected inbox tab and unsaved form text.

### Contracts

- Load signals, source configs, and topics together after authentication. A 401 returns to the login boundary.
- Qualified signals appear exactly once according to `inbox|saved|done`; dismissed and rejected signals never render as inbox outcomes.
- Pending signals render only in the collapsed collection queue. Never render `evidence.excerpt` in list HTML.
- Preserve backend order. The backend is the ranking authority; frontend code must not invent a second score formula.
- A card shows source, matched topics, value score, timestamp, concise interpretation, action, and original URL.
- `inbox` offers save, done, and dismiss. `saved` offers done, return to inbox, and dismiss. `done` exposes optional content generation.
- Topic controls use the existing keyword endpoints but present the rows as user interests. Do not expose the obsolete per-keyword interval field.
- Source settings stay in a collapsed low-frequency panel. GitHub watchlist, RSS feeds, and Reddit communities use simple multiline editors and backend validation.
- Content generation is secondary. Do not make Xiaohongshu, WeChat, or X a prerequisite for using the information inbox.

### Component boundaries

- Keep request functions in `src/api`; components receive callbacks and do not import the Axios client.
- Use small local components only where they remove real repetition (`Metric`, `ListEditor`, `SignalCard`). Do not recreate the deleted generic hot-card abstraction.
- Define props explicitly and use backend-aligned types from `src/types` or the owning API module.
- Tailwind utility classes are the styling system; shared button treatment lives in `index.css` as `radar-button`.

### Accessibility

- Every icon-only button needs an `aria-label` or descriptive `title`.
- Tab buttons expose `role=tab` and `aria-selected`.
- Inputs and textareas must have visible label text or an `aria-label`.
- Original links open in a new tab with `rel=noreferrer`.

### Required checks

- Static render tests cover the personal-inbox identity, value/topic information, pending isolation, source editors, and optional content workspace.
- Run `npm test -- --run`, `npm run build`, and `npm run lint` after frontend changes.
- Browser validation must cover topic CRUD, source settings, lifecycle persistence after refresh, empty states, and API errors.
