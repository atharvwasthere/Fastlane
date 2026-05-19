package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/internal/logging"
	"github.com/atharvwasthere/Fastlane/pkg/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var (
	globalFlags config.GlobalFlags
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fastlane",
	Short: "Fastlane ▸ Network benchmarking made simple.",
	Long: `Fastlane is a modular, cross-platform CLI tool for network benchmarking.
Measure latency, download/upload speeds, and packet loss with JSON or text output.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Resolve config with precedence: flag > env > file > defaults.
		changed := config.ChangedFlags{
			JSON:    cmd.Flags().Changed("json"),
			Verbose: cmd.Flags().Changed("verbose"),
			Timeout: cmd.Flags().Changed("timeout"),
			Debug:   cmd.Flags().Changed("debug"),
			NoColor: cmd.Flags().Changed("no-color"),
		}
		merged, err := config.Load(globalFlags, changed)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		globalFlags = merged

		// Initialize logger AFTER config merge so --debug/--verbose from any
		// source flows through. JSON mode mutes the logger entirely.
		slog.SetDefault(logging.Init(globalFlags))

		// Honor --no-color and the NO_COLOR convention (https://no-color.org).
		// JSON output is structured data — never colored regardless.
		if globalFlags.NoColor || os.Getenv("NO_COLOR") != "" || globalFlags.JSON {
			color.NoColor = true
			lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout, termenv.WithProfile(termenv.Ascii)))
		}
		return nil
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
