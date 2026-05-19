// cmd/report.go implements `fastlane report {list,show,compare,trend}`.
//
// The command tree is intentionally read-only: report writing happens in
// cmd/internal/run.go::RunBench (single point of integration). This file
// loads from the FileStore and routes to the renderer for `show`, or
// formats deltas/trends inline for `compare` and `trend`.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	cmdint "github.com/atharvwasthere/Fastlane/cmd/internal"
	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/report"
	"github.com/fatih/color"
	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
)

// metricDirection encodes whether higher or lower is better per metric key.
// This drives the ▲/▼ arrows in `compare`. Keep additions here — the renderer
// stays generic but compare needs the semantic.
var metricDirection = map[string]int{
	// higher is better
	"download_mbps":    +1,
	"upload_mbps":      +1,
	"mean_mbps":        +1,
	"ewma_mbps":        +1,
	"final_mbps":       +1,
	// lower is better
	"latency_ms":   -1,
	"jitter_ms":    -1,
	"min_ms":       -1,
	"max_ms":       -1,
	"stddev_mbps":  -1,
	"stddev_ms":    -1,
	"loss_percent": -1,
	"loss_pct":     -1,
	"dns_ms":       -1,
	"tcp_ms":       -1,
	"tls_ms":       -1,
	"http_ms":      -1,
}

// directionFor falls back to +1 (higher better) when the metric is unknown —
// the comparison still shows a delta, just without colored arrows asserting
// improvement.
func directionFor(key string) int {
	if d, ok := metricDirection[key]; ok {
		return d
	}
	return 0
}

var (
	reportListLimit  int
	reportListKind   string
	reportListSince  string
	reportTrendKey   string
	reportTrendDays  int
	reportTrendKind  string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Inspect saved benchmark reports",
	Long:  `Browse, render, compare, and visualize previously saved Fastlane reports.`,
}

var reportListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved reports",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := report.NewFileStore("")
		f := report.Filter{Limit: reportListLimit}
		if reportListKind != "" {
			f.Kind = bench.Kind(reportListKind)
		}
		if reportListSince != "" {
			t, err := time.Parse("2006-01-02", reportListSince)
			if err != nil {
				return fmt.Errorf("invalid --since (want YYYY-MM-DD): %w", err)
			}
			f.Since = t
		}
		metas, err := s.List(context.Background(), f)
		if err != nil {
			return err
		}
		if globalFlags.JSON {
			return json.NewEncoder(os.Stdout).Encode(metas)
		}
		printList(os.Stdout, metas)
		return nil
	},
}

var reportShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Render a saved report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := report.NewFileStore("")
		r, err := s.Load(context.Background(), args[0])
		if err != nil {
			return err
		}
		// Hand off to the standard renderer — same code path as a live run.
		renderer := cmdint.NewRenderer(globalFlags)
		renderer.Header(string(r.Kind), r.Server)
		renderer.Final(r)
		return nil
	},
}

var reportCompareCmd = &cobra.Command{
	Use:   "compare <a> <b>",
	Short: "Diff two reports metric-by-metric",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := report.NewFileStore("")
		ctx := context.Background()
		a, err := s.Load(ctx, args[0])
		if err != nil {
			return fmt.Errorf("load a: %w", err)
		}
		b, err := s.Load(ctx, args[1])
		if err != nil {
			return fmt.Errorf("load b: %w", err)
		}
		if a.Kind != b.Kind {
			return fmt.Errorf("cannot compare different kinds: %s vs %s", a.Kind, b.Kind)
		}
		printCompare(os.Stdout, a, b)
		return nil
	},
}

var reportTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "ASCII sparkline of a metric over time",
	RunE: func(cmd *cobra.Command, args []string) error {
		if reportTrendKey == "" {
			return fmt.Errorf("--metric is required")
		}
		s := report.NewFileStore("")
		f := report.Filter{}
		if reportTrendKind != "" {
			f.Kind = bench.Kind(reportTrendKind)
		}
		if reportTrendDays > 0 {
			f.Since = time.Now().Add(-time.Duration(reportTrendDays) * 24 * time.Hour)
		}
		metas, err := s.List(context.Background(), f)
		if err != nil {
			return err
		}
		// Load each report (small files; metas already filtered).
		series := make([]float64, 0, len(metas))
		// Iterate oldest-first for left-to-right time axis.
		for i := len(metas) - 1; i >= 0; i-- {
			r, err := s.Load(context.Background(), metas[i].ID)
			if err != nil {
				continue
			}
			if v, ok := r.Metrics[reportTrendKey]; ok && !math.IsNaN(v) {
				series = append(series, v)
			}
		}
		if len(series) < 2 {
			fmt.Fprintf(os.Stderr, "trend: need at least 2 data points for metric %q (got %d)\n", reportTrendKey, len(series))
			return nil
		}
		graph := asciigraph.Plot(series, asciigraph.Height(10), asciigraph.Caption(fmt.Sprintf("%s (last %d reports)", reportTrendKey, len(series))))
		fmt.Println(graph)
		return nil
	},
}

func printList(w *os.File, metas []report.Meta) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tKIND\tSERVER\tAT")
	for _, m := range metas {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.ID, m.Kind, m.Server, m.Timestamp.Local().Format("2006-01-02 15:04:05"))
	}
	tw.Flush()
	if len(metas) == 0 {
		fmt.Fprintln(w, "(no reports)")
	}
}

// printCompare prints a metric-by-metric diff. Direction map decides whether
// the delta is an improvement (▲ green) or regression (▼ red). Dim arrow for
// near-zero deltas (<0.5% relative).
func printCompare(w *os.File, a, b bench.Result) {
	up := color.New(color.FgGreen).SprintFunc()
	down := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// Union of metric keys, sorted.
	keys := unionKeys(a.Metrics, b.Metrics)
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "METRIC\t%s\t%s\tΔ\n",
		a.StartedAt.Local().Format("2006-01-02 15:04"),
		b.StartedAt.Local().Format("2006-01-02 15:04"))
	for _, k := range keys {
		va, oa := a.Metrics[k]
		vb, ob := b.Metrics[k]
		if !oa || !ob {
			// Show one-sided values for completeness, no arrow.
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", k, fmtOpt(va, oa), fmtOpt(vb, ob), dim("—"))
			continue
		}
		delta := vb - va
		relPct := 0.0
		if va != 0 {
			relPct = (delta / va) * 100
		}
		dir := directionFor(k)
		arrow := dim("·")
		// Near-zero: dim regardless of direction.
		if math.Abs(relPct) >= 0.5 && dir != 0 {
			improving := (delta > 0 && dir > 0) || (delta < 0 && dir < 0)
			if improving {
				arrow = up("▲")
			} else {
				arrow = down("▼")
			}
		}
		fmt.Fprintf(tw, "%s\t%.3f\t%.3f\t%+.3f (%+.1f%%) %s\n", k, va, vb, delta, relPct, arrow)
	}
	tw.Flush()
}

func unionKeys(a, b map[string]float64) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func fmtOpt(v float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.3f", v)
}

func init() {
	reportListCmd.Flags().IntVar(&reportListLimit, "limit", 20, "Maximum reports to list")
	reportListCmd.Flags().StringVar(&reportListKind, "kind", "", "Filter by kind (ping|download|upload|loss|test)")
	reportListCmd.Flags().StringVar(&reportListSince, "since", "", "Filter by date (YYYY-MM-DD)")

	reportTrendCmd.Flags().StringVar(&reportTrendKey, "metric", "", "Metric key to plot (e.g. download_mbps)")
	reportTrendCmd.Flags().IntVar(&reportTrendDays, "days", 7, "Lookback window in days")
	reportTrendCmd.Flags().StringVar(&reportTrendKind, "kind", "", "Restrict to a kind")

	reportCmd.AddCommand(reportListCmd, reportShowCmd, reportCompareCmd, reportTrendCmd)
	rootCmd.AddCommand(reportCmd)
}
