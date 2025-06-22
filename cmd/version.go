/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/atharvwasthere/Fastlane/internal/ui"
	"os"

	"github.com/spf13/cobra"
)



// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of Fastlane CLI",
	Long:  `Displays the current version, author, and tagline for the Fastlane CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui := ui.NewPrinter(false)
		ui.PrintVersion()
		ui.PrintVersionTaglineBox()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().BoolP("version", "V", false, "Show version info")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			ui := ui.NewPrinter(false)
			ui.PrintVersion()
			ui.PrintVersionTaglineBox()

			os.Exit(0)
		}
	}

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
