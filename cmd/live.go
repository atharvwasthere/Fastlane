package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/live"
	"github.com/spf13/cobra"
)

// liveCmd shows real-time visualization
var liveCmd = &cobra.Command{
	Use:   "live [host]",
	Short: "Show live test visualization",
	Long:  `Display real-time graphs and live updates during testing using ASCII rendering.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui := live.NewUI()
		ui.Clear()

		done := make(chan struct{})
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			close(done)
		}()

		// Simulated stream — Phase 9 will replace this with internal/live/runner.go.
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					stats := live.Stats{
						LatencyMS:       20 + rand.Float64()*30,
						DownloadMbps:    100 + rand.Float64()*200,
						UploadMbps:      50 + rand.Float64()*100,
						LossPercent:     rand.Float64() * 2,
						PacketsSent:     100 + rand.Int63n(50),
						PacketsReceived: 98 + rand.Int63n(3),
						Jitter:          2 + rand.Float64()*5,
					}
					ui.UpdateStats(stats)
				}
			}
		}()

		ui.RenderLoop(150*time.Millisecond, done)

		fmt.Println("\n\nLive mode ended.")
	},
}

func init() {
	rootCmd.AddCommand(liveCmd)
}
