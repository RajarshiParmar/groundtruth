# groundtruth

> Developer-owned appraisal reports from Git commit history.

`groundtruth` is a small, local-first CLI tool that generates appraisal-style reports from your **own Git repositories**.

It is built for developers who want a **clear, honest summary of their work**, without surveillance, hidden scoring, or management-driven KPIs.

---

## Installation

Clone the repository and build locally:

```bash
git clone https://github.com/<your-username>/groundtruth
cd groundtruth
go build ./cmd/groundtruth
````

This produces a `groundtruth` binary in the project directory.

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

For CSV output, one file is generated per enabled metric, for example:

* `commit_count.csv`
* `commit_by_type.csv`
* `lines_changed.csv`
* `files_changed.csv`
* `activity_timeline_weekly.csv`
* `activity_timeline_monthly.csv`
* `merge_summaries.csv`

Only enabled metrics produce output files.

---

## Example configuration (all options)

```yaml
version: 1

repository:
  # Path to a single git repository or a base directory
  path: .

  # If true, all subdirectories will be scanned for git repositories
  base_dir: false

identity:
  # Only commits matching these authors will be included
  # Useful if you use multiple emails
  authors:
    - name: Your Name
      email: you@company.com
    - email: you@gmail.com

# Exactly one time range mode must be used
time_range:
  # Option A: explicit range
  from: 2025-01-01
  to: 2025-06-30

  # Option B: relative range (comment out from/to if used)
  # last: 6m

merge_commits:
  # ignore | include | analyze
  mode: analyze

metrics:
  commit_count: true
  commit_by_type: true
  lines_changed: false
  files_changed: true
  activity_timeline: true
  merge_summaries: true
  contribution_score: false

report:
  sections:
    summary: true
    metrics: true
    timeline: true
    merges: true

output:
  # Supported: csv
  formats:
    - csv

  # Output directory (created if missing)
  directory: ./groundtruth-report

scoring:
  # Disabled by default. Enable only if you explicitly want a score.
  enabled: false

  # Weights are applied only if scoring is enabled
  weights:
    commit_count: 1
    commit_by_type:
      feat: 3
      fix: 2
      docs: 1
```

---

## What it does

* Runs locally on your machine
* Analyzes local Git repositories
* Supports Conventional Commits (v1.0.0)
* Understands merge commits
* Generates reports you control

---

## What it does *not* do

* No tracking or monitoring
* No background collection
* No forced metrics
* No required scoring
* No cloud or accounts

---

## Configuration

Everything is configurable via a config file:

* Enable or disable metrics
* Enable or disable report sections
* Control merge commit handling
* Include or exclude scoring

If a metric does not represent your work accurately, remove it.

---

## Philosophy

* Developers own their work narrative
* Git is the source of truth
* Metrics must be explainable or removable
* Context matters more than counts

---

## License

MIT License. See `LICENSE` for details.
