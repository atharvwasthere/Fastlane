package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

type Printer struct {
	verbose bool
}

func NewPrinter(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

func (p *Printer) PrintLogo() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	logo := []string{
		"                                                                   ",
		"███████╗ █████╗ ███████╗████████╗██╗      █████╗ ███╗   ██╗███████╗",
		"██╔════╝██╔══██╗██╔════╝╚══██╔══╝██║     ██╔══██╗████╗  ██║██╔════╝",
		"█████╗  ███████║███████╗   ██║   ██║     ███████║██╔██╗ ██║█████╗  ",
		"██╔══╝  ██╔══██║╚════██║   ██║   ██║     ██╔══██║██║╚██╗██║██╔══╝  ",
		"██║     ██║  ██║███████║   ██║   ███████╗██║  ██║██║ ╚████║███████╗",
		"╚═╝     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝",
		"                                                                   ",
	}
	for _, line := range logo {
		fmt.Println(cyan(line))
	}
}

func (p *Printer) PrintTaglineBox() {
	magenta := color.New(color.FgHiMagenta).SprintFunc()
	tagline := "Fastlane - Benchmark your bandwidth. Beautifully."

	width := len(tagline)
	total := width + 4

	top := "╔" + strings.Repeat("═", total-2) + "╗"
	bottom := "╚" + strings.Repeat("═", total-2) + "╝"

	fmt.Println(magenta(top))
	fmt.Printf("%s %s %s\n", magenta("║"), tagline, magenta("║"))
	fmt.Println(magenta(bottom))
	fmt.Println()
}
