package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/internal/ping"
	"github.com/atharvwasthere/Fastlane/pkg/output"
	"github.com/atharvwasthere/Fastlane/pkg/ui"
	"github.com/spf13/cobra"
)

var pingFlags config.CommandFlags

var pingCmd = &cobra.Command{
	Use:   "ping [host]",
	Short: "Measure latency to a server",
	Long:  `Measure network latency with breakdown (DNS, TCP, TLS, HTTP). Supports custom server selection.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server := pingFlags.Server
		if server == "" {
			if len(args) > 0 {
				server = args[0]
			} else {
				server = "google.com"
			}
		}

		timeout := time.Duration(globalFlags.Timeout) * time.Second
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		if globalFlags.JSON {
			result, err := ping.MeasureLayered(server, 5, timeout)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			filter := ping.NewPautaFilter()
			filtered, _ := filter.Filter(result.Samples)

			jsonResult := output.NewResult("ping", server)
			jsonResult.Data["latency_ms"] = result.MeanMS
			jsonResult.Data["jitter_ms"] = result.JitterMS
			jsonResult.Data["min_ms"] = result.MinMS
			jsonResult.Data["max_ms"] = result.MaxMS
			jsonResult.Data["dns_ms"] = result.DNSLatencyMS
			jsonResult.Data["tcp_ms"] = result.TCPLatencyMS
			jsonResult.Data["tls_ms"] = result.TLSLatencyMS
			jsonResult.Data["http_ms"] = result.HTTPLatencyMS
			jsonResult.Data["samples"] = len(filtered)
			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(jsonResult)
			return
		}

		printer := ui.NewPrinter(globalFlags.Verbose)
		printer.PrintLogo()
		printer.PrintTaglineBox()

		printer.PrintBox("FASTLANE NETWORK BENCHMARK")
		printer.PrintDetails(server, "Remote", "Remote", time.Now().Format(time.RFC3339))

		printer.StartSpinner("PING", "Testing latency...")

		result, err := ping.MeasureLayered(server, 5, timeout)
		if err != nil {
			printer.StopSpinner()
			printer.PrintError(fmt.Sprintf("Ping failed: %v", err))
			return
		}

		filter := ping.NewPautaFilter()
		filtered, removed := filter.Filter(result.Samples)
		if removed > 0 {
			printer.PrintVerbose(fmt.Sprintf("Removed %d outliers", removed))
		}

		printer.StopSpinner()

		printer.PrintSection("Layered Latency Analysis")
		fmt.Printf("  DNS Resolution:  %.2f ms\n", result.DNSLatencyMS)
		fmt.Printf("  TCP Handshake:   %.2f ms\n", result.TCPLatencyMS)
		fmt.Printf("  TLS Handshake:   %.2f ms\n", result.TLSLatencyMS)
		fmt.Printf("  HTTP Fallback:   %.2f ms\n", result.HTTPLatencyMS)
		
		printer.PrintSection("Overall Statistics")
		fmt.Printf("  Mean Latency:    %.2f ms\n", result.MeanMS)
		fmt.Printf("  Jitter (StdDev): %.2f ms\n", result.JitterMS)
		fmt.Printf("  Min:             %.2f ms\n", result.MinMS)
		fmt.Printf("  Max:             %.2f ms\n", result.MaxMS)
		fmt.Printf("  Samples:         %d (after filtering %d outliers)\n", len(filtered), removed)

		printer.PrintBoxFooter("Ping test completed successfully")
	},
}

func init() {
	pingCmd.Flags().StringVar(&pingFlags.Server, "server", "", "Target server host")
	pingCmd.Flags().BoolVar(&pingFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(pingCmd)
}
