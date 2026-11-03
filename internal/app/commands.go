package app

import (
	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *shell) menuItem(label, accelerator string, action func()) *mygo.MenuItem {
	return &mygo.MenuItem{Label: label, Accelerator: menuAccelerator(accelerator), Click: func(*mygo.MenuItem, *mygo.Window) {
		if a.loading || a.closed {
			return
		}
		action()
		a.win.Invalidate()
	}}
}

func (a *shell) installMenu() {
	sidebarItem := a.menuItem("Hide Sidebar", "CmdOrCtrl+B", a.toggleSidebar)
	sidebarItem.ID = "sidebar-visibility"
	a.menu = mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "Workspace", Submenu: []*mygo.MenuItem{
			a.menuItem("Open Folder…", "CmdOrCtrl+O", a.openFolder),
			a.menuItem("New Workspace…", "CmdOrCtrl+N", a.showNewWorkspace),
			a.menuItem("New Workspace Tab…", "CmdOrCtrl+T", a.showNewWorkspace),
			a.menuItem("New Terminal Panel", "", func() { a.addPanel(workspace.Terminal) }),
			a.menuItem("Project Files", "CmdOrCtrl+Shift+E", func() { a.addPanel(workspace.Files) }),
			a.menuItem("Git Changes", "", func() { a.addPanel(workspace.Changes) }),
			a.menuItem("Browser…", "", func() { a.addPanel(workspace.Browser) }),
			mygo.Separator(),
			a.menuItem("Next Workspace", "CmdOrCtrl+Alt+Right", func() { a.nextWorkspace(1) }),
			a.menuItem("Previous Workspace", "CmdOrCtrl+Alt+Left", func() { a.nextWorkspace(-1) }),
			a.menuItem("Move Workspace Left", "CmdOrCtrl+Alt+Shift+Left", func() { a.reorderWorkspace(-1) }),
			a.menuItem("Move Workspace Right", "CmdOrCtrl+Alt+Shift+Right", func() { a.reorderWorkspace(1) }),
			a.menuItem("Close Workspace", "CmdOrCtrl+Shift+W", func() {
				if a.state != nil {
					a.changed(a.state.CloseWorkspace(a.state.ActiveWorkspaceID))
				}
			}),
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "Panel", Submenu: []*mygo.MenuItem{
			a.menuItem("Refresh", "CmdOrCtrl+R", func() {
				if w := a.active(); w != nil && w.ActiveGroup() != nil {
					a.panels.Refresh(w.ActiveGroup().ActivePanelID)
				}
			}),
			a.menuItem("Split Right", "CmdOrCtrl+D", func() {
				if w := a.active(); w != nil {
					a.split(w.ActiveGroupID, workspace.Right)
				} else {
					a.openFolder()
				}
			}),
			a.menuItem("Split Down", "CmdOrCtrl+Shift+D", func() {
				if w := a.active(); w != nil {
					a.split(w.ActiveGroupID, workspace.Bottom)
				} else {
					a.openFolder()
				}
			}),
			a.menuItem("Zoom / Restore", "CmdOrCtrl+Shift+Enter", func() {
				if w := a.active(); w != nil {
					a.zoom(w.ActiveGroupID)
				}
			}),
			a.menuItem("Equalize Splits", "CmdOrCtrl+Shift+0", func() {
				if w := a.active(); w != nil {
					w.Equalize()
					a.changed(nil)
				}
			}),
			mygo.Separator(),
			a.menuItem("Next Saved Panel", "Ctrl+Tab", func() { a.nextPanel(1) }),
			a.menuItem("Previous Saved Panel", "Ctrl+Shift+Tab", func() { a.nextPanel(-1) }),
			a.menuItem("Focus Next Group", "CmdOrCtrl+Alt+Down", a.nextGroup),
			a.menuItem("Swap With Next Panel", "CmdOrCtrl+Alt+M", func() { a.moveToNextGroup(workspace.Center) }),
			a.menuItem("Dock Left of Next Group", "", func() { a.moveToNextGroup(workspace.Left) }),
			a.menuItem("Dock Right of Next Group", "", func() { a.moveToNextGroup(workspace.Right) }),
			a.menuItem("Dock Above Next Group", "", func() { a.moveToNextGroup(workspace.Top) }),
			a.menuItem("Dock Below Next Group", "", func() { a.moveToNextGroup(workspace.Bottom) }),
			mygo.Separator(),
			a.menuItem("Close Panel", "CmdOrCtrl+W", a.closePanel),
		}},
		{Label: "View", Submenu: []*mygo.MenuItem{
			{Label: "Customize View", Submenu: []*mygo.MenuItem{
				{Label: "Tab Style", Submenu: []*mygo.MenuItem{
					a.tabStyleItem("Above", "tab-style-above", config.Tabs, config.SidebarLeft),
					a.tabStyleItem("Left Side", "tab-style-left", config.Sidebar, config.SidebarLeft),
					a.tabStyleItem("Right Side", "tab-style-right", config.Sidebar, config.SidebarRight),
				}},
			}},
			sidebarItem,
			a.menuItem("Settings…", "CmdOrCtrl+,", func() { a.settingsOpen = true }),
			mygo.Separator(), {Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
		{Label: "Developer", Submenu: []*mygo.MenuItem{a.menuItem("Native Controls", "", func() { a.addPanel(workspace.Primitives) })}},
	})
	mygo.App.SetMenu(a.menu)
	a.syncNavigationMenu()
}

func (a *shell) toggleSidebar() {
	if a.settings.Navigation != config.Sidebar {
		return
	}
	a.settings.SidebarVisible = !a.settings.SidebarVisible
	a.syncNavigationMenu()
	a.save()
}

func (a *shell) tabStyleItem(label, id string, navigation config.Navigation, position config.SidebarPosition) *mygo.MenuItem {
	item := a.menuItem(label, "", func() { a.setTabStyle(navigation, position) })
	item.ID, item.Type = id, mygo.MenuItemRadio
	return item
}

func (a *shell) setTabStyle(navigation config.Navigation, position config.SidebarPosition) {
	a.settings.Navigation = navigation
	if navigation == config.Sidebar {
		a.settings.SidebarPosition = position
	}
	a.syncNavigationMenu()
	a.save()
}

func (a *shell) syncNavigationMenu() {
	if a.menu == nil {
		return
	}
	sidebar := a.settings.Navigation == config.Sidebar
	a.menu.ItemByID("tab-style-above").SetChecked(!sidebar)
	a.menu.ItemByID("tab-style-left").SetChecked(sidebar && a.settings.SidebarPosition == config.SidebarLeft)
	a.menu.ItemByID("tab-style-right").SetChecked(sidebar && a.settings.SidebarPosition == config.SidebarRight)
	item := a.menu.ItemByID("sidebar-visibility")
	item.SetEnabled(sidebar && !a.loading)
	label := "Show Sidebar"
	if a.settings.SidebarVisible {
		label = "Hide Sidebar"
	}
	item.SetLabel(label)
}

func (a *shell) closePanel() {
	if w := a.active(); w != nil && w.ActiveGroup() != nil {
		a.changed(w.ClosePanel(w.ActiveGroup().ActivePanelID))
	}
}

func (a *shell) nextWorkspace(delta int) {
	if a.state == nil {
		return
	}
	for index, w := range a.state.Workspaces {
		if w.ID == a.state.ActiveWorkspaceID {
			next := (index + delta + len(a.state.Workspaces)) % len(a.state.Workspaces)
			a.changed(a.state.SelectWorkspace(a.state.Workspaces[next].ID))
			return
		}
	}
}

func (a *shell) reorderWorkspace(delta int) {
	if a.state == nil {
		return
	}
	for index, w := range a.state.Workspaces {
		if w.ID == a.state.ActiveWorkspaceID {
			next := max(0, min(len(a.state.Workspaces)-1, index+delta))
			a.changed(a.state.ReorderWorkspace(w.ID, next))
			return
		}
	}
}

func (a *shell) nextPanel(delta int) {
	w := a.active()
	if w == nil || w.ActiveGroup() == nil {
		return
	}
	group := w.ActiveGroup()
	for i, panel := range group.Panels {
		if panel.ID == group.ActivePanelID {
			next := (i + delta + len(group.Panels)) % len(group.Panels)
			a.changed(w.SelectPanel(group.Panels[next].ID))
			return
		}
	}
}

func (a *shell) groupIDs() []string {
	var ids []string
	var visit func(*workspace.Node)
	visit = func(n *workspace.Node) {
		if n == nil {
			return
		}
		if n.IsLeaf() {
			ids = append(ids, n.ID)
			return
		}
		visit(n.First)
		visit(n.Second)
	}
	if w := a.active(); w != nil {
		visit(w.Root)
	}
	return ids
}

func (a *shell) followingGroup() string {
	ids := a.groupIDs()
	for i, id := range ids {
		if id == a.active().ActiveGroupID {
			return ids[(i+1)%len(ids)]
		}
	}
	return ""
}

func (a *shell) nextGroup() {
	if next := a.followingGroup(); next != "" {
		a.changed(a.active().FocusGroup(next))
	}
}

func (a *shell) moveToNextGroup(dock workspace.Dock) {
	w := a.active()
	if w == nil || w.ActiveGroup() == nil {
		return
	}
	target := a.followingGroup()
	if target == "" || target == w.ActiveGroupID {
		return
	}
	panelID := w.ActiveGroup().ActivePanelID
	a.changed(a.movePanel(w, panelID, target, dock))
	a.requestPanelFocus(panelID)
}

func (a *shell) workspaceMenu(m *ui.Menu, id string, index int) {
	if m.Item("Select workspace").Chosen() {
		a.deferAction(func() { a.changed(a.state.SelectWorkspace(id)) })
	}
	if m.Item("Move earlier").Disabled(index == 0).Chosen() {
		a.deferAction(func() { a.changed(a.state.ReorderWorkspace(id, index-1)) })
	}
	if m.Item("Move later").Disabled(index == len(a.state.Workspaces)-1).Chosen() {
		a.deferAction(func() { a.changed(a.state.ReorderWorkspace(id, index+1)) })
	}
	m.Separator()
	if m.Item("Close workspace").Chosen() {
		a.deferAction(func() { a.changed(a.state.CloseWorkspace(id)) })
	}
}
