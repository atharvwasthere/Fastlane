package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// versionCmd shows CLI version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display Fastlane version information.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionBanner()
	},
}

// printVersionBanner prints the version info with ASCII art.
// Logo and tagline stay (per user direction); version line will switch
// to ldflags-injected values in Phase 12.
func printVersionBanner() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	magenta := color.New(color.FgHiMagenta).SprintFunc()

	logo := []string{
		"              ___   ____     ____ ",
		"       _   __/  /  / __ \\   / __ \\",
		"      | | / // /  / / / /  / / / /",
		"      | |/ // /_ / /_/ /_ / /_/ / ",
		"      |___//_/(_)\\____/(_)\\____/  ",
		"",
	}

	for _, line := range logo {
		fmt.Println(cyan(line))
	}

	fmt.Printf("\n%s\n\n", magenta("Built for developers who hate lag."))
	fmt.Printf("Fastlane v0.1.0\n")
	fmt.Printf("Built with Go 1.24.4\n")
	fmt.Printf("Cross-platform: Linux, macOS, Windows\n")
	fmt.Printf("https://github.com/atharvwasthere/Fastlane\n")
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
