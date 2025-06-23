/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/atharvwasthere/Fastlane/internal/ui"
	"github.com/spf13/cobra"
)

var saveReport bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "Fastlane",
	Short: "Fastlane ▸ Benchmark your bandwidth. Beautifully.",
	Long: ` Get blazing fast network performance insights from your command line.
 It's built for speed and simplicity, helping you quickly measure ping latency, download, upload, and advanced network diagnostics with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui := ui.NewPrinter(false) // or pass verbose as needed
		ui.PrintLogo()
		ui.PrintTaglineBox()
		ui.PrintHelpSectionInline()
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.Fastlane.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.SetHelpTemplate(`
Fastlane ▸ Benchmark your bandwidth. Beautifully.

Usage:
  {{.UseLine}}

Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name 10}} {{.Short}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
}
