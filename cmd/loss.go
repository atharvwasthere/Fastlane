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
				host = "8.8.8.8"
			}
		}

		timeout := time.Duration(globalFlags.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		engine := cmdint.LossEngine(cmdint.LossParams{
			Host:         host,
			Port:         7,
			Count:        lossFlags.Count,
			PacketSize:   32,
			Rate:         lossFlags.Rate,
			Timeout:      timeout,
			EnableJitter: true,
		})
		renderer := cmdint.NewRenderer(globalFlags)
		if err := cmdint.RunBench(ctx, "loss", host, engine, renderer); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	lossCmd.Flags().StringVar(&lossFlags.Server, "server", "", "Target server host")
	lossCmd.Flags().IntVar(&lossFlags.Count, "count", 100, "Number of packets to send")
	lossCmd.Flags().IntVar(&lossFlags.Rate, "rate", 10, "Packets per second")
	rootCmd.AddCommand(lossCmd)
}
