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

var  (
	uploadServer string
	uploadVerbose bool
	uploadJSON bool
)


// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Test your upload speed.",
	Long: `Runs an upload speed test using TCP sockets with HTTP fallback, measuring bandwidth and speed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return uploadTest(cmd, args)
	},
}

func uploadTest(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Select server
	selector, err := server.NewSelector("assets\\server.json")
	if err != nil {
		return fmt.Errorf("failed to load te servers: %v", err)
	}
	targetServer := uploadServer
	if targetServer == ""{
		targetServer = "httpbin.org"
	}

	// Initializing UI
	printer := ui.NewPrinter(uploadVerbose)
	printer.PrintBox("FASTLANE NETWORK BENCHMARK")
	printer.PrintDetails(targetServer, selector.GetServer(targetServer).Location, selector.GetServer(targetServer).Country, time.Now().Format(time.RFC3339))

	// Run Upload Test
	printer.StartSpinner("UPLOAD", "Testing upstream...")
	result, err := nettest.Upload(ctx , targetServer, uploadVerbose, printer)
	if err != nil {
		printer.StopSpinner()
		printer.PrintError(fmt.Sprintf("UPload failed: %v", err))
		printer.FinishProgressBar()
		return err
	}
	printer.StopSpinner()
	printer.FinishProgressBar()

	// Output results
	if uploadJSON {
		jsonOut, err := utils.ToJSON(result)
		if err != nil {
			return fmt.Errorf("JSON encoding failed: %v", err)
		}
		fmt.Println(jsonOut)
		return nil
	}

	printer.PrintUploadResult(result)
	printer.PrintBoxFooter("Completed: Upload test passed successfully")
	return nil


}

func init() {
	uploadCmd.Flags().StringVar(&uploadServer, "server", "", "Target server host (e.g., speedtest.hetzner.de)")
	uploadCmd.Flags().BoolVarP(&uploadVerbose, "verbose", "v", false, "Verbose ouput")
	uploadCmd.Flags().BoolVar(&uploadVerbose, "json", false, "Output result as JSON")
	
	rootCmd.AddCommand(uploadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// uploadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// uploadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
