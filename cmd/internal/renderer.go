package internal

import (
	"os"

	"github.com/atharvwasthere/Fastlane/internal/config"
	"github.com/atharvwasthere/Fastlane/pkg/render"
	"golang.org/x/term"
)

// SelectFormat picks the renderer Format based on global flags. Today the
// JSON flag wins; future surfaces (--prometheus) plug in here.
func SelectFormat(flags config.GlobalFlags) render.Format {
	if flags.JSON {
		return render.FormatJSON
	}
	return render.FormatCard
}

// NewRenderer builds the renderer with width resolved once (per skill spec —
// never re-read mid-render).
func NewRenderer(flags config.GlobalFlags) render.Renderer {
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	return render.New(SelectFormat(flags), os.Stdout, render.Options{
		Width:   width,
		NoColor: flags.NoColor,
		Verbose: flags.Verbose,
	})
}
