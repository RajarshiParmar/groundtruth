# groundtruth

> Developer-owned appraisal reports from Git commit history.

`groundtruth` is a local-first CLI tool that generates appraisal-style reports from your **own Git repositories**.

It is built for developers who want a **clear, honest summary of their work**, without surveillance, hidden scoring, or management-driven KPIs.

---

## Installation

Clone the repository and build locally:

```bash
git clone https://github.com/RajarshiParmar/groundtruth
cd groundtruth
go build ./cmd/groundtruth
```

This produces a `groundtruth` binary.

---

## Basic usage

Create a configuration file (see example below), then run:

```bash
./groundtruth run --config config.example.yaml
```

Check the configuration and repository access without scanning history:

```bash
./groundtruth validate --config config.example.yaml
```

Scan and compute metrics, but write no reports:

```bash
./groundtruth run --config config.example.yaml --dry-run
```

Write reports somewhere other than `output.directory`:

```bash
./groundtruth run --config config.example.yaml --output ./my-report
```

Enable verbose output for debugging:

```bash
./groundtruth run --config config.example.yaml --verbose
```

---

## Output

Reports are written to the directory specified in the config file.

Supported output formats:

- CSV
- XLSX
- PDF
- Markdown (`md`)

### CSV
One file is generated per enabled metric, for example:

- `commit_count.csv`
- `commit_by_type.csv`
- `commit_by_scope.csv`
- `lines_changed.csv`
- `files_changed.csv`
- `activity_timeline_weekly.csv`
- `activity_timeline_monthly.csv`
- `merge_summaries.csv`
- `contribution_score.csv`
- `highlights.csv` (one row per highlight: date, scope, type, breaking, subject, tickets, hash)
- `tickets.csv`

### XLSX
A single Excel file is generated:

```
report.xlsx
```

Each enabled metric is written to its own sheet.

### PDF
A single, print-friendly document is generated:

```
report.pdf
```

Each metric is rendered as a readable section.

The PDF uses the built-in fonts, which cover Western European characters (Windows-1252) only. Other scripts, such as Polish, Cyrillic, Greek, CJK or emoji, show up as dots; use the CSV or XLSX output for the exact text.

### Markdown
A single document for pasting into an appraisal doc (GitHub, Notion, Confluence, or Google Docs via Markdown import):

```
report.md
```

It starts with the period, then lists **every** highlight grouped by scope (not the short view), then the other metrics as tables:

```markdown
# groundtruth report

2025-01-01 to 2025-06-30

## Highlights

42 of 42 commits classified (100%)

### auth

feat: 1 · fix: 1

- add SSO login (PROJ-12, #88)
- handle expired tokens (#88)

### Breaking changes

- api: drop v1 endpoints (PROJ-40)

## Commit By Type

| key | count |
|---|---|
| feat | 18 |
| fix | 12 |
```

Commit text is escaped, so characters such as `*`, `_`, `[` or `|` in a subject never turn into formatting or break a table. The file has no timestamp: the same history and range always give the same `report.md`.

Only enabled metrics produce output.

### Terminal summary

A concise summary is always printed to stdout after every successful run:

```
── groundtruth summary ─────────────────
  Commit Count         42
  Commit By Type       chore: 5  docs: 7  feat: 18  fix: 12
  Commit By Scope      api: 9  auth: 14  general: 19
  Highlights           42 of 42 commits classified (100%)
    auth               feat: 5  fix: 2
      - add SSO login [PROJ-12]
      - handle expired tokens [#88]
      - rotate signing keys
      … 4 more
  Breaking changes
      - api: drop v1 endpoints (2025-03-02)
  Tickets              #88: 2  PROJ-12: 3
  Lines Changed        added: 1204  net: 817  removed: 387
  Contribution         84.0  (42 commits)
─────────────────────────────────────────
```

---

## Example configuration (all options)

```yaml
version: 1

repository:
  path: .
  base_dir: false

identity:
  authors:
    - name: Your Name
      email: you@company.com
    - email: you@gmail.com
  mailmap: ~/.mailmap
  co_authored: true

exclude_paths:
  - go.sum
  - "**/vendor/**"

time_range:
  from: 2025-01-01
  to: 2025-06-30
  # last: 6m

merge_commits:
  mode: analyze

metrics:
  commit_count: true
  commit_by_type: true
  lines_changed: false
  files_changed: true
  activity_timeline: true
  merge_summaries: true
  contribution_score: false
  commit_by_scope: true
  highlights: true
  tickets: true
  repo_breakdown: true

highlights:
  types: [feat, fix, perf]
  per_scope: 3

tickets:
  prefixes: [PROJ]

output:
  formats:
    - csv
    - xlsx
    - pdf
    - md
  directory: ./groundtruth-report

scoring:
  enabled: false

# compare: previous
```

### Time range

- `from` / `to` are calendar dates (`YYYY-MM-DD`) in your local time zone. Both days are included, so `from: 2025-01-01` and `to: 2025-01-01` covers that single day.
- `last` is relative to now. Supported values: `30d`, `3m`, `6m`, `1y`.
- Use one style or the other, not both.

### Merge commits

| Mode | Behaviour |
|---|---|
| `ignore` | Merge commits are left out of every metric. |
| `include` | Merge commits are counted like any other commit. |
| `analyze` | Merge commits are counted, and each one records the commits it brought in. Required by `metrics.merge_summaries`. |

A merge summary counts only the merged commits that are themselves in the report (your authors, inside the time range). A branch that is merged more than once is attributed only the commits that are new in each merge.

Merge commits never contribute to `lines_changed` or `files_changed`; those changes are already counted on the commits that made them.

### Scanning several repositories

With `repository.base_dir: true`, `path` is treated as a folder of checkouts:

- Every directory containing `.git` is scanned, including linked worktrees and submodules (where `.git` is a file).
- Repositories nested inside another repository (vendored clones, test fixtures) are not scanned separately.
- Directories that cannot be read, and repositories without commits, are skipped with a warning on stderr. The run fails only if no usable repository is found.
- A commit that appears in more than one repository (forks, backup clones) is counted once.

### Repositories and periods

**Per-repository breakdown.** With `metrics.repo_breakdown: true` and commits from two or more repositories, the report adds a `repo_breakdown` table:

- Each repository is named by its path under `repository.path`, such as `services/api`. If that folder is itself the repository, its folder name is used.
- The columns are `commits`, plus `lines_added` and `lines_removed` when `lines_changed` is on, `files` (distinct files) when `files_changed` is on, and `co_authored` when `identity.co_authored` is on.
- Rows are ordered by commit count, most first, then by name. They add up to the combined totals: a commit found in several clones counts once, for the first repository in alphabetical path order.
- With a single repository, or only one with commits in the period, nothing is added.

**Highlights by repository.** When the commits come from two or more repositories, highlights are grouped by repository first, then by scope. Breaking changes stay one list, each item starting with its repository (`services/api · auth: drop v1`), and `highlights.csv` gains a `repo` column. A single repository's highlights look as they always did.

**Comparing with an earlier period.** `compare` adds a `comparison` table that puts this period's totals next to an earlier period's:

```yaml
compare: previous        # the period of the same length just before time_range

# or an explicit earlier period
compare:
  from: 2024-01-01
  to: 2024-06-30
```

`previous` is worked out from `time_range`:

| time_range | previous |
|---|---|
| `from: 2025-01-01`, `to: 2025-06-30` (whole months) | 2024-07-01 to 2024-12-31 |
| `from: 2025-03-10`, `to: 2025-03-23` (14 days) | 2025-02-24 to 2025-03-09 |
| `last: 6m` | the 6 months before that |
| `last: 30d` | the 30 days before that |

- Every enabled metric that is a number or a count is compared: `commit_count`, `co_authored`, and each key of `lines_changed`, `files_changed`, `commit_by_type` and `commit_by_scope`, as rows such as `commit_by_type.feat`. A key that appears in only one period counts as 0 in the other.
- `tickets` is compared as one row, the number of distinct tickets your commits mention, rather than a row per ticket.
- Highlights, the activity timeline, merge summaries and the per-repository table cover the current period only. The contribution score is not compared: it is off by default, and a trend would make it a headline number.
- The change column is the plain difference (`+6`, `-15`, `0`). Percentages are left out on purpose: going from 1 to 3 commits in a scope reads as `+200%` and says little.
- `comparison.csv` and the `comparison` sheet have the columns `metric, current, previous, change`. The terminal, PDF and `report.md` show which two periods were compared above the table.

### Highlights

`metrics.highlights` lists the commits worth mentioning in an appraisal, grouped by scope:

- A commit is a highlight if its type is in `highlights.types` (default `feat`, `fix`, `perf`), or if it is a breaking change of any type.
- The terminal and PDF show a short view: per scope, the type counts and the `highlights.per_scope` most recent subjects (default 3), then every breaking change.
- `highlights.csv` and the `highlights` sheet in `report.xlsx` contain the full list.
- Commits without a scope are grouped under `general`. Merge commits are never highlights; the commits they bring in are.
- The summary shows how many commits follow Conventional Commits. If your history doesn't use them, highlights will be empty; add `unknown` to `highlights.types` to list those commits too.

```yaml
highlights:
  types: [feat, fix, perf, unknown]
  per_scope: 5
```

### Scopes

`metrics.commit_by_scope` counts your commits per Conventional Commits scope (`feat(auth): ...` counts towards `auth`), which shows the areas you worked on. Commits without a scope count as `general`; commits that aren't Conventional Commits at all are left out (they appear as `unknown` in `commit_by_type`).

### Tickets

`metrics.tickets` counts how many of your commits mention each ticket:

- `#123` is always recognised, including pull request numbers in merge messages (unless `merge_commits.mode` is `ignore`, which drops merge commits before any metric runs). `page#3` and HTML entities such as `&#39;` are ignored.
- Tracker keys such as `PROJ-123` are recognised only for the prefixes you list, so `UTF-8` or `SHA-256` are never mistaken for tickets.
- The whole commit message is searched, including footers such as `Refs: PROJ-12`.

```yaml
tickets:
  prefixes: [PROJ, OPS]
```

### Accurate numbers

Three options keep the numbers close to the work you did by hand.

**`exclude_paths`** keeps lockfiles, vendored code and generated files out of `lines_changed` and `files_changed`:

- A pattern without `/` matches the file name at any depth: `package-lock.json` also matches `web/package-lock.json`.
- A pattern with `/` matches the whole path from the repository root: `vendor/**` matches everything under the top-level `vendor/`, and `**/vendor/**` matches every `vendor/` directory at any depth.
- `*` stays inside one directory and `**` crosses directories. Write directories as `name/**`, not `name/` or `/name`: a pattern that starts or ends with `/`, or contains `\`, never matches a file, and `validate` warns about it.
- The commit still counts everywhere else (commit count, types, scopes, timeline, tickets, highlights). Only its lines and files in excluded paths are left out.
- When `exclude_paths` is set, `lines_changed` and `files_changed` gain an `excluded` entry: the excluded lines (added plus removed), and the number of distinct excluded files. It is shown even when it is `0`, so you can see the option is on.
- Nothing is excluded by default. `config.example.yaml` lists common lockfiles and generated files to start from.

**Mailmaps** let commits from your old emails count as yours:

- Each repository's `.mailmap` is read from its root (or from `HEAD` in a bare repository), in git's format.
- `identity.mailmap` is an optional personal mailmap for every repository; `~/` is expanded to your home directory. It is read after the repository's file and wins where they disagree.
- An author email counts if it, or the email the mailmaps map it to, is listed in `identity.authors`. Emails and mailmap names are compared case-insensitively, as git does.
- Lines the parser cannot read are skipped with a warning naming the file and line.
- Because a repository's `.mailmap` is always read and emails ignore case, an existing config can count more of your commits after upgrading, even without any of the new keys.

**`identity.co_authored: true`** credits commits where you are named in a `Co-authored-by:` trailer but someone else is the author:

- Only the trailer block (the message's last paragraph) is read, and the trailer email goes through the same mailmap and matching as authors.
- Those commits count in every metric, and a `co_authored` count is shown after `commit_count`.
- Highlights mark them `(paired)`, and `highlights.csv` / the XLSX sheet gain a `paired` column.
- It has no effect without `identity.authors`, since every author already counts; `validate` warns about that.

```yaml
identity:
  authors:
    - email: you@company.com
  mailmap: ~/.mailmap
  co_authored: true

exclude_paths:
  - package-lock.json
  - "**/vendor/**"
  - "*.min.js"
```

### Contribution score

When `metrics.contribution_score` is enabled, a weighted composite score is computed from your commits. Default weights are used automatically:

| Type | Weight |
|---|---|
| feat | 3 |
| fix, refactor, perf | 2 |
| docs, chore, test, style, ci, build | 1 |

To override weights, enable scoring and define your own:

```yaml
scoring:
  enabled: true
  weights:
    commit_count: 1
    commit_by_type:
      feat: 3
      fix: 2
      docs: 1
```

**Validation rules:**

- `scoring.enabled: true` requires `metrics.contribution_score: true`.
- `metrics.merge_summaries: true` requires `merge_commits.mode: analyze`.
- Only `commit_count` and `commit_by_type.<type>` are valid keys under `weights`; unknown keys are rejected at load time.
- Weights must be non-negative.
- `highlights.types` entries must be lowercase words, such as `feat` or `unknown`.
- `highlights.per_scope` must be non-negative; `0` means the default of 3.
- `report.sections` has been removed: the `metrics.*` toggles decide what appears in every report. Configs that still contain a `report:` block load with a warning.
- `tickets.prefixes` entries must match `^[A-Z][A-Z0-9]*$`, such as `PROJ` (not `proj` or `PROJ-`).
- Every `exclude_paths` entry must be a valid pattern; an invalid one is named in the error.
- `identity.mailmap` must be a readable file.
- `compare` is either `previous` or a mapping with both `from` and `to`; an explicit earlier period must end before `time_range` starts.

---

## What it does

- Runs locally on your machine
- Analyzes local Git repositories
- Supports Conventional Commits (v1.0.0)
- Understands merge commits
- Generates reports you control

---

## What it does *not* do

- No tracking or monitoring
- No background collection
- No forced metrics
- No required scoring
- No cloud or accounts

---

## Philosophy

- Developers own their work narrative
- Git is the source of truth
- Metrics must be explainable or removable
- Context matters more than counts

---

## License

MIT License. See `LICENSE` for details.
