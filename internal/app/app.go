// Package app connects the native window to the platform-independent workspace model.
package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/panels"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/darylcecile/patch/resources"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type shell struct {
	win               *mygo.Window
	state             *workspace.State
	settings          *config.Config
	panels            *panels.Registry
	store             *persistence
	menu              *mygo.Menu
	home, root        string
	message           string
	loading           bool
	closed            bool
	settingsOpen      bool
	newWorkspace      bool
	creatingWorkspace bool
	choosingFolder    bool
	workspaceName     string
	workspacePath     string
	dialogError       string
	geometry          workspace.Geometry
	canvasBounds      workspace.Rect
	resize            resizeGesture
	actions           []func()
	focusedPanel      string
	focusPending      bool
	dropPreview       *dockPreview
}

func Run(root, home string, explicitRoot bool) error {
	if err := resources.RegisterFonts(); err != nil {
		return err
	}
	a := &shell{root: root, home: home, settings: config.Default(), loading: true}
	mygo.App.SetName("Patch")
	mygo.App.SetPath(mygo.PathUserData, home)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "Patch", StateKey: "main", Width: 1440, Height: 900, MinWidth: 620, MinHeight: 420,
			TitleBarStyle: mygo.TitleBarHiddenInset, Content: ui.View(a.view),
		})
		a.panels = panels.NewRegistry(panels.Host{
			Window: a.win, Open: a.addPanel, Settings: func() { a.settingsOpen = true },
			SetResource: a.setPanelResource,
			OpenFolder:  a.openFolder,
		})
		a.store = newPersistence(home, a.win.Invalidate)
		a.installMenu()
		a.win.OnClosed(a.close)
		go a.load(explicitRoot)
	})
	return mygo.App.Run()
}

func (a *shell) load(explicitRoot bool) {
	cfg, cfgErr := config.Load(filepath.Join(a.home, "config.json"))
	if cfgErr != nil {
		cfg = config.Default()
	}
	state, stateErr := workspace.Load(filepath.Join(a.home, "session.json"))
	if stateErr != nil {
		state = nil
		if explicitRoot && errors.Is(stateErr, os.ErrNotExist) {
			state = workspace.New(a.root)
			stateErr = defaultPanels(state.ActiveWorkspace())
		}
	} else if explicitRoot {
		found := false
		for _, w := range state.Workspaces {
			if w.RootPath == a.root {
				stateErr = state.SelectWorkspace(w.ID)
				found = true
				break
			}
		}
		if !found {
			stateErr = defaultPanels(state.CreateWorkspace("", a.root))
		}
	}
	var loadErr error
	if cfgErr != nil && !errors.Is(cfgErr, os.ErrNotExist) {
		loadErr = errors.Join(loadErr, cfgErr)
	}
	if stateErr != nil && !errors.Is(stateErr, os.ErrNotExist) {
		loadErr = errors.Join(loadErr, stateErr)
	}
	if explicitRoot {
		if info, err := os.Stat(a.root); err != nil || !info.IsDir() {
			loadErr = errors.Join(loadErr, fmt.Errorf("workspace directory is unavailable: %s", a.root))
		}
	}
	a.win.Update(func() {
		if a.closed {
			return
		}
		a.settings, a.state, a.loading = cfg, state, false
		a.syncNavigationMenu()
		a.report(loadErr)
		a.panels.Sync(a.state)
		a.focusActive()
		a.save()
	})
}

func defaultPanels(w *workspace.Workspace) error {
	if w.Root != nil {
		if err := w.ClosePanel(w.Root.ActivePanelID); err != nil {
			return err
		}
	}
	term, err := w.AddPanel("", workspace.Center, workspace.PanelSpec{Kind: workspace.Terminal, Title: "Terminal", Resource: w.RootPath})
	if err != nil {
		return err
	}
	_, err = w.AddPanel(w.ActiveGroupID, workspace.Right, workspace.PanelSpec{Kind: workspace.Files, Title: "Project files", Resource: w.RootPath})
	if err != nil {
		return err
	}
	group, _ := w.FindPanel(term.ID)
	if _, err = w.AddPanel(group.ID, workspace.Bottom, workspace.PanelSpec{Kind: workspace.Changes, Title: "Changes", Resource: w.RootPath}); err != nil {
		return err
	}
	if err := w.SetRatio(w.Root.ID, .58); err != nil {
		return err
	}
	return w.SelectPanel(term.ID)
}

func (a *shell) close() {
	if a.closed {
		return
	}
	a.closed = true
	a.save()
	if a.panels != nil {
		a.panels.Close()
	}
	if a.store != nil {
		a.store.close()
	}
}

func (a *shell) report(err error) {
	if err != nil {
		a.message = err.Error()
		log.Printf("patch: %v", err)
	}
}

func (a *shell) save() {
	if a.store != nil {
		a.report(a.store.enqueue(a.state, a.settings))
	}
}

func (a *shell) changed(err error) {
	if err != nil {
		a.report(err)
		return
	}
	a.panels.Sync(a.state)
	a.focusActive()
	a.save()
}

func (a *shell) focusActive() {
	if w := a.active(); w != nil && w.ActiveGroup() != nil {
		id := w.ActiveGroup().ActivePanelID
		if id != a.focusedPanel {
			a.requestPanelFocus(id)
		}
	}
}

func (a *shell) requestPanelFocus(id string) {
	a.focusedPanel, a.focusPending = id, true
	a.panels.Focus(id)
}

func (a *shell) active() *workspace.Workspace {
	if a.state == nil {
		return nil
	}
	return a.state.ActiveWorkspace()
}

func (a *shell) addPanel(kind workspace.Kind) {
	w := a.active()
	if w == nil {
		a.openFolder()
		return
	}
	if kind == workspace.Primitives {
		var existing string
		var find func(*workspace.Node)
		find = func(n *workspace.Node) {
			if n == nil {
				return
			}
			for _, panel := range n.Panels {
				if panel.Kind == kind {
					existing = panel.ID
					return
				}
			}
			find(n.First)
			find(n.Second)
		}
		find(w.Root)
		if existing != "" {
			a.changed(w.SelectPanel(existing))
			return
		}
	}
	resource := w.RootPath
	if kind == workspace.Browser {
		resource = ""
	}
	dock := workspace.Right
	if w.Root == nil {
		dock = workspace.Center
	}
	_, err := w.AddPanel(w.ActiveGroupID, dock, workspace.PanelSpec{Kind: kind, Title: panels.Title(kind), Resource: resource})
	a.changed(err)
}

// Panel callbacks run on the main thread; saving queues a detached snapshot.
func (a *shell) setPanelResource(panelID, resource string) {
	if a.closed || a.state == nil {
		return
	}
	for _, w := range a.state.Workspaces {
		if _, panel := w.FindPanel(panelID); panel != nil {
			if panel.Resource == resource {
				return
			}
			if err := w.SetPanelResource(panelID, resource); err != nil {
				a.report(err)
				return
			}
			a.save()
			return
		}
	}
	a.report(fmt.Errorf("update panel resource %q: %w", panelID, workspace.ErrNotFound))
}
