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
	"github.com/atharvwasthere/Fastlane/internal/testsuite"
	"github.com/spf13/cobra"
)

var testFlags struct {
	Server          string
	Threads         int
	SaveReport      bool
	CI              bool
	MinDownloadMbps float64
	MinUploadMbps   float64
	MaxLatencyMS    float64
	MaxJitterMS     float64
	MaxLossPercent  float64
}

var testCmd = &cobra.Command{
	Use:   "test [host]",
	Short: "Run the full ping → download → upload → loss suite",
	Long: `Sequentially runs all four benchmarks against a single endpoint and
aggregates the result into a single card. Use --ci to enforce thresholds
suitable for CI pipelines.

Exit codes (with --ci):
  0  all thresholds passed
  1  one or more thresholds violated
  2  test could not run (DNS failure, no network, etc.)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		host := testFlags.Server
		if host == "" {
			if len(args) > 0 {
				host = args[0]
			} else {
				host = "speed.cloudflare.com"
			}
		}

		dlURL := fmt.Sprintf("https://%s/__down?bytes=104857600", host)
		upURL := fmt.Sprintf("https://%s/__up", host)

		testDuration := time.Duration(globalFlags.Timeout) * time.Second
		if testDuration == 0 {
			testDuration = 15 * time.Second
		}

		engine := cmdint.TestEngine(cmdint.TestParams{
			Host:         host,
			DownloadURL:  dlURL,
			UploadURL:    upURL,
			Threads:      testFlags.Threads,
			Timeout:      testDuration,
			TestDuration: testDuration,
			LossCount:    50,
			LossRate:     10,
		})
		renderer := cmdint.NewRenderer(globalFlags)

		renderer.Header("test", host)
		res, err := engine.Run(ctx)
		if err != nil && ctx.Err() == nil {
			renderer.Error(err)
			os.Exit(exitTestUnableToRun)
		}
		renderer.Final(res)
		cmdint.PersistResult(ctx, testFlags.SaveReport, res)

		if testFlags.CI {
			os.Exit(ciExit(res))
		}
	},
}

const (
	exitCIPass          = 0
	exitCIViolation     = 1
	exitTestUnableToRun = 2
)

func ciExit(r bench.Result) int {
	t := testsuite.Thresholds{
		MinDownloadMbps: testFlags.MinDownloadMbps,
		MinUploadMbps:   testFlags.MinUploadMbps,
		MaxLatencyMS:    testFlags.MaxLatencyMS,
		MaxJitterMS:     testFlags.MaxJitterMS,
		MaxLossPercent:  testFlags.MaxLossPercent,
	}
	viol := t.Check(r)
	if len(viol) == 0 {
		return exitCIPass
	}
	for _, v := range viol {
		fmt.Fprintf(os.Stderr, "threshold violated: %s\n", v.String())
	}
	return exitCIViolation
}

func init() {
	testCmd.Flags().StringVar(&testFlags.Server, "server", "", "Target server host")
	testCmd.Flags().IntVar(&testFlags.Threads, "threads", 4, "Number of concurrent streams for download/upload")
	testCmd.Flags().BoolVar(&testFlags.SaveReport, "save-report", false, "Save aggregate report to ~/.fastlane/reports/")
	testCmd.Flags().BoolVar(&testFlags.CI, "ci", false, "Enable CI mode (enforce thresholds, exit non-zero on violation)")
	testCmd.Flags().Float64Var(&testFlags.MinDownloadMbps, "min-download", 0, "CI: minimum download Mbps")
	testCmd.Flags().Float64Var(&testFlags.MinUploadMbps, "min-upload", 0, "CI: minimum upload Mbps")
	testCmd.Flags().Float64Var(&testFlags.MaxLatencyMS, "max-latency", 0, "CI: maximum latency ms")
	testCmd.Flags().Float64Var(&testFlags.MaxJitterMS, "max-jitter", 0, "CI: maximum jitter ms")
	testCmd.Flags().Float64Var(&testFlags.MaxLossPercent, "max-loss", 0, "CI: maximum loss percent")
	rootCmd.AddCommand(testCmd)
}
