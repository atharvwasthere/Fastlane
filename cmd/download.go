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

var downloadFlags config.CommandFlags

var downloadCmd = &cobra.Command{
	Use:   "download [host]",
	Short: "Measure download speed",
	Long:  `Measure download bandwidth using multi-threaded TCP streams with convergence detection.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		server := downloadFlags.Server
		switch {
		case server != "":
			// explicit --server wins
		case downloadFlags.AutoServer:
			server = cmdint.AutoServerURL(ctx, "download", "https://speed.cloudflare.com/__down?bytes=104857600")
		default:
			server = "https://speed.cloudflare.com/__down?bytes=104857600"
		}

		testDuration := time.Duration(globalFlags.Timeout) * time.Second
		if testDuration == 0 {
			testDuration = 60 * time.Second
		}
		timeout := testDuration

		engine := cmdint.DownloadEngine(cmdint.DownloadParams{
			URL:          server,
			Threads:      downloadFlags.Threads,
			Timeout:      timeout,
			TestDuration: testDuration,
			Adaptive:     downloadFlags.AdaptiveThreads,
		})
		renderer := cmdint.NewRenderer(globalFlags)
		if err := cmdint.RunBench(ctx, "download", server, engine, renderer, downloadFlags.SaveReport); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	downloadCmd.Flags().StringVar(&downloadFlags.Server, "server", "", "Target server host")
	downloadCmd.Flags().BoolVar(&downloadFlags.AutoServer, "auto-server", false, "Pick nearest server via geoip + probing")
	downloadCmd.Flags().IntVar(&downloadFlags.Threads, "threads", 4, "Number of concurrent streams")
	downloadCmd.Flags().BoolVar(&downloadFlags.AdaptiveThreads, "adaptive-threads", false, "Hill-climb thread count for max throughput before recording")
	downloadCmd.Flags().BoolVar(&downloadFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(downloadCmd)
}
