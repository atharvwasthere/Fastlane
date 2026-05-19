package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmdint "github.com/atharvwasthere/Fastlane/cmd/internal"
	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/live"
	"github.com/spf13/cobra"
)

var liveCmd = &cobra.Command{
	Use:   "live [host]",
	Short: "Continuous network dashboard",
	Long:  `Cycles ping → short download → short upload → loss probe in a loop, renders updated stats with a sparkline-based bubbletea dashboard.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		host := "speed.cloudflare.com"
		if len(args) > 0 {
			host = args[0]
		}
		dlURL := fmt.Sprintf("https://%s/__down?bytes=10485760", host)
		upURL := fmt.Sprintf("https://%s/__up", host)
		stepDur := 3 * time.Second

		// Factory closures — bench.Engine is single-shot, so the runner must
		// rebuild each cycle. Closures keep internal/live free of concrete
		// engine imports.
		sources := live.Sources{
			Ping: func() bench.Engine {
				return cmdint.PingEngine(cmdint.PingParams{
					Host:       host,
					Iterations: 3,
					Timeout:    5 * time.Second,
				})
			},
			Download: func() bench.Engine {
				return cmdint.DownloadEngine(cmdint.DownloadParams{
					URL:          dlURL,
					Threads:      2,
					Timeout:      stepDur + 2*time.Second,
					TestDuration: stepDur,
				})
			},
			Upload: func() bench.Engine {
				return cmdint.UploadEngine(cmdint.UploadParams{
					URL:          upURL,
					Threads:      2,
					Timeout:      stepDur + 2*time.Second,
					TestDuration: stepDur,
				})
			},
			Loss: func() bench.Engine {
				return cmdint.LossEngine(cmdint.LossParams{
					Mode:    "http",
					Host:    dlURL,
					Count:   20,
					Rate:    10,
					Timeout: 3 * time.Second,
				})
			},
		}

		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		runner := live.NewRunner(sources)
		prog := live.NewProgram(os.Stdout, host, cancel)

		// Pump runner stats into the bubbletea program.
		go runner.Run(runCtx)
		go func() {
			for stats := range runner.C() {
				prog.Send(live.StatsMsg(stats))
			}
		}()

		if _, err := prog.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "live:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(liveCmd)
}
