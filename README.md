# groundtruth

> Developer-owned appraisal reports from Git commit history.

`groundtruth` is a local-first CLI tool that generates appraisal-style reports from your **own Git repositories**.

It is built for developers who want a **clear, honest summary of their work**, without surveillance, hidden scoring, or management-driven KPIs.

---

## Installation

Clone the repository and build locally:

```bash
git clone https://github.com/<your-username>/groundtruth
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

Validate configuration and repository access without generating reports:

```bash
./groundtruth run --config config.example.yaml --dry-run
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

### CSV
One file is generated per enabled metric, for example:

- `commit_count.csv`
- `commit_by_type.csv`
- `lines_changed.csv`
- `files_changed.csv`
- `activity_timeline_weekly.csv`
- `activity_timeline_monthly.csv`
- `merge_summaries.csv`

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

Only enabled metrics produce output.

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

output:
  formats:
    - csv
    - xlsx
    - pdf
  directory: ./groundtruth-report

scoring:
  enabled: false
```

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
