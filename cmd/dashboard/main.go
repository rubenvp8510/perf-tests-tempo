package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redhat/perf-tests-tempo/test/framework/metrics/dashboard"
)

func main() {
	var (
		inputFlag   = flag.String("input", "", "Input CSV metrics file")
		outputFlag  = flag.String("output", "", "Output HTML file (default: input with .html extension)")
		compareFlag = flag.String("compare", "", "Comma-separated list of CSV files to compare")
		profileFlag = flag.String("profile", "", "Profile name (auto-detected from filename if not set)")
		titleFlag   = flag.String("title", "Tempo Performance Test Report", "Dashboard title")
		testType    = flag.String("test-type", "combined", "Test type: ingestion, query, combined")
	)
	flag.Parse()

	// Determine mode: single or comparison
	if *compareFlag != "" {
		// Comparison mode
		csvPaths := strings.Split(*compareFlag, ",")
		for i := range csvPaths {
			csvPaths[i] = strings.TrimSpace(csvPaths[i])
		}

		if len(csvPaths) < 2 {
			fmt.Fprintln(os.Stderr, "Error: --compare requires at least 2 CSV files")
			flag.Usage()
			os.Exit(1)
		}

		// Validate files exist
		for _, p := range csvPaths {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", p)
				os.Exit(1)
			}
		}

		// Auto-detect output path
		output := *outputFlag
		if output == "" {
			output = "comparison-dashboard.html"
		}

		config := dashboard.DashboardConfig{
			Title:       *titleFlag,
			ProfileName: "comparison",
			TestType:    *testType,
			GeneratedAt: time.Now(),
			CompareMode: true,
		}

		fmt.Printf("Generating comparison dashboard from %d files...\n", len(csvPaths))
		for _, p := range csvPaths {
			fmt.Printf("  - %s\n", p)
		}

		if err := dashboard.GenerateComparison(csvPaths, output, config); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating comparison dashboard: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Dashboard generated: %s\n", output)
		return
	}

	// Single file mode
	if *inputFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --input is required (or use --compare for multiple files)")
		flag.Usage()
		os.Exit(1)
	}

	// Validate input file exists
	if _, err := os.Stat(*inputFlag); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: input file not found: %s\n", *inputFlag)
		os.Exit(1)
	}

	// Auto-detect output path
	output := *outputFlag
	if output == "" {
		// Remove .csv extension and -metrics suffix, then add -dashboard.html
		base := strings.TrimSuffix(*inputFlag, ".csv")
		base = strings.TrimSuffix(base, "-metrics")
		output = base + "-dashboard.html"
	}

	// Auto-detect profile name from filename (e.g., "small-metrics.csv" -> "small")
	profile := *profileFlag
	if profile == "" {
		base := filepath.Base(*inputFlag)
		profile = strings.TrimSuffix(base, "-metrics.csv")
		profile = strings.TrimSuffix(profile, ".csv")
	}

	config := dashboard.DashboardConfig{
		Title:       *titleFlag,
		ProfileName: profile,
		TestType:    *testType,
		GeneratedAt: time.Now(),
	}

	fmt.Printf("Generating dashboard from %s...\n", *inputFlag)

	if err := dashboard.Generate(*inputFlag, output, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating dashboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Dashboard generated: %s\n", output)
}
