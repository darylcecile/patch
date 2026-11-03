package panels

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

type terminalPanel struct {
	host          Host
	root          string
	term          *terminal.Terminal
	err           error
	closed        bool
	mu            sync.Mutex
	ready         *terminal.Terminal
	focus         ui.Handle
	program       string
	appearanceSet bool
	dark          bool
	background    ui.Color
}

func newTerminal(host Host, spec workspace.Panel, root string) Instance {
	if spec.Resource != "" {
		root = spec.Resource
	}
	p := &terminalPanel{host: host, root: root, program: "Terminal"}
	p.start()
	return p
}

func (p *terminalPanel) start() {
	p.err = nil
	family := "monospace, JetBrainsMono Nerd Font"
	if runtime.GOOS == "darwin" {
		family = "Menlo, JetBrainsMono Nerd Font"
	}
	// Loading libghostty-vt (including a first-run development download) and shell startup never block a frame.
	go func() {
		program := terminalProgram()
		term, err := terminal.New(terminal.Options{Dir: p.root, ShellIntegration: true,
			Font: terminal.Font{Family: family, Size: 13, LineHeight: 1.15}, Transparent: true,
			Padding: terminal.Padding{X: 14, Y: 8}, OnTitle: func(string) { p.host.Window.Invalidate() }})
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			if term != nil {
				term.Close()
			}
			return
		}
		p.ready = term
		p.mu.Unlock()
		p.host.Window.Update(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.closed {
				return
			}
			p.term, p.err, p.program = term, err, program
			if err != nil {
				log.Printf("patch: terminal: %v", err)
			}
		})
	}()
}

// Mirror mygo's default selection only for the display label; Options.Command
// remains unset so the plugin still owns process launch and login-shell behavior.
func terminalProgram() string {
	if runtime.GOOS == "windows" {
		for _, name := range []string{"pwsh.exe", "powershell.exe", os.Getenv("COMSPEC"), "cmd.exe"} {
			if name == "" {
				continue
			}
			if path, err := exec.LookPath(name); err == nil {
				return strings.TrimSuffix(filepath.Base(path), ".exe")
			}
		}
		return "Terminal"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}
	if runtime.GOOS == "darwin" {
		return "zsh"
	}
	return "sh"
}

func (p *terminalPanel) View(c *ui.Context) {
	if p.err != nil {
		ui.Column(c.Key("terminal-status")).Bind(&p.focus).Focusable().FocusRing(false).Fill().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "Terminal couldn't start").Bold()
			ui.Text(c, p.err.Error()).TextColor(c.Theme().Danger)
			ui.Button(c, "Retry terminal").OnClick(p.start)
		})
		return
	}
	if p.term == nil {
		ui.Column(c.Key("terminal-status")).Bind(&p.focus).Focusable().FocusRing(false).Fill().Children(func() {
			empty(c, "Starting shell…", "Opening your login shell in "+p.root)
		})
		return
	}
	wasFocused := p.focus.FocusWithin(c)
	if !p.appearanceSet || p.dark != c.Theme().Dark || p.background != c.Theme().Background {
		theme := terminal.LightTheme()
		if c.Theme().Dark {
			theme = terminal.DarkTheme()
		}
		theme.Background, theme.Foreground = c.Theme().Background, c.Theme().Text
		p.term.SetTheme(theme, theme)
		p.appearanceSet, p.dark, p.background = true, c.Theme().Dark, c.Theme().Background
	}
	view := terminal.View(c.Key("terminal-content"), p.term).Bind(&p.focus).Label("Terminal input").Grow(1).FillWidth().MinHeight(0)
	if wasFocused {
		view.Focus()
	}
	select {
	case <-p.term.Done():
		ui.Text(c, fmt.Sprintf("Process exited (%d)", p.term.ExitCode())).TextColor(c.Theme().TextMuted).Padding(6, 12)
	default:
	}
}

func (p *terminalPanel) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.ready != nil {
		p.ready.Close()
	}
}

func (p *terminalPanel) Focus()       { p.focus.Focus() }
func (p *terminalPanel) CancelFocus() { p.focus.CancelFocus() }
func (p *terminalPanel) Header() (string, string) {
	dir := p.root
	if p.term != nil && p.term.Dir() != "" {
		dir = p.term.Dir()
	}
	return p.program, dir
}
