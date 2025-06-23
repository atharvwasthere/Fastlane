/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/nettest"
	"github.com/atharvwasthere/Fastlane/internal/server"
	"github.com/atharvwasthere/Fastlane/internal/types"
	"github.com/atharvwasthere/Fastlane/internal/ui"
	"github.com/atharvwasthere/Fastlane/internal/utils"
	"github.com/spf13/cobra"
)

var (
	fullServer string
	fullVerbose bool
	fullJSON bool
)


// fullCmd represents the full command
var fullCmd = &cobra.Command{
	Use:   "full",
	Short: "Runs the complete suite: ping, download, upload.",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fullTest(cmd, args)
	},
}

func fullTest(cmd *cobra.Command, args []string) error {
	// Initialize server 
	selector, err := server.NewSelector("assets\\server.json")
	if err != nil {
		return fmt.Errorf("failed to load servers: %v", err)
	}
	targetServer := fullServer
	if targetServer == ""{
		targetServer = selector.SelectDefault().Host
	}

		// Initialize UI
		printer := ui.NewPrinter(fullVerbose)
		printer.PrintBox("FASTLANE NETWORK BENCHMARK")
		printer.PrintDetails(targetServer, selector.GetServer(targetServer).Location, selector.GetServer(targetServer).Country, time.Now().Format(time.RFC3339))

		// Initialize full result
		fullResult := &types.FullResult{}

		// Run ping test
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		printer.StartSpinner("PING", "Testing latency...")
		pingResult, err := nettest.Ping(ctx, targetServer, fullVerbose)
		if err != nil {
			printer.StopSpinner()
			printer.PrintError(fmt.Sprintf("Ping failed: %v", err))
			return err
		}
		printer.StopSpinner()
		printer.PrintLatencyBreakdown(pingResult)
		fullResult.Ping = pingResult

		// Run download test
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		printer.StartSpinner("DOWNLOAD", "Testing downstream...")
		downloadResult, err := nettest.Download(ctx, targetServer, fullVerbose, printer)
		if err != nil {
			printer.StopSpinner()
			printer.PrintError(fmt.Sprintf("Download failed: %v", err))
			printer.FinishProgressBar()
			return err
		}
		printer.StopSpinner()
		printer.FinishProgressBar()
		printer.PrintDownloadResult(downloadResult)
		fullResult.Download = downloadResult

		// Run upload test
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		printer.StartSpinner("UPLOAD", "Testing upstream...")
		uploadResult, err := nettest.Upload(ctx, targetServer, fullVerbose, printer)
		if err != nil {
			printer.StopSpinner()
			printer.PrintError(fmt.Sprintf("Upload failed: %v", err))
			printer.FinishProgressBar()
			return err
		}
		printer.StopSpinner()
		printer.FinishProgressBar()
		printer.PrintUploadResult(uploadResult)
		fullResult.Upload = uploadResult

		if saveReport {
			filepath, err := utils.SaveReport(fullResult, targetServer)
			if err != nil {
				printer.PrintError(fmt.Sprintf("Failed to save report: %v", err))
				return err
			}
			printer.PrintReportSaved(filepath)
		}

		// Output results
		if fullJSON {
			jsonOut, err := utils.ToJSON(fullResult)
			if err != nil {
				return fmt.Errorf("JSON encoding failed: %v", err)
			}
			fmt.Println(jsonOut)
			return nil
		}

		printer.PrintBoxFooter("Completed: Full test passed successfully")
		return nil
}

func init() {
	rootCmd.AddCommand(fullCmd)
	fullCmd.Flags().StringVar(&fullServer, "server", "", "Target server host (e.g., speedtest.hetzner.de)")
	fullCmd.Flags().BoolVarP(&fullVerbose, "verbose", "v", false, "Verbose output")
	fullCmd.Flags().BoolVar(&fullJSON, "json", false, "Output result as JSON")
	fullCmd.Flags().BoolVar(&saveReport, "save-report", true, "Save report to JSON file")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fullCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fullCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
