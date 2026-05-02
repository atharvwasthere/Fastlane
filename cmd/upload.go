package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmdint "github.com/atharvwasthere/Fastlane/cmd/internal"
	"github.com/atharvwasthere/Fastlane/internal/config"
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
			server = "https://speed.cloudflare.com/__up"
		}

		testDuration := time.Duration(globalFlags.Timeout) * time.Second
		if testDuration == 0 {
			testDuration = 60 * time.Second
		}
		timeout := testDuration

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		engine := cmdint.UploadEngine(cmdint.UploadParams{
			URL:          server,
			Threads:      uploadFlags.Threads,
			Timeout:      timeout,
			TestDuration: testDuration,
		})
		renderer := cmdint.NewRenderer(globalFlags)
		if err := cmdint.RunBench(ctx, "upload", server, engine, renderer); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadFlags.Server, "server", "", "Target server host")
	uploadCmd.Flags().IntVar(&uploadFlags.Threads, "threads", 4, "Number of concurrent streams")
	uploadCmd.Flags().BoolVar(&uploadFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(uploadCmd)
}
