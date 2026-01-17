package commands

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/git"
	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
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
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			if verbose {
				fmt.Println("config loaded successfully")
				if outputDir != "" {
					fmt.Println("output override:", outputDir)
				}
			}

			// Initialize git scanner
			scanner := git.NewScanner(cfg.Repository.Path)

			// Resolve time range
			from, to, err := config.TimeRangeToBounds(cfg.TimeRange)
			if err != nil {
				return err
			}

			// Build allowed author email set
			emails := make(map[string]bool)
			for _, a := range cfg.Identity.Authors {
				if a.Email != "" {
					emails[a.Email] = true
				}
			}

			// Scan commits
			commits, err := scanner.Scan(from, to, emails)
			if err != nil {
				return err
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
			registry.Register(&metrics.CommitByTypeMetric{})
			registry.Register(&metrics.LinesChangedMetric{})
			registry.Register(&metrics.FilesChangedMetric{})

			var results []metrics.Result

			if cfg.Metrics.CommitCount {
				if m, ok := registry.Get("commit_count"); ok {
					results = append(results, m.Compute(classified))
				}
			}

			if cfg.Metrics.CommitByType {
				if m, ok := registry.Get("commit_by_type"); ok {
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
