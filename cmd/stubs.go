package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/live"
	"github.com/atharvwasthere/Fastlane/pkg/output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// testCmd runs a full test suite
var testCmd = &cobra.Command{
	Use:   "test [host]",
	Short: "Run complete network benchmark",
	Long:  `Run a full test suite: ping, download, upload, and loss. Includes live visualization.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if globalFlags.JSON {
			result := output.NewResult("test", "")
			result.Data["ping_ms"] = 0.0
			result.Data["download_mbps"] = 0.0
			result.Data["upload_mbps"] = 0.0
			result.Data["loss_percent"] = 0.0
			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(result)
			return
		}

		fmt.Println("=== Full Test Suite ===")
		fmt.Println("✓ Test completed (stub)")
	},
}

// liveCmd shows real-time visualization
var liveCmd = &cobra.Command{
	Use:   "live [host]",
	Short: "Show live test visualization",
	Long:  `Display real-time graphs and live updates during testing using ASCII rendering.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create live UI
		ui := live.NewUI()
		ui.Clear()

		// Setup signal handling for clean exit
		done := make(chan struct{})
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			close(done)
		}()

		// Simulate live updates (in real implementation, this would run actual tests)
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					// Simulate changing stats
					stats := live.Stats{
						LatencyMS:       20 + rand.Float64()*30,
						DownloadMbps:    100 + rand.Float64()*200,
						UploadMbps:      50 + rand.Float64()*100,
						LossPercent:     rand.Float64() * 2,
						PacketsSent:     100 + rand.Int63n(50),
						PacketsReceived: 98 + rand.Int63n(3),
						Jitter:          2 + rand.Float64()*5,
					}
					ui.UpdateStats(stats)
				}
			}
		}()

		// Run render loop at 150ms interval
		ui.RenderLoop(150*time.Millisecond, done)

		// Clean up on exit
		fmt.Println("\n\nLive mode ended.")
	},
}

// serversCmd lists available test servers
var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "List available test servers",
	Long:  `List all reachable test servers sorted by latency.`,
	Run: func(cmd *cobra.Command, args []string) {
		if globalFlags.JSON {
			result := output.NewResult("servers", "")
			serverList := []map[string]interface{}{
				{
					"id":       "us-va-1",
					"name":     "Virginia, USA",
					"location": "Ashburn",
					"latency":  45.5,
					"distance": 1200.0,
				},
				{
					"id":       "eu-de-1",
					"name":     "Germany",
					"location": "Frankfurt",
					"latency":  85.3,
					"distance": 8500.0,
				},
			}
			result.Data["servers"] = serverList
			result.Data["total"] = len(serverList)
			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(result)
			return
		}

		fmt.Println("\n=== Available Test Servers ===")
		fmt.Println("  Name                           Location             RTT           Distance")
		fmt.Println("  ─────────────────────────────────────────────────────────────────────────")
		fmt.Println("  Virginia, USA                  Ashburn              45.50 ms      1200.0 km")
		fmt.Println("  Germany                        Frankfurt            85.30 ms      8500.0 km")
		fmt.Println("  ─────────────────────────────────────────────────────────────────────────")
		fmt.Println("  Total: 2 servers")
		fmt.Println()
		}, 
}

// reportCmd views saved test reports
var reportCmd = &cobra.Command{
	Use:   "report [compare] [id1] [id2]",
	Short: "View or compare test reports",
	Long:  `View saved test reports or compare two reports side-by-side.`,
	Args:  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if globalFlags.JSON {
			result := output.NewResult("report", "")
			result.Data["reports"] = []map[string]interface{}{}
			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(result)
			return
		}

		if len(args) > 0 && args[0] == "compare" {
			fmt.Println("=== Report Comparison ===")
		} else {
			fmt.Println("=== Reports ===")
		}
		fmt.Println("✓ Report command completed (stub)")
	},
}

// versionCmd shows CLI version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display Fastlane version information.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionBanner()
	},
}

// printVersionBanner prints the version info with ASCII art
func printVersionBanner() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	magenta := color.New(color.FgHiMagenta).SprintFunc()

	logo := []string{
		"              ___   ____     ____ ",
		"       _   __/  /  / __ \\   / __ \\",
		"      | | / // /  / / / /  / / / /",
		"      | |/ // /_ / /_/ /_ / /_/ / ",
		"      |___//_/(_)\\____/(_)\\____/  ",
		"",
	}

	for _, line := range logo {
		fmt.Println(cyan(line))
	}

	fmt.Printf("\n%s\n\n", magenta("Built for developers who hate lag."))
	fmt.Printf("Fastlane v0.1.0\n")
	fmt.Printf("Phase 2: Server Discovery & Selection\n")
	fmt.Printf("Built with Go 1.24.4\n")
	fmt.Printf("Cross-platform: Linux, macOS, Windows\n")
	fmt.Printf("https://github.com/atharvwasthere/Fastlane\n")
}

func init() {
	// lossCmd is now in loss.go
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(liveCmd)
	rootCmd.AddCommand(serversCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(versionCmd)
}
