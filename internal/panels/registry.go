// Package panels owns live native resources, keyed by durable panel identity.
package panels

import (
	"fmt"

	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type Instance interface {
	View(*ui.Context)
	Close()
}

type Host struct {
	Window      *mygo.Window
	Open        func(workspace.Kind)
	Settings    func()
	SetResource func(panelID, resource string)
	OpenFolder  func()
}

type Factory func(Host, workspace.Panel, string) Instance

type Registry struct {
	host      Host
	factories map[workspace.Kind]Factory
	instances map[string]Instance
	focused   string
}

func NewRegistry(host Host) *Registry {
	r := &Registry{host: host, factories: make(map[workspace.Kind]Factory), instances: make(map[string]Instance)}
	r.Register(workspace.Terminal, newTerminal)
	r.Register(workspace.Files, newFiles)
	r.Register(workspace.Changes, newChanges)
	r.Register(workspace.Browser, newBrowser)
	r.Register(workspace.Primitives, newPrimitives)
	r.Register(workspace.Welcome, func(h Host, p workspace.Panel, root string) Instance { return &welcome{host: h, root: root} })
	return r
}

func (r *Registry) Register(kind workspace.Kind, factory Factory) {
	if _, exists := r.factories[kind]; exists {
		panic("duplicate native panel kind: " + string(kind))
	}
	r.factories[kind] = factory
}

// Sync is called after model actions, outside view construction. Moving a panel never recreates it.
func (r *Registry) Sync(state *workspace.State) {
	keep := make(map[string]bool)
	var visit func(*workspace.Node, string)
	visit = func(n *workspace.Node, root string) {
		if n == nil {
			return
		}
		for _, p := range n.Panels {
			keep[p.ID] = true
			if r.instances[p.ID] == nil {
				if factory := r.factories[p.Kind]; factory != nil {
					r.instances[p.ID] = factory(r.host, p, root)
				} else {
					r.instances[p.ID] = &unavailable{kind: p.Kind}
				}
			}
		}
		visit(n.First, root)
		visit(n.Second, root)
	}
	if state != nil {
		for _, w := range state.Workspaces {
			visit(w.Root, w.RootPath)
		}
	}
	for id, instance := range r.instances {
		if !keep[id] {
			instance.Close()
			delete(r.instances, id)
		}
	}
}

func (r *Registry) View(c *ui.Context, id string) {
	if instance := r.instances[id]; instance != nil {
		instance.View(c)
	}
}

func (r *Registry) Focus(id string) {
	if old, ok := r.instances[r.focused].(interface{ CancelFocus() }); ok {
		old.CancelFocus()
	}
	r.focused = id
	if instance, ok := r.instances[id].(interface{ Focus() }); ok {
		instance.Focus()
	}
}

func (r *Registry) Header(panel workspace.Panel) (string, string) {
	if instance, ok := r.instances[panel.ID].(interface{ Header() (string, string) }); ok {
		return instance.Header()
	}
	return panel.Title, panel.Resource
}

func (r *Registry) CanRefresh(id string) bool {
	_, ok := r.instances[id].(interface{ Refresh() })
	return ok
}
func (r *Registry) Refresh(id string) {
	if instance, ok := r.instances[id].(interface{ Refresh() }); ok {
		instance.Refresh()
	}
}

// Menu contributes optional panel-specific view commands to the host's menu.
func (r *Registry) Menu(id string, menu *ui.Menu) bool {
	if instance, ok := r.instances[id].(interface{ Menu(*ui.Menu) }); ok {
		instance.Menu(menu)
		return true
	}
	return false
}

func (r *Registry) Close() {
	for id, instance := range r.instances {
		instance.Close()
		delete(r.instances, id)
	}
}

func Title(kind workspace.Kind) string {
	switch kind {
	case workspace.Terminal:
		return "Terminal"
	case workspace.Files:
		return "Project files"
	case workspace.Changes:
		return "Changes"
	case workspace.Browser:
		return "Browser"
	case workspace.Primitives:
		return "Native controls"
	default:
		return "Welcome"
	}
}

type unavailable struct {
	kind  workspace.Kind
	focus ui.Handle
}

func (p *unavailable) Close()       {}
func (p *unavailable) Focus()       { p.focus.Focus() }
func (p *unavailable) CancelFocus() { p.focus.CancelFocus() }
func (p *unavailable) View(c *ui.Context) {
	ui.Column(c.Key("unavailable-root")).Bind(&p.focus).Focusable().FocusRing(false).Fill().Children(func() {
		empty(c, "Panel unavailable", fmt.Sprintf("No native factory is registered for %q. Its saved descriptor is preserved.", p.kind))
	})
}

func empty(c *ui.Context, title, detail string) {
	ui.Column(c).Fill().Center().Padding(24).Gap(10).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		ui.Text(c, detail).TextColor(c.Theme().TextMuted).MaxWidth(420)
	})
}
