package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/internal/loss"
	"github.com/atharvwasthere/Fastlane/pkg/output"
	"github.com/atharvwasthere/Fastlane/pkg/ui"
	"github.com/spf13/cobra"
)

var lossFlags config.CommandFlags

var lossCmd = &cobra.Command{
	Use:   "loss [host]",
	Short: "Measure packet loss",
	Long:  `Measure UDP packet loss with sequence tracking and jitter calculation.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := lossFlags.Server
		if host == "" {
			if len(args) > 0 {
				host = args[0]
			} else {
				host = "8.8.8.8" // Google DNS as default
			}
		}

		// Build loss test configuration
		lossConfig := loss.Config{
			Host:         host,
			Port:         7, // Echo protocol port
			Count:        100,
			PacketSize:   32,
			Rate:         10, // 10 packets/sec
			Timeout:      time.Duration(globalFlags.Timeout) * time.Second,
			EnableJitter: true,
		}

		if lossConfig.Timeout == 0 {
			lossConfig.Timeout = 30 * time.Second
		}

		// Create and run engine
		engine := loss.NewEngine(lossConfig)

		// JSON output mode
		if globalFlags.JSON {
			ctx := context.Background()
			result, err := engine.Run(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			jsonResult := output.NewResult("loss", host)
			jsonResult.Data["packets_sent"] = result.PacketsSent
			jsonResult.Data["packets_received"] = result.PacketsReceived
			jsonResult.Data["packets_lost"] = result.PacketsLost
			jsonResult.Data["loss_percent"] = result.LossPercent
			jsonResult.Data["jitter_ms"] = result.JitterMS
			jsonResult.Data["duration_seconds"] = result.Duration.Seconds()
			jsonResult.Data["test_complete"] = result.TestComplete

			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(jsonResult)
			return
		}

		// Text output mode
		printer := ui.NewPrinter(globalFlags.Verbose)
		printer.PrintLogo()
		printer.PrintTaglineBox()

		printer.PrintBox("FASTLANE NETWORK BENCHMARK")
		printer.PrintDetails(host, "Unknown", "Unknown", time.Now().Format(time.RFC3339))

		printer.PrintSection("UDP Packet Loss Test")
		fmt.Printf("  Sending %d packets to %s:%d...\n", lossConfig.Count, host, lossConfig.Port)
		fmt.Println()

		// Run test with progress
		printer.StartSpinner("loss", "Testing packet loss")
		
		ctx := context.Background()
		result, err := engine.Run(ctx)
		
		printer.StopSpinner()

		if err != nil {
			printer.PrintError(fmt.Sprintf("Test failed: %v", err))
			os.Exit(1)
		}

		// Print results
		printer.PrintSection("Packet Loss Results")
		printer.PrintLossResult(result.LossPercent, result.PacketsSent, result.PacketsReceived)
		
		if lossConfig.EnableJitter && result.JitterMS > 0 {
			fmt.Printf("    Jitter:   %.2f ms\n", result.JitterMS)
		}
		fmt.Printf("    Duration: %s\n", result.Duration.Round(10*time.Millisecond))

		if result.LossPercent == 0 {
			printer.PrintSuccess("No packet loss detected")
		} else if result.LossPercent < 1.0 {
			printer.PrintInfo(fmt.Sprintf("Low packet loss: %.2f%%", result.LossPercent))
		} else if result.LossPercent < 5.0 {
			fmt.Printf("\n⚠️  Moderate packet loss detected\n")
		} else {
			fmt.Printf("\n❌ High packet loss detected\n")
		}

		printer.PrintBoxFooter("Packet loss test completed")
	},
}

func init() {
	lossCmd.Flags().StringVar(&lossFlags.Server, "server", "", "Target server host")
	lossCmd.Flags().IntVar(&lossFlags.Count, "count", 100, "Number of packets to send")
	lossCmd.Flags().IntVar(&lossFlags.Rate, "rate", 10, "Packets per second")
	rootCmd.AddCommand(lossCmd)
}
