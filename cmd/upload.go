package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/internal/upload"
	"github.com/atharvwasthere/Fastlane/pkg/output"
	"github.com/atharvwasthere/Fastlane/pkg/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var uploadFlags config.CommandFlags

var uploadCmd = &cobra.Command{
	Use:   "upload [host]",
	Short: "Measure upload speed",
	Long:  `Measure upload bandwidth using multi-threaded TCP streams with convergence detection.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server := uploadFlags.Server
		if server == "" {
			// Use Cloudflare speed test upload endpoint
			server = "https://speed.cloudflare.com/__up"
		}

		// Build download engine configuration
		testDuration := time.Duration(globalFlags.Timeout) * time.Second
		if testDuration == 0 {
			testDuration = 60 * time.Second
		}

		timeout := time.Duration(globalFlags.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		uploadCfg := upload.Config{
			URL:            server,
			Threads:        uploadFlags.Threads,
			Timeout:        timeout,
			TestDuration:   testDuration,
			CVThreshold:    0.03,
			UpdateInterval: 100 * time.Millisecond,
			MinSamples:     5,
			ChunkSize:      1024 * 1024, // 1MB chunks
		}

		// Create and run engine
		engine := upload.NewEngine(uploadCfg)

		// JSON output mode
		if globalFlags.JSON {
			result, err := engine.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			jsonResult := output.NewResult("upload", server)
			jsonResult.Data["final_mbps"] = result.FinalMbps
			jsonResult.Data["mean_mbps"] = result.MeanMbps
			jsonResult.Data["stddev_mbps"] = result.StdDevMbps
			jsonResult.Data["ewma_mbps"] = result.EWMAMbps
			jsonResult.Data["min_mbps"] = result.MinMbps
			jsonResult.Data["max_mbps"] = result.MaxMbps
			jsonResult.Data["bytes_uploaded"] = result.BytesUploaded
			jsonResult.Data["threads"] = result.Threads
			jsonResult.Data["samples"] = result.SamplesCollected
			jsonResult.Data["duration_seconds"] = result.Duration.Seconds()
			jsonResult.Data["convergence_cv"] = result.ConvergenceCV
			jsonResult.Data["converged"] = result.Converged

			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(jsonResult)
			return
		}

		// Text output mode with live progress
		printer := ui.NewPrinter(globalFlags.Verbose)
		
		// Clear screen and print header once
		fmt.Print("\033[2J\033[H") // Clear screen and move to top
		printer.PrintLogo()
		printer.PrintTaglineBox()
		printer.PrintBox("FASTLANE NETWORK BENCHMARK")
		printer.PrintDetails(server, "Unknown", "Unknown", time.Now().Format(time.RFC3339))
		fmt.Println()

		// Run test in background and display live progress
		resultChan := make(chan *upload.Result)
		errChan := make(chan error)
		go func() {
			result, err := engine.Run()
			if err != nil {
				errChan <- err
			} else {
				resultChan <- result
			}
		}()

		// Display live progress
		progress := upload.NewLiveProgress(60)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		// Render the static box structure ONCE
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Println(cyan("╔════════════════════════════════════════════════════════╗"))
		fmt.Println(cyan("║") + " UPLOAD TEST IN PROGRESS                               " + cyan("║"))
		fmt.Println(cyan("║") + " Speed:  [                                    ]        " + cyan("║"))
		fmt.Println(cyan("║") + " EWMA:                                                  " + cyan("║"))
		fmt.Println(cyan("║") + " Mean:                                                  " + cyan("║"))
		fmt.Println(cyan("║") + " CV:                                                    " + cyan("║"))
		fmt.Println(cyan("║") + " Samples:    │  Threads:    │  Data:                   " + cyan("║"))
		fmt.Println(cyan("║") + " Status:                                                " + cyan("║"))
		fmt.Println(cyan("╚════════════════════════════════════════════════════════╝"))
		
		// Save cursor position at the top of the box for updates
		fmt.Printf("\033[9A") // Move back up to top of box
		fmt.Printf("\033[s")  // Save cursor position
		
		var result *upload.Result
		for result == nil {
			select {
			case result = <-resultChan:
				break
			case err := <-errChan:
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			case <-ticker.C:
				mean, ewma, stddev, converged, cv := engine.GetCurrentStats()
				bytes := engine.GetBytesUploaded()
				samples := engine.GetSampleCount()
				
				// Precompute values
				barWidth := 36
				maxMbps := 100.0
				if ewma > maxMbps {
					maxMbps = ewma * 1.2
				}
				filled := int((ewma / maxMbps) * float64(barWidth))
				if filled < 0 {
					filled = 0
				}
				if filled > barWidth {
					filled = barWidth
				}
				barStr := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
				
				green := color.New(color.FgGreen).SprintFunc()
				yellow := color.New(color.FgYellow).SprintFunc()
				
				// Restore cursor to saved position (top of box)
				fmt.Printf("\033[u")
				
				// Update Speed bar: move to line 3, column 11
				fmt.Printf("\033[2B\033[11C") // Down 2 lines, right 11 cols
				fmt.Printf("\033[K")          // Clear to end of line
				fmt.Printf("[%s]", barStr)
				
				// Update EWMA: restore, move to line 4, column 10
				fmt.Printf("\033[u\033[3B\033[10C\033[K") // Restore, down 3, right 10, clear to EOL
				fmt.Printf("%s", green(fmt.Sprintf("%.1f Mbps", ewma)))
				
				// Update Mean ± StdDev: restore, move to line 5, column 10
				fmt.Printf("\033[u\033[4B\033[10C\033[K")
				fmt.Printf("%.1f Mbps ± %.1f Mbps", mean, stddev)
				
				// Update CV: restore, move to line 6, column 7
				cvStatus := "○"
				if cv < 0.03 && cv > 0 {
					cvStatus = green("✓")
				} else if cv < 0.1 {
					cvStatus = yellow("○")
				}
				fmt.Printf("\033[u\033[5B\033[7C\033[K")
				fmt.Printf("%.3f %s Convergence threshold: 0.030", cv, cvStatus)
				
				// Update Samples, Threads, Data: restore, move to line 7, column 4
				dataStr := formatBytes(bytes)
				fmt.Printf("\033[u\033[6B\033[4C\033[K")
					fmt.Printf("Samples: %-3d  │  Threads: %-3d  │  Data: %s", samples, uploadCfg.Threads, dataStr)
				
				// Update Status: restore, move to line 8, column 10
				fmt.Printf("\033[u\033[7B\033[10C\033[K")
				if converged {
					fmt.Printf("%s", green("CONVERGED"))
				} else {
					fmt.Printf("Testing...")
				}
				
				// Move cursor below the box
				fmt.Printf("\033[u\033[9B\r")
			}
		}

		// Print final summary
		fmt.Print("\033[2J\033[H") // Clear screen
		printer.PrintLogo()
		printer.PrintTaglineBox()
		printer.PrintBox("FASTLANE NETWORK BENCHMARK")
		fmt.Println(progress.RenderSummary(result))

		// Print text summary
		printer.PrintSection("Upload Speed Summary")
		printer.PrintUploadResult(result.FinalMbps, result.StdDevMbps)
		fmt.Printf("    EWMA:       %.2f Mbps\n", result.EWMAMbps)
		fmt.Printf("    Min/Max:    %.2f / %.2f Mbps\n", result.MinMbps, result.MaxMbps)
		fmt.Printf("    Threads:    %d\n", result.Threads)
		fmt.Printf("    Samples:    %d\n", result.SamplesCollected)
		fmt.Printf("    Duration:   %s\n", result.Duration.Round(10*time.Millisecond))
		fmt.Printf("    Data:       %.2f MB\n", float64(result.BytesUploaded)/1024/1024)
		if result.Converged {
			fmt.Printf("    Status:     ✓ CONVERGED (CV: %.4f)\n", result.ConvergenceCV)
		} else {
			fmt.Printf("    Status:     ○ No convergence (CV: %.4f)\n", result.ConvergenceCV)
		}

		printer.PrintBoxFooter("Upload test completed successfully")
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadFlags.Server, "server", "", "Target server host")
	uploadCmd.Flags().IntVar(&uploadFlags.Threads, "threads", 4, "Number of concurrent streams")
	uploadCmd.Flags().BoolVar(&uploadFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(uploadCmd)
}
