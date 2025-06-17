package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/nettest"
	"github.com/atharvwasthere/Fastlane/internal/server"
	"github.com/atharvwasthere/Fastlane/internal/ui"
	"github.com/atharvwasthere/Fastlane/internal/utils"
	"github.com/spf13/cobra"
)

var (
	pingServer  string
	pingVerbose bool
	pingJSON    bool
)

func init() {
	pingCmd.Flags().StringVar(&pingServer, "server", "", "Target server host (e.g., google.com)")
	pingCmd.Flags().BoolVarP(&pingVerbose, "verbose", "v", false, "Verbose output")
	pingCmd.Flags().BoolVar(&pingJSON, "json", false, "Output result as JSON")
	rootCmd.AddCommand(pingCmd)
}

func testping(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 1. Select server
		selector, err := server.NewSelector("assets\\server.json")
		if err != nil {
			return fmt.Errorf("failed to load servers: %v", err)
		}
		targetServer := pingServer
		if targetServer == "" {
			targetServer = selector.SelectDefault().Host
		}

		// 2. Initialize UI
		printer := ui.NewPrinter(pingVerbose)
		serverInfo := fmt.Sprintf("Target Server: %s (%s, %s)", targetServer, selector.GetServer(targetServer).Location, selector.GetServer(targetServer).Country)
		printer.PrintBox("FASTLANE NETWORK BENCHMARK", serverInfo, time.Now().Format(time.RFC3339))

		// 3. Run ping test
		printer.PrintSpinner("Testing latency...")
		
		result, err := nettest.Ping(ctx, targetServer, pingVerbose)
		if err != nil {
			printer.PrintError(fmt.Sprintf("Ping failed: %v", err))
			return err
		}
		printer.StopSpinner()

		// 4. Output results
		if pingJSON {
			jsonOut, err := utils.ToJSON(result)
			if err != nil {
				return fmt.Errorf("JSON encoding failed: %v", err)
			}
			fmt.Println(jsonOut)
			return nil
		}

		printer.PrintLatencyBreakdown(result)
		printer.PrintBoxFooter("Completed: Ping test passed successfully")
		return nil
	}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Measure latency to nearest server",
	Long:  `Measures network latency with detailed breakdown (DNS, TCP, TLS, TTFB) using TCP sockets and HTTP fallback.`,
	RunE: testping,
}