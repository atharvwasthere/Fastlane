package cmd

import (
	"os"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/pkg/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	globalFlags config.GlobalFlags
	saveReport  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fastlane",
	Short: "Fastlane ▸ Network benchmarking made simple.",
	Long: `Fastlane is a modular, cross-platform CLI tool for network benchmarking.
Measure latency, download/upload speeds, and packet loss with JSON or text output.`,
	Version: "0.1.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Honor --no-color and the NO_COLOR convention (https://no-color.org).
		// JSON output is structured data — never colored regardless.
		if globalFlags.NoColor || os.Getenv("NO_COLOR") != "" || globalFlags.JSON {
			color.NoColor = true
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		printer := ui.NewPrinter(globalFlags.Verbose)
		printer.PrintLogo()
		printer.PrintTaglineBox()
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().BoolVar(&globalFlags.JSON, "json", false, "Output results as JSON")
	rootCmd.PersistentFlags().BoolVarP(&globalFlags.Verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Uint32Var(&globalFlags.Timeout, "timeout", 30, "Timeout in seconds for tests")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.NoColor, "no-color", false, "Disable colored output")
}
