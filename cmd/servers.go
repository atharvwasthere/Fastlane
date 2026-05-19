package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/server"
	"github.com/spf13/cobra"
)

const remoteServersURL = "https://raw.githubusercontent.com/atharvwasthere/Fastlane/master/internal/server/data/servers.json"

var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "List, probe, or update the test-server catalog",
	Long:  `Inspect the embedded server list, probe servers for latency, or refresh the cached list from upstream.`,
}

var serversListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show embedded (or cached) server list",
	Run: func(cmd *cobra.Command, args []string) {
		sl, err := server.LoadCachedOrEmbedded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "load: %v\n", err)
			os.Exit(1)
		}
		if globalFlags.JSON {
			_ = json.NewEncoder(os.Stdout).Encode(sl)
			return
		}
		printServerTable(os.Stdout, sl.Servers, nil)
	},
}

var serversProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Probe each server and sort by latency",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		sl, err := server.LoadCachedOrEmbedded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "load: %v\n", err)
			os.Exit(1)
		}
		metrics := server.ProbeAll(ctx, sl.Servers, 8)
		server.SortByLatency(metrics)

		if globalFlags.JSON {
			out := make([]map[string]any, 0, len(metrics))
			for _, m := range metrics {
				out = append(out, map[string]any{
					"id":         m.Server.ID,
					"name":       m.Server.Name,
					"host":       m.Server.Host,
					"available":  m.Available,
					"latency_ms": m.LatencyMS,
				})
			}
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"results": out})
			return
		}
		printProbeTable(os.Stdout, metrics)
	},
}

var serversUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetch the latest server list from upstream",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := updateServerCache(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "update: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "server list cached to ~/.fastlane/servers.json")
	},
}

func updateServerCache(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, remoteServersURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d from upstream", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var sl server.ServerList
	if err := json.Unmarshal(body, &sl); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(sl.Servers) == 0 {
		return fmt.Errorf("upstream returned empty list")
	}
	return server.SaveCache(&sl)
}

func printServerTable(w io.Writer, servers []server.Server, _ any) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tLOCATION\tCOUNTRY\tHOST:PORT")
	for _, s := range servers {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s:%d\n", s.ID, s.Name, s.Location, s.Country, s.Host, s.Port)
	}
	tw.Flush()
}

func printProbeTable(w io.Writer, metrics []*server.ServerWithMetrics) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tLOCATION\tHOST\tRTT")
	for _, m := range metrics {
		state := "unreachable"
		if m.Available {
			state = fmt.Sprintf("%6.1f ms", m.LatencyMS)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.Server.Name, m.Server.Location, m.Server.Host, state)
	}
	tw.Flush()
}

func init() {
	serversCmd.AddCommand(serversListCmd, serversProbeCmd, serversUpdateCmd)
	rootCmd.AddCommand(serversCmd)
}
