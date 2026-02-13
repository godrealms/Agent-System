package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"AI-agent/internal/config"
	"AI-agent/pkg/harness"
)

func main() {
	// Command line flags
	projectDir := flag.String("project-dir", "", "Project directory path (required)")
	projectType := flag.String("project-type", "web-chat-app", "Type of project to create")
	sessions := flag.Int("sessions", 1, "Number of agent sessions to run")
	initialize := flag.Bool("init", false, "Initialize a new project")
	report := flag.Bool("report", false, "Generate project report")
	monitor := flag.Bool("monitor", false, "Start monitoring dashboard")

	flag.Parse()

	// Validate required arguments
	if *projectDir == "" {
		fmt.Println("Error: -project-dir is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg := config.LoadConfig(*projectDir)

	// Create harness
	h := harness.NewHarness(cfg)

	// Handle different modes
	if *initialize {
		if err := h.InitializeProject(*projectType); err != nil {
			log.Fatalf("Failed to initialize project: %v", err)
		}
		fmt.Printf("Project initialized successfully in %s\n", *projectDir)
		return
	}

	if *report {
		report, err := h.GenerateReport()
		if err != nil {
			log.Fatalf("Failed to generate report: %v", err)
		}
		fmt.Println(report)
		return
	}

	if *monitor {
		if err := h.StartMonitoring(); err != nil {
			log.Fatalf("Failed to start monitoring: %v", err)
		}
		return
	}

	// Run agent sessions
	fmt.Printf("Running %d agent session(s)...\n", *sessions)

	results, err := h.RunMultipleSessions(*sessions)
	if err != nil {
		log.Fatalf("Failed to run sessions: %v", err)
	}

	// Print results summary
	fmt.Printf("\n=== Session Results ===\n")
	for i, result := range results {
		status := "✅ SUCCESS"
		if !result.Success {
			status = "❌ FAILED"
		}

		fmt.Printf("Session %d (%s): %s\n", i+1, result.SessionID, status)
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Features Done: %v\n", result.FeaturesDone)
		fmt.Printf("  Tokens Used: %d\n", result.TokenUsage.TotalTokens)
		if result.CommitHash != "" {
			fmt.Printf("  Commit: %s\n", result.CommitHash)
		}
		fmt.Println()
	}

	// Generate final report
	finalReport, err := h.GenerateReport()
	if err != nil {
		log.Printf("Warning: Failed to generate final report: %v", err)
	} else {
		fmt.Println(finalReport)
	}

	// Cleanup
	if err := h.Cleanup(); err != nil {
		log.Printf("Warning: Cleanup failed: %v", err)
	}

	fmt.Println("Agent execution completed!")
}
