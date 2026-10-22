package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/git"
	"github.com/RajarshiParmar/groundtruth/internal/identity"
	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/RajarshiParmar/groundtruth/internal/paths"
	"github.com/RajarshiParmar/groundtruth/internal/report"
	"github.com/spf13/cobra"
)

func RunCmd() *cobra.Command {
	var configPath string
	var outputDir string
	var dryRun bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Generate appraisal report from git history",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load and validate config
			cfg, err := config.Load(configPath, func(c *config.Config) {
				if outputDir != "" {
					c.Output.Directory = outputDir
				}
			})
			if err != nil {
				return err
			}
			for _, w := range cfg.Warnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}

			if verbose {
				fmt.Println("config loaded successfully")
				if outputDir != "" {
					fmt.Println("output override:", outputDir)
				}
			}

			// Resolve repositories
			repoPaths, repoWarnings, err := git.ResolveRepos(cfg.Repository.Path, cfg.Repository.BaseDir)
			for _, w := range repoWarnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}
			if err != nil {
				return err
			}

			// Resolve time range; the md period label reuses this instant
			now := time.Now()
			from, to, err := config.TimeRangeBoundsAt(cfg.TimeRange, now)
			if err != nil {
				return err
			}

			// Decide which commits are the user's and which paths count
			opts := git.ScanOptions{
				CoAuthored:    cfg.Identity.CoAuthored,
				AnalyzeMerges: cfg.MergeCommits.Mode == "analyze",
			}
			for _, a := range cfg.Identity.Authors {
				if a.Email != "" {
					opts.Emails = append(opts.Emails, a.Email)
				}
			}
			if opts.Mailmap, err = personalMailmap(cfg.Identity.Mailmap); err != nil {
				return err
			}
			if opts.Exclude, err = paths.NewExcluder(cfg.ExcludePaths); err != nil {
				return err
			}

			// Scan commits
			scans := git.ScanRepos(repoPaths, from, to, opts)

			commits, scanWarnings, duplicates, err := git.MergeScans(scans)
			for _, w := range scanWarnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}
			if err != nil {
				return err
			}

			if verbose && len(scans) > 1 {
				for _, sc := range scans {
					fmt.Printf("%s: %d commits\n", sc.Path, len(sc.Commits))
				}
				if duplicates > 0 {
					fmt.Printf("skipped %d commits found in more than one repository\n", duplicates)
				}
			}

			if cfg.MergeCommits.Mode == "ignore" {
				kept := commits[:0]
				for _, c := range commits {
					if !c.IsMerge {
						kept = append(kept, c)
					}
				}
				commits = kept
			}

			if verbose {
				fmt.Printf("scanned %d commits\n", len(commits))
			}

			// Classify commits using Conventional Commits
			classifier := classify.NewConventionalClassifier()

			classified := make([]model.ClassifiedCommit, 0, len(commits))
			for _, c := range commits {
				classified = append(classified, classifier.Classify(c))
			}

			if verbose {
				fmt.Printf("classified %d commits\n", len(classified))
			}

			// Metrics
			registry := metrics.NewRegistry()
			registry.Register(&metrics.CommitCountMetric{})
			registry.Register(&metrics.CoAuthoredMetric{})
			registry.Register(&metrics.CommitByTypeMetric{})
			// The excluded keys appear only when exclude_paths is set
			showExcluded := len(cfg.ExcludePaths) > 0
			registry.Register(&metrics.LinesChangedMetric{ShowExcluded: showExcluded})
			registry.Register(&metrics.FilesChangedMetric{ShowExcluded: showExcluded})
			registry.Register(&metrics.ActivityTimelineMetric{})
			registry.Register(&metrics.MergeSummariesMetric{})

			// One matcher so tickets and highlights agree on what a reference is
			matcher := classify.NewTicketMatcher(cfg.Tickets.Prefixes)
			registry.Register(&metrics.CommitByScopeMetric{})
			registry.Register(&metrics.TicketsMetric{Matcher: matcher})
			registry.Register(&metrics.HighlightsMetric{
				Types:      cfg.Highlights.Types,
				PerScope:   cfg.Highlights.PerScope,
				Matcher:    matcher,
				CoAuthored: cfg.Identity.CoAuthored,
			})

			// Contribution score: typed weights; zero value yields defaults.
			var scoreWeights config.ScoringWeights
			if cfg.Scoring.Enabled {
				scoreWeights = cfg.Scoring.Weights
			}
			registry.Register(&metrics.ContributionScoreMetric{Weights: scoreWeights})

			var results []metrics.Result

			if cfg.Metrics.CommitCount {
				if m, ok := registry.Get("commit_count"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			// Co-author credit is opt-in, so its count is too
			if cfg.Identity.CoAuthored {
				if m, ok := registry.Get("co_authored"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.CommitByType {
				if m, ok := registry.Get("commit_by_type"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.CommitByScope {
				if m, ok := registry.Get("commit_by_scope"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.Highlights {
				if m, ok := registry.Get("highlights"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.Tickets {
				if m, ok := registry.Get("tickets"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.LinesChanged {
				if m, ok := registry.Get("lines_changed"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.FilesChanged {
				if m, ok := registry.Get("files_changed"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.ActivityTimeline {
				if m, ok := registry.Get("activity_timeline"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			// Merge summaries need merge analysis
			if cfg.Metrics.MergeSummaries && cfg.MergeCommits.Mode == "analyze" {
				if m, ok := registry.Get("merge_summaries"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.ContributionScore {
				if m, ok := registry.Get("contribution_score"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if verbose {
				for _, r := range results {
					fmt.Printf("metric %s = %v\n", r.Name, r.Value)
				}
			}

			// Dry run stops here
			if dryRun {
				fmt.Println("dry run complete")
				return nil
			}

			// Terminal summary
			report.WriteSummary(results)

			// Reports
			for _, f := range cfg.Output.Formats {
				switch f {
				case "csv":
					if err := report.WriteCSV(results, cfg.Output.Directory); err != nil {
						return err
					}
				case "xlsx":
					if err := report.WriteXLSX(results, cfg.Output.Directory); err != nil {
						return err
					}
				case "pdf":
					if err := report.WritePDF(results, cfg.Output.Directory); err != nil {
						return err
					}
				case "md":
					period := config.PeriodLabel(cfg.TimeRange, now)
					if err := report.WriteMarkdown(results, cfg.Output.Directory, period); err != nil {
						return err
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to config file (required)")
	cmd.Flags().StringVar(&outputDir, "output", "", "Override output directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config and repo without running analysis")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	if err := cmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	return cmd
}

// personalMailmap parses identity.mailmap, printing malformed lines as warnings
func personalMailmap(path string) (identity.Mailmap, error) {
	if path == "" {
		return identity.Mailmap{}, nil
	}
	path, err := config.ExpandHome(path)
	if err != nil {
		return identity.Mailmap{}, fmt.Errorf("identity.mailmap: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return identity.Mailmap{}, fmt.Errorf("identity.mailmap: %w", err)
	}
	defer f.Close()

	m, warnings := identity.ParseMailmap(f, path)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return m, nil
}
