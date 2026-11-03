package app

import "runtime"

// Windows leaves ordinary Control editing/terminal keys and AltGr combinations
// to the focused tool. Menu actions retain explicit, distinct app shortcuts.
func menuAccelerator(accelerator string) string {
	if runtime.GOOS != "windows" {
		return accelerator
	}
	if shortcut, ok := windowsAccelerators[accelerator]; ok {
		return shortcut
	}
	return accelerator
}

var windowsAccelerators = map[string]string{
	"CmdOrCtrl+O":               "Ctrl+Shift+O",
	"CmdOrCtrl+N":               "Ctrl+Shift+N",
	"CmdOrCtrl+T":               "Ctrl+Shift+T",
	"CmdOrCtrl+Shift+E":         "Ctrl+Shift+F",
	"CmdOrCtrl+Alt+Right":       "Alt+Right",
	"CmdOrCtrl+Alt+Left":        "Alt+Left",
	"CmdOrCtrl+Alt+Shift+Right": "Alt+Shift+Right",
	"CmdOrCtrl+Alt+Shift+Left":  "Alt+Shift+Left",
	"CmdOrCtrl+Shift+W":         "Ctrl+Shift+Q",
	"CmdOrCtrl+R":               "Ctrl+Shift+R",
	"CmdOrCtrl+D":               "Ctrl+Shift+D",
	"CmdOrCtrl+Shift+D":         "Ctrl+Shift+E",
	"CmdOrCtrl+Shift+Enter":     "Ctrl+Shift+Enter",
	"CmdOrCtrl+Shift+0":         "Ctrl+Shift+0",
	"CmdOrCtrl+Alt+Down":        "Alt+Down",
	"CmdOrCtrl+Alt+M":           "Ctrl+Shift+M",
	"CmdOrCtrl+W":               "Ctrl+Shift+W",
	"CmdOrCtrl+B":               "Ctrl+Shift+B",
	"CmdOrCtrl+,":               "Ctrl+Shift+,",
}
