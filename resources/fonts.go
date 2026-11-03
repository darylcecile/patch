// Package resources contains assets registered only within Patch's process.
package resources

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/egoist/mygo/ui"
)

//go:embed fonts/JetBrainsMonoNerdFont-Regular.ttf
var symbolsFont []byte

var registerFonts = sync.OnceValue(func() error {
	if err := ui.RegisterFont(symbolsFont, ""); err != nil {
		return fmt.Errorf("register bundled JetBrainsMono Nerd Font fallback: %w", err)
	}
	return nil
})

func RegisterFonts() error { return registerFonts() }
