# groundtruth

> Developer-owned appraisal reports generated from Git commit history.
> Transparent. Configurable. Local-first.

`groundtruth` is a CLI tool that helps developers generate **clear, honest, and configurable appraisal reports** from their own Git repositories. It is designed for developers who want to **own their work narrative**, without surveillance, opaque scoring, or management-driven KPIs that do not reflect real engineering effort.

This tool runs **locally**, analyzes **your Git history**, and produces reports you control — both in **content** and **format**.

---

## Why groundtruth exists

Software development is complex, non-linear, and deeply contextual. Yet performance discussions are often reduced to shallow metrics such as commit counts, lines of code, or arbitrary productivity scores.

`groundtruth` exists to push back against that by:

* Using **Git as the single source of truth**
* Favoring **transparent, explainable metrics**
* Letting **developers choose what is shown**
* Avoiding surveillance-style tracking
* Remaining **offline and local-first**

This is not a monitoring tool. It is a **self-reporting tool**.

---

## What groundtruth is

* A **CLI tool**
* Runs **entirely on your machine**
* Analyzes **local Git repositories**
* Supports **Conventional Commits (v1.0.0)**
* Generates reports in **CSV, XLSX, and PDF**
* Fully **configurable via a config file**

---

## What groundtruth is NOT

* ❌ A productivity surveillance tool
* ❌ A real-time tracker
* ❌ A replacement for human judgment
* ❌ A tool for ranking or policing developers
* ❌ A management dashboard

If you are looking to monitor people rather than understand work, this tool is not for you.

---

## Core principles

1. **Developer ownership**
   You decide what metrics appear in your report.

2. **Transparency over scoring**
   Metrics are visible and explainable. No hidden formulas.

3. **Local-first**
   No cloud. No accounts. No data collection.

4. **Context-aware**
   Merge commits and structured commit messages are treated as valuable signals.

5. **Extensible, not prescriptive**
   The tool provides data. Interpretation remains human.

---

## Supported analysis (v1)

### Commit analysis

* Conventional Commits parsing (`feat`, `fix`, `docs`, `refactor`, etc.)
* Merge commit detection and analysis
* Optional inclusion or analysis of merge commits

### Metrics (all optional via config)

Examples:

* Total commits
* Commits by type
* Lines added / removed
* Files changed
* Activity over time (weekly / monthly)
* Merge-level summaries
* Optional aggregate contribution score

Every metric and section can be **disabled** via configuration.

---

## Configuration-first design

All customization happens via a config file:

* Enable or disable specific metrics
* Enable or disable report sections
* Control merge commit handling
* Control scoring inclusion
* Control output formats

If a metric does not represent your work accurately, **remove it**.

---

## Output formats

* CSV (machine-readable)
* XLSX (spreadsheet-friendly)
* PDF (shareable, human-readable)

---

## Typical use cases

* Personal performance reviews
* Self-reflection and growth tracking
* Preparing appraisal discussions
* Creating a defensible work summary
* Maintaining a private work log

---

## Roadmap (non-binding)

Future versions may include:

* Optional AI-assisted summaries
* Optional GitHub / GitLab / Bitbucket integrations
* Custom classifiers
* Team-level aggregation (opt-in)

These will remain **optional** and **non-invasive**.

---

## License

This project is licensed under the MIT License.

See the `LICENSE` file for details.

---

## Contribution philosophy

Contributions are welcome.

Before contributing, please understand:

* This project prioritizes **developer autonomy** over managerial convenience
* Features that enable surveillance or forced scoring will be rejected
* Transparency and configurability are non-negotiable

---

## Final note

`groundtruth` does not measure your worth.

It helps you **document your work**, honestly and on your own terms.
