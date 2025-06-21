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
	"github.com/atharvwasthere/Fastlane/internal/ui"
	"github.com/atharvwasthere/Fastlane/internal/utils"
    "github.com/spf13/cobra"
)

var (
	downloadServer string
	downloadVerbose bool
	downloadJSON bool 
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Get a summary of your test results.",
	Long: `Runs a download speed test using TCP sockets with HTTP fallback, measuring bandwidth and speed.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := runDownload(cmd, args) 
		if err != nil {
		fmt.Println("Download failed:", err)
	}
	},
}

func runDownload(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load servers
	selector, err := server.NewSelector("assets\\server.json")
	if err != nil {
		return fmt.Errorf("failed to load servers: %v", err)
	}

	targetServer := downloadServer
	if targetServer == "" {
		targetServer = "proof.ovh.net"
	}

	printer := ui.NewPrinter(downloadVerbose)

	// UI Box Header
	printer.PrintBox("FASTLANE NETWORK BENCHMARK")

	// Server Info
	srv := selector.GetServer(targetServer)
	printer.PrintDetails(targetServer, srv.Location, srv.Country, time.Now().Format(time.RFC3339))

	// Start Spinner
	printer.StartSpinner("DOWNLOAD", "Testing bandwidth...")
	result, err := nettest.Download(ctx, targetServer, downloadVerbose, printer)
	printer.StopSpinner()

	if err != nil {
		printer.PrintError(fmt.Sprintf("Download failed: %v", err))
		printer.FinishProgressBar()
		return err
	}

	printer.FinishProgressBar()

	// Output results
	if downloadJSON {
		jsonOut, err := utils.ToJSON(result)
		if err != nil {
			return fmt.Errorf("JSON encoding failed: %v", err)
		}
		fmt.Println(jsonOut)
		return nil
	}

	printer.PrintDownloadResult(result)
	printer.PrintBoxFooter("Completed: Download test passed successfully")
	return nil
}

func init() {
	downloadCmd.Flags().StringVar(&downloadServer, "server", "", "Target server host (e.g., speedtest.hetzner.de)")
	downloadCmd.Flags().BoolVarP(&downloadVerbose, "verbose", "v", false, "Verbose output")
	downloadCmd.Flags().BoolVar(&downloadJSON, "json", false, "Output result as JSON")
	rootCmd.AddCommand(downloadCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
