package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmdint "github.com/atharvwasthere/Fastlane/cmd/internal"
	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/spf13/cobra"
)

var (
	lossFlags config.CommandFlags
	lossMode  string
)

var lossCmd = &cobra.Command{
	Use:   "loss [host]",
	Short: "Measure packet loss",
	Long: `Measure packet loss with HEAD probes (default) or UDP echo (LAN-only).

Modes:
  http   HEAD requests over HTTP/HTTPS. Cross-platform, public-internet safe. Default.
  udp    UDP packets to a controlled echo server (port 9 by default). LAN-only —
         port 7/9 is firewalled on every realistic public host.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := lossFlags.Server
		if host == "" {
			if len(args) > 0 {
				host = args[0]
			} else {
				host = defaultLossHost(lossMode)
			}
		}

		switch lossMode {
		case "http", "udp":
			// ok
		case "":
			lossMode = "http"
		default:
			fmt.Fprintf(os.Stderr, "loss: unknown --mode %q (want http|udp)\n", lossMode)
			os.Exit(2)
		}

		timeout := time.Duration(globalFlags.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		engine := cmdint.LossEngine(cmdint.LossParams{
			Mode:         lossMode,
			Host:         host,
			Port:         9, // ignored for http
			Count:        lossFlags.Count,
			PacketSize:   32,
			Rate:         lossFlags.Rate,
			Timeout:      timeout,
			EnableJitter: true,
		})
		renderer := cmdint.NewRenderer(globalFlags)
		if err := cmdint.RunBench(ctx, "loss", host, engine, renderer, lossFlags.SaveReport); err != nil {
			os.Exit(1)
		}
	},
}

func defaultLossHost(mode string) string {
	if mode == "udp" {
		return "127.0.0.1"
	}
	return "https://www.cloudflare.com"
}

func init() {
	lossCmd.Flags().StringVar(&lossMode, "mode", "http", "Loss probe mode: http|udp")
	lossCmd.Flags().StringVar(&lossFlags.Server, "server", "", "Target host (URL for http, host for udp)")
	lossCmd.Flags().IntVar(&lossFlags.Count, "count", 100, "Number of probes to send")
	lossCmd.Flags().IntVar(&lossFlags.Rate, "rate", 10, "Probes per second")
	lossCmd.Flags().BoolVar(&lossFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(lossCmd)
}
