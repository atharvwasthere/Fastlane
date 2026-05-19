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

var pingFlags config.CommandFlags

var pingCmd = &cobra.Command{
	Use:   "ping [host]",
	Short: "Measure latency to a server",
	Long:  `Measure network latency with breakdown (DNS, TCP, TLS, HTTP). Supports custom server selection.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		host := pingFlags.Server
		switch {
		case host != "":
		case len(args) > 0:
			host = args[0]
		case pingFlags.AutoServer:
			host = cmdint.AutoServerURL(ctx, "ping", "google.com")
		default:
			host = "google.com"
		}

		timeout := time.Duration(globalFlags.Timeout) * time.Second
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		engine := cmdint.PingEngine(cmdint.PingParams{
			Host:       host,
			Iterations: 5,
			Timeout:    timeout,
		})
		renderer := cmdint.NewRenderer(globalFlags)
		if err := cmdint.RunBench(ctx, "ping", host, engine, renderer, pingFlags.SaveReport); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	pingCmd.Flags().StringVar(&pingFlags.Server, "server", "", "Target server host")
	pingCmd.Flags().BoolVar(&pingFlags.AutoServer, "auto-server", false, "Pick nearest server via geoip + probing")
	pingCmd.Flags().BoolVar(&pingFlags.SaveReport, "save-report", true, "Save report to file")
	rootCmd.AddCommand(pingCmd)
}
