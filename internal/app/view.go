package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
)

type workspaceDrag struct{ ID string }
type panelDrag struct{ WorkspaceID, PanelID string }

func (a *shell) view(c *ui.Context) {
	p := a.theme(c)
	c.Root().LinearGradient(p.backdrop)
	width, height := c.Size()
	a.titlebar(c, width, p)
	if a.loading {
		ui.Column(c).Absolute().Top(60).Left(20).Right(20).Bottom(20).Center().Gap(12).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "Opening your workspace…").TextColor(p.muted)
		})
		return
	}
	if a.store != nil {
		select {
		case err := <-a.store.errs:
			a.message = err.Error()
		default:
		}
	}
	w := a.active()
	if w == nil {
		a.projectWelcome(c, p)
		if a.message != "" {
			a.errorBanner(c)
		}
		a.dialogs(c)
		return
	}
	sidebarWidth := float32(0)
	showSidebar := a.sidebarShown()
	sidebarLeft := float32(0)
	if showSidebar {
		sidebarWidth = a.renderedSidebarWidth(width)
		if a.settings.SidebarPosition == config.SidebarRight {
			sidebarLeft = width - sidebarWidth
		}
		a.sidebar(c, sidebarLeft, sidebarWidth, height, p)
		a.sidebarDivider(c, sidebarLeft, sidebarWidth, height, p)
	}
	left, right, top := paneGap, width-paneGap, float32(58)
	if showSidebar && a.settings.SidebarPosition == config.SidebarLeft {
		left += sidebarWidth
	}
	if showSidebar && a.settings.SidebarPosition == config.SidebarRight {
		right -= sidebarWidth
	}
	if a.compactSidebarHeader(c) {
		top = paneGap
	}
	bottom := paneGap
	if a.message != "" {
		bottom = 56
		a.errorBanner(c)
	}
	a.canvasBounds = workspace.Rect{X: left, Y: top, Width: max(0, right-left), Height: max(0, height-top-bottom)}
	a.geometry = w.Layout(a.canvasBounds, paneGap, 160)
	a.canvas(c, w, p)
	a.dialogs(c)
	// Drop queries are consumed during construction; apply the collected changes only after the tree is built.
	a.flushActions()
}

func (a *shell) sidebarShown() bool {
	return a.state != nil && a.settings.Navigation == config.Sidebar && a.settings.SidebarVisible
}

func (a *shell) renderedSidebarWidth(width float32) float32 {
	return min(a.settings.SidebarWidth, max(config.MinSidebarWidth, width*.35))
}

func (a *shell) compactSidebarHeader(c *ui.Context) bool {
	return a.sidebarShown() && a.settings.SidebarPosition == config.SidebarLeft && c.TitleBar().Right == 0
}

func (a *shell) titlebar(c *ui.Context, width float32, p palette) {
	bar := c.TitleBar()
	if a.compactSidebarHeader(c) {
		sidebarWidth := a.renderedSidebarWidth(width)
		ui.Row(c.Key("window-titlebar")).Absolute().Top(0).Left(0).Width(sidebarWidth).Height(52).AlignItems(ui.Center).Disabled(a.loading).
			Padding(0, 10, 0, bar.Left+12).Gap(4).DragWindow().Children(func() {
			ui.Spacer(c)
			iconButton(c, "toggle-sidebar", "Toggle sidebar", sidebarIcon).OnClick(a.toggleSidebar)
			iconButton(c, "new-workspace", "New workspace", plusIcon).OnClick(a.showNewWorkspace)
		})
		return
	}
	ui.Row(c.Key("window-titlebar")).Absolute().Top(0).Left(0).Right(0).Height(52).AlignItems(ui.Center).Disabled(a.loading).
		Padding(0, bar.Right+14, 0, bar.Left+12).Gap(10).DragWindow().Children(func() {
		if a.settings.Navigation == config.Tabs {
			if a.state != nil {
				a.workspaceTabs(c, p)
			} else {
				ui.Spacer(c)
			}
		} else {
			if a.settings.SidebarPosition == config.SidebarLeft {
				iconButton(c, "toggle-sidebar", "Toggle sidebar", sidebarIcon).OnClick(a.toggleSidebar)
				iconButton(c, "new-workspace", "New workspace", plusIcon).OnClick(a.showNewWorkspace)
			}
			ui.Spacer(c)
			if a.settings.SidebarPosition == config.SidebarRight {
				iconButton(c, "toggle-sidebar", "Toggle sidebar", sidebarIcon).OnClick(a.toggleSidebar)
				iconButton(c, "new-workspace", "New workspace", plusIcon).OnClick(a.showNewWorkspace)
			}
		}
		if !a.sidebarShown() {
			iconButton(c, "open-settings", "Settings", settingsIcon).OnClick(func() { a.settingsOpen = true })
		}
	})
}

func (a *shell) sidebar(c *ui.Context, left, width, height float32, p palette) {
	inset := float32(12)
	if a.settings.SidebarPosition == config.SidebarRight {
		inset = 6
	}
	ui.Column(c.Key("workspace-sidebar")).Absolute().Left(left + inset).Top(60).Width(width - 18).Height(height - 72).Gap(8).Children(func() {
		ui.Scroll(c).Grow(1).MinHeight(0).Gap(4).Children(func() {
			last := ""
			for index, w := range a.state.Workspaces {
				parent := workspaceSection(w.RootPath)
				if parent != last {
					ui.Text(c, parent).FontSize(11).TextColor(p.muted).Padding(10, 10, 4, 10).SingleLine().Ellipsis("…")
					last = parent
				}
				a.workspaceRow(c, w, index, false, p)
			}
		})
		ui.Row(c).Height(32).Padding(0, 5).Gap(6).Children(func() {
			iconButton(c, "open-folder", "Open project folder", folderIcon).OnClick(a.openFolder)
			iconButton(c, "open-settings", "Settings", settingsIcon).OnClick(func() { a.settingsOpen = true })
		})
	})
}

func (a *shell) workspaceTabs(c *ui.Context, p palette) {
	ui.Row(c.Key("workspace-navigation")).Grow(1).MinWidth(0).Height(34).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.ScrollHorizontal(c.Key("workspace-tabs")).Grow(1).MinWidth(0).Height(34).AlignItems(ui.Center).Radius(17).Background(p.selection.Alpha(.3)).Children(func() {
			ui.Row(c).Gap(4).Height(34).Padding(2).AlignItems(ui.Center).Children(func() {
				for index, w := range a.state.Workspaces {
					a.workspaceRow(c, w, index, true, p)
				}
			})
		})
		iconButton(c, "new-workspace-tab", "New workspace", plusIcon).OnClick(a.showNewWorkspace)
	})
}

func (a *shell) workspaceRow(c *ui.Context, w *workspace.Workspace, index int, tab bool, p palette) {
	name := workspaceName(w)
	radius := float32(9)
	if tab {
		radius = 15
	}
	row := ui.Row(c.Key(w.ID)).Height(36).Shrink(0).Radius(radius).Gap(2).Padding(0, 4).AlignItems(ui.Center)
	if tab {
		row.Width(180).Height(30)
	}
	if w.ID == a.state.ActiveWorkspaceID {
		row.Background(p.selection).Shadow(0, 1, 2, 0, ui.RGBA(40, 20, 60, .05))
	} else if row.Hovered() {
		row.Background(p.selection.Alpha(.4))
	}
	if _, over := ui.DragOver[workspaceDrag](row); over {
		row.Border(1, p.accent)
	}
	if drag, ok := ui.Drop[workspaceDrag](row); ok {
		a.deferAction(func() { a.changed(a.state.ReorderWorkspace(drag.ID, index)) })
	}
	showClose := row.Hovered() || row.FocusWithin()
	var pressed, keyboardFocus bool
	row.Children(func() {
		button := ui.Button(c.Key("workspace-select"), "").Label("Workspace "+name).Tooltip(w.RootPath).Grow(1).MinWidth(0).
			Height(28).Padding(0, 7).AlignItems(ui.Center).Background(ui.Transparent).Border(0, ui.Transparent).Drag(workspaceDrag{w.ID}).Cursor(ui.CursorGrab).KeepFocus().FocusRing(false)
		button.Children(func() {
			ui.Icon(c, folderIcon).Size(15, 15).TextColor(p.muted)
			ui.Text(c, name).SingleLine().Ellipsis("…").MinWidth(0).Grow(1)
		})
		pressed, keyboardFocus = button.Pressed(), button.FocusVisible()
		button.OnClick(func() { a.changed(a.state.SelectWorkspace(w.ID)) }).ContextMenu(func(m *ui.Menu) { a.workspaceMenu(m, w.ID, index) })
		close := smallIconButton(c, "close-workspace", "Close workspace "+name, closeIcon).IgnoreInput(!showClose).FocusRing(false).
			OnClick(func() { a.changed(a.state.CloseWorkspace(w.ID)) })
		pressed = pressed || close.Pressed()
		keyboardFocus = keyboardFocus || close.FocusVisible()
		if close.FocusVisible() {
			close.Background(c.Theme().TextMuted.Alpha(.12))
		}
		if !showClose {
			close.Opacity(0)
		}
	})
	if pressed {
		row.Background(p.selection.Mix(ui.RGB(0, 0, 0), .10))
	}
	row.DrawOver(func(painter *ui.Painter, rect ui.Rect) {
		if keyboardFocus {
			painter.FocusRing(rect, [4]float32{radius, radius, radius, radius})
		}
	})
}

func workspaceName(w *workspace.Workspace) string {
	clean := filepath.Clean(w.RootPath)
	if filepath.IsAbs(clean) && filepath.Dir(clean) == clean && (w.Name == clean || w.Name == filepath.Base(clean) || w.Name == "") {
		return filesystemRootLabel(clean)
	}
	return w.Name
}

func workspaceSection(root string) string {
	parent := filepath.Dir(root)
	if parent == "." || filepath.Dir(parent) == parent {
		return "Local"
	}
	return filepath.Base(parent)
}

func filesystemRootLabel(root string) string {
	if volume := filepath.VolumeName(root); volume != "" {
		return volume + " root"
	}
	return "File system"
}

func (a *shell) projectWelcome(c *ui.Context, p palette) {
	ui.Column(c.Key("project-welcome")).Absolute().Left(paneGap).Right(paneGap).Top(58).Bottom(paneGap).Radius(paneRadius).Background(p.card).Center().Gap(14).Padding(32).Children(func() {
		ui.Icon(c, folderIcon).Size(30, 30).TextColor(p.muted)
		ui.Text(c, "Open a project").FontSize(23).Bold()
		ui.Text(c, "Choose a folder for your terminal, files and Git changes.").TextColor(p.muted).MaxWidth(380)
		ui.PrimaryButton(c.Key("open-project"), "Open folder…").OnClick(a.openFolder)
	})
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if path == home {
			return "~"
		}
		if strings.HasPrefix(path, home+string(filepath.Separator)) {
			return "~" + strings.TrimPrefix(path, home)
		}
	}
	return path
}

func (a *shell) errorBanner(c *ui.Context) {
	ui.Row(c.Key("app-error")).Absolute().Left(16).Right(16).Bottom(12).Height(32).Padding(4, 10).Gap(8).
		Radius(8).Background(c.Theme().Surface).Border(1, c.Theme().Danger.Alpha(.5)).Children(func() {
		ui.Text(c, a.message).FontSize(11).SingleLine().Ellipsis("…").Tooltip(a.message).Grow(1).MinWidth(0).TextColor(c.Theme().Danger)
		iconButton(c, "dismiss-error", "Dismiss error", closeIcon).Size(22, 22).OnClick(func() { a.message = "" })
	})
}
