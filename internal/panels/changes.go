package panels

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
)

type changesPanel struct {
	host     Host
	root     string
	title    string
	snapshot diffSnapshot
	err      error
	loading  bool
	closed   bool
	cancel   context.CancelFunc
	focus    ui.Handle
	list     ui.ListState
	split    bool
}

func newChanges(host Host, spec workspace.Panel, root string) Instance {
	p := &changesPanel{host: host, root: root, title: spec.Title}
	p.list.Key = func(i int) any {
		return struct {
			file   string
			header bool
		}{p.snapshot.files[i/2].key(), i%2 == 0}
	}
	p.list.Header = func(i int) bool { return i%2 == 0 }
	p.list.Label = func(i int) string {
		f := p.snapshot.files[i/2]
		return f.scope + " — " + f.path
	}
	p.refresh()
	return p
}

func (p *changesPanel) refresh() {
	if p.loading || p.closed {
		return
	}
	p.loading, p.err = true, nil
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	p.cancel = cancel
	go func() {
		defer cancel()
		snapshot, err := loadDiffSnapshot(ctx, p.root)
		p.host.Window.Update(func() {
			if p.closed {
				return
			}
			collapsed := make(map[string]bool, len(p.snapshot.files))
			for _, f := range p.snapshot.files {
				collapsed[f.key()] = f.collapsed
			}
			for _, f := range snapshot.files {
				f.collapsed = collapsed[f.key()]
			}
			p.snapshot, p.err, p.loading = snapshot, err, false
			if err != nil {
				log.Printf("patch: git: %v", err)
			}
		})
	}()
}

func (p *changesPanel) View(c *ui.Context) {
	wasFocused := p.focus.FocusWithin(c)
	hasFiles := p.err == nil && len(p.snapshot.files) > 0
	// Keep the list/editor's rectangular clips inside one viewport-sized
	// layer, composited beneath the panel ancestor's rounded clip. Caching
	// this clipped root bounds the texture independently of the diff's length.
	root := ui.Column(c.Key("changes-root")).Fill().MinHeight(0).MinWidth(0).Clip().Cache()
	if !hasFiles {
		root.Bind(&p.focus).Focusable().FocusRing(false).Label("Git changes")
	}
	root.Children(func() {
		switch {
		case p.err != nil:
			p.errorView(c)
		case hasFiles:
			notice := p.snapshot.notice
			if p.loading {
				notice = "Refreshing changes…"
			}
			if notice != "" {
				ui.Text(c.Key("changes-notice"), notice).FontSize(11).TextColor(c.Theme().TextMuted).
					Padding(4, 12, 2).FillWidth()
			}
			p.document(c, wasFocused)
		case p.loading:
			p.emptyView(c, "Reading changes…", "Checking staged, unstaged and untracked files.")
		default:
			p.emptyView(c, "Working tree clean", "No staged, unstaged or untracked changes.")
		}
	})
}

// Menu contributes view commands to the host's existing panel menus.
// Refresh continues to use the registry's Refresh contract.
func (p *changesPanel) Menu(m *ui.Menu) {
	if m.Item("Unified diff").Checked(!p.split).Chosen() {
		p.split = false
	}
	if m.Item("Split diff").Checked(p.split).Chosen() {
		p.split = true
	}
	m.Separator()
	if m.Item("Collapse all files").Disabled(len(p.snapshot.files) == 0).Chosen() {
		for _, f := range p.snapshot.files {
			f.collapsed = true
		}
	}
	if m.Item("Expand all files").Disabled(len(p.snapshot.files) == 0).Chosen() {
		for _, f := range p.snapshot.files {
			f.collapsed = false
		}
	}
}

func (p *changesPanel) document(c *ui.Context, wasFocused bool) {
	// List virtualizes the file cards and pins their headers. DiffView uses
	// the ancestor's visible clip to paint only the code lines in the pane,
	// even when one FitContent document is thousands of lines tall.
	width := p.list.Bounds(c).W
	list := ui.List(c.Key("changes-document"), &p.list, 2*len(p.snapshot.files), func(i int) {
		f := p.snapshot.files[i/2]
		if i%2 == 0 {
			p.fileHeader(c, f)
		} else {
			p.fileBody(c, f, p.split && width >= 720)
		}
	}).Bind(&p.focus).Label("Local Git changes").Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(0, 10, 10)
	if wasFocused && !p.list.FocusWithin(c) {
		list.Focus()
	}
}

func (p *changesPanel) fileHeader(c *ui.Context, f *diffFile) {
	t := c.Theme()
	ui.Column(c).Padding(7, 0, 0).Background(t.Background).Children(func() {
		h := ui.ButtonBase(c.Key("changes-file-header")).Label(f.scope+" — "+f.path).Tooltip(f.path).
			FillWidth().Height(44).Padding(0, 10, 0, 8).Gap(8).Border(1, t.Border.Alpha(.55)).
			Background(t.Surface).Radius(8, 8, 0, 0)
		if f.collapsed {
			h.Radius(8)
		}
		if h.Hovered() {
			h.Background(t.SurfaceHover)
		}
		h.OnClick(func() { f.collapsed = !f.collapsed })
		h.Children(func() {
			chevron := ui.Icon(c, diffChevronIcon).Size(13, 13).Shrink(0).TextColor(t.TextMuted)
			if f.collapsed {
				chevron.Rotate(-90)
			}
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				name := path.Base(f.path)
				dir := strings.TrimSuffix(f.path, name)
				ui.RichText(c, ui.Span{Text: dir, Color: t.TextMuted}, ui.Span{Text: name, Weight: 600}).
					Font("Menlo, monospace").FontSize(12).SingleLine().FillWidth()
				detail := f.scope + " · " + f.status
				if f.oldPath != "" && f.oldPath != f.path {
					detail += " · from " + f.oldPath
				}
				ui.Text(c, detail).FontSize(10).TextColor(t.TextMuted).SingleLine().FillWidth()
			})
			if f.diff != nil {
				diffCounts(c, f.added, f.deleted)
			}
		})
	})
}

func (p *changesPanel) fileBody(c *ui.Context, f *diffFile, split bool) {
	if f.collapsed {
		ui.Box(c).Height(0)
		return
	}
	t := c.Theme()
	ui.Column(c.Key("changes-file-body")).FillWidth().MinWidth(0).Clip().Radius(0, 0, 8, 8).
		BorderWidth(0, 1, 1, 1).BorderColor(t.Border.Alpha(.55)).Background(t.Background).Children(func() {
		if f.diff != nil {
			f.diff.View(c.Key("changes-file-diff"), split).Label(f.scope + " diff — " + f.path).FillWidth().MinWidth(0)
		}
		if f.note != "" {
			ui.Text(c.Key("changes-file-note"), f.note).FontSize(11).TextColor(t.TextMuted).FillWidth().Padding(10, 12)
		}
	})
}

func diffCounts(c *ui.Context, added, deleted int) {
	add, del := diffChangeColors(c.Theme())
	ui.Row(c).Gap(7).Shrink(0).Children(func() {
		ui.Text(c, fmt.Sprintf("+%d", added)).Font("Menlo, monospace").FontSize(11).TextColor(add)
		ui.Text(c, fmt.Sprintf("−%d", deleted)).Font("Menlo, monospace").FontSize(11).TextColor(del)
	})
}

func diffChangeColors(theme *ui.Theme) (ui.Color, ui.Color) {
	if theme.Dark {
		return ui.Hex("#81cf9b"), ui.Hex("#f29093")
	}
	return ui.Hex("#237b45"), ui.Hex("#bc434a")
}

func (p *changesPanel) errorView(c *ui.Context) {
	title, detail := "Couldn't read Git changes", "Refresh to try again, or open another project folder."
	if strings.Contains(p.err.Error(), "not a git repository") {
		title, detail = "No Git repository", "This folder isn't part of a Git repository. Open a project to see its changes."
	}
	ui.Column(c).Grow(1).FillWidth().Center().Padding(24).Gap(10).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		ui.Text(c, detail).MaxWidth(340).TextColor(c.Theme().TextMuted).Tooltip(p.err.Error())
		if p.host.OpenFolder != nil {
			ui.Button(c, "Open project…").OnClick(p.host.OpenFolder)
		}
	})
}

func (p *changesPanel) emptyView(c *ui.Context, title, detail string) {
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Center().Padding(24).Gap(10).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		ui.Text(c, detail).TextColor(c.Theme().TextMuted).MaxWidth(340)
	})
}

var diffChevronIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m4 6 4 4 4-4"/></svg>`))

func (p *changesPanel) Focus()       { p.focus.Focus() }
func (p *changesPanel) CancelFocus() { p.focus.CancelFocus() }
func (p *changesPanel) Refresh()     { p.refresh() }
func (p *changesPanel) Header() (string, string) {
	root := p.snapshot.root
	if root == "" {
		root = p.root
	}
	return p.title, root
}

func (p *changesPanel) Close() {
	p.closed = true
	if p.cancel != nil {
		p.cancel()
	}
}
