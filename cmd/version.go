package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Build-time injected. The Makefile passes -ldflags to set these; defaults
// below let `go run .` still produce a sensible banner.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display Fastlane version information.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionBanner()
	},
}

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
	fmt.Printf("Fastlane %s\n", Version)
	fmt.Printf("Commit:     %s\n", Commit)
	fmt.Printf("Built:      %s\n", BuildDate)
	fmt.Printf("Go runtime: %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("https://github.com/atharvwasthere/Fastlane\n")
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
