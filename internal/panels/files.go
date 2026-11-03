package panels

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

const previewLimit = 512 * 1024

type filesPanel struct {
	host       Host
	root       string
	dirs       map[string][]os.DirEntry
	open       map[string]bool
	errors     map[string]error
	loading    map[string]bool
	selected   string
	preview    *editor.State
	previewErr error
	request    uint64
	split      float32
	closed     bool
	scroll     ui.ScrollState
	focus      ui.Handle
}

func newFiles(host Host, _ workspace.Panel, root string) Instance {
	p := &filesPanel{host: host, root: root, split: 39, dirs: make(map[string][]os.DirEntry),
		open: make(map[string]bool), errors: make(map[string]error), loading: make(map[string]bool)}
	p.loadDir(root)
	return p
}

func (p *filesPanel) loadDir(path string) {
	if p.loading[path] {
		return
	}
	p.loading[path] = true
	go func() {
		entries, err := os.ReadDir(path)
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})
		p.host.Window.Update(func() {
			if p.closed {
				return
			}
			p.dirs[path], p.errors[path], p.loading[path] = entries, err, false
		})
	}()
}

func (p *filesPanel) selectFile(path string) {
	p.selected, p.preview, p.previewErr = path, nil, nil
	p.request++
	request := p.request
	go func() {
		text, err := readPreview(path)
		var state *editor.State
		if err == nil {
			buf := editor.NewBuffer(text)
			buf.SetLanguage(editor.LanguageFor(path))
			state = &editor.State{Buffer: buf, ReadOnly: true, NoScrollPastEnd: true}
		}
		p.host.Window.Update(func() {
			if !p.closed && request == p.request {
				p.preview, p.previewErr = state, err
			}
		})
	}()
}

func readPreview(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("only regular text files can be previewed")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, previewLimit+1))
	if err != nil {
		return "", err
	}
	if len(data) > previewLimit {
		return "", fmt.Errorf("file is larger than the 512 KiB preview limit; open it from the terminal")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", fmt.Errorf("binary or non-UTF-8 file; open it from the terminal")
	}
	return string(data), nil
}

func (p *filesPanel) View(c *ui.Context) {
	contentTheme := *c.Theme()
	splitTheme := contentTheme
	// SplitPercent uses Border at rest and Accent while hovered. Scope those
	// paints to transparent, retaining its native sizing, capture and focus ring.
	splitTheme.Border, splitTheme.Accent = ui.Transparent, ui.Transparent
	// Composite the entire nested split once, so its rectangular clips remain
	// inside the bitmap and the panel ancestor's rounded clip applies to the quad.
	ui.Themed(c, &splitTheme).Fill().MinHeight(0).Background(contentTheme.Background).Cache().Children(func() {
		ui.SplitPercent(c, &p.split, func() {
			ui.Themed(c, &contentTheme).Fill().Background(ui.Transparent).Children(func() {
				ui.Scroll(c).TrackScroll(&p.scroll).Fill().Padding(8).Children(func() {
					ui.Tree(c, func() { p.tree(c, p.root, 0) }).Bind(&p.focus).Focusable()
				})
			})
		}, func() {
			ui.Themed(c, &contentTheme).Fill().MinWidth(0).MinHeight(0).Background(ui.Transparent).Children(func() {
				switch {
				case p.previewErr != nil:
					empty(c, "Preview unavailable", p.previewErr.Error())
				case p.preview != nil:
					editor.View(c.Key("file-preview"), p.preview).Label("Read-only file preview").Grow(1).FillWidth().MinHeight(0)
				case p.selected != "":
					empty(c, "Reading file…", filepath.Base(p.selected))
				default:
					ui.Column(c).Fill().Center().Padding(24).Children(func() {
						ui.Text(c, "Select a file to preview.").FontSize(13).TextColor(c.Theme().TextMuted)
					})
				}
			})
		}).Grow(1).MinHeight(0)
	})
}

func (p *filesPanel) tree(c *ui.Context, dir string, depth int) {
	if err := p.errors[dir]; err != nil {
		ui.Text(c, err.Error()).TextColor(c.Theme().Danger)
		ui.Button(c, "Retry folder").OnClick(func() { p.loadDir(dir) })
		return
	}
	if p.loading[dir] {
		ui.Text(c, "Loading…").TextColor(c.Theme().TextMuted)
		return
	}
	entries := p.dirs[dir]
	for _, entry := range entries[:min(len(entries), 1000)] {
		if entry.Name() == ".git" || entry.Name() == ".DS_Store" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() && depth < 20 {
			open := p.open[path]
			ui.TreeItem(c.Key(path), entry.Name(), &open, func() { p.tree(c, path, depth+1) })
			if open != p.open[path] {
				p.open[path] = open
				if open && p.dirs[path] == nil {
					p.loadDir(path)
				}
			}
		} else {
			ui.TreeItem(c.Key(path), entry.Name(), nil, nil).Selected(p.selected == path).
				OnClick(func() { p.selectFile(path) })
		}
	}
	if len(entries) > 1000 {
		ui.Text(c, "Showing the first 1,000 entries.").FontSize(11).TextColor(c.Theme().TextMuted)
	}
	if len(entries) == 0 {
		ui.Text(c, "Empty folder").FontSize(12).TextColor(c.Theme().TextMuted)
	}
}

func (p *filesPanel) Close()       { p.closed = true }
func (p *filesPanel) Focus()       { p.focus.Focus() }
func (p *filesPanel) CancelFocus() { p.focus.CancelFocus() }
func (p *filesPanel) Refresh()     { p.loadDir(p.root) }
