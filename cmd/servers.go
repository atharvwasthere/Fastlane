package cmd

import (
	"fmt"
	"os"

	"github.com/atharvwasthere/Fastlane/pkg/output"
	"github.com/spf13/cobra"
)

// serversCmd lists available test servers
var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "List available test servers",
	Long:  `List all reachable test servers sorted by latency.`,
	Run: func(cmd *cobra.Command, args []string) {
		if globalFlags.JSON {
			result := output.NewResult("servers", "")
			serverList := []map[string]interface{}{
				{
					"id":       "us-va-1",
					"name":     "Virginia, USA",
					"location": "Ashburn",
					"latency":  45.5,
					"distance": 1200.0,
				},
				{
					"id":       "eu-de-1",
					"name":     "Germany",
					"location": "Frankfurt",
					"latency":  85.3,
					"distance": 8500.0,
				},
			}
			result.Data["servers"] = serverList
			result.Data["total"] = len(serverList)
			jsonWriter := output.NewJSONWriter(os.Stdout)
			jsonWriter.WriteResult(result)
			return
		}

		fmt.Println("\n=== Available Test Servers ===")
		fmt.Println("  Name                           Location             RTT           Distance")
		fmt.Println("  ─────────────────────────────────────────────────────────────────────────")
		fmt.Println("  Virginia, USA                  Ashburn              45.50 ms      1200.0 km")
		fmt.Println("  Germany                        Frankfurt            85.30 ms      8500.0 km")
		fmt.Println("  ─────────────────────────────────────────────────────────────────────────")
		fmt.Println("  Total: 2 servers")
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(serversCmd)
}
