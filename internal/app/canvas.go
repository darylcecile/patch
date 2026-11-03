package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/panels"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
)

func (a *shell) deferAction(action func()) { a.actions = append(a.actions, action) }
func (a *shell) flushActions() {
	actions := a.actions
	a.actions = nil
	for _, action := range actions {
		action()
	}
}

func (a *shell) canvas(c *ui.Context, w *workspace.Workspace, p palette) {
	a.dropPreview = nil
	if w.Root == nil {
		r := a.canvasBounds
		ui.Column(c.Key("empty-workspace")).Absolute().Left(r.X).Top(r.Y).Size(r.Width, r.Height).Radius(paneRadius).Background(p.card).Center().Gap(14).Children(func() {
			ui.Text(c, "Workspace is empty").FontSize(22).Bold()
			ui.Text(c, "Add your next panel.").TextColor(p.muted)
			ui.PrimaryButton(c, "Open terminal").OnClick(func() { a.addPanel(workspace.Terminal) })
		})
	}
	for _, leaf := range a.geometry.Leaves {
		if node := w.FindNode(leaf.NodeID); node != nil {
			a.card(c, w, node, leaf.Rect, p)
		}
	}
	a.panelContents(c, w)
	for _, divider := range a.geometry.Dividers {
		a.divider(c, w, divider, p)
	}
	a.drawDropPreview(c, p)
}

func (a *shell) card(c *ui.Context, w *workspace.Workspace, group *workspace.Node, rect workspace.Rect, p palette) {
	_, current := w.FindPanel(group.ActivePanelID)
	if current == nil {
		return
	}
	panel := *current
	px, py, pointerIn := c.Root().PointerPosition()
	hovered := pointerIn && px >= rect.X && py >= rect.Y && px < rect.X+rect.Width && py < rect.Y+rect.Height
	card := ui.Column(c.Key(group.ID)).Absolute().Left(rect.X).Top(rect.Y).Size(rect.Width, rect.Height).
		Radius(paneRadius).Background(p.card).Shadow(0, 1, 2, 0, ui.RGBA(35, 25, 45, .05)).Clip().MinWidth(0).MinHeight(0)
	card.Children(func() {
		a.panelToolbar(c, w, group, panel, rect.Width, hovered || group.ID == w.ActiveGroupID, p)
	})
	a.observeDrop(card, w, group, rect, 0)
	if card.FocusWithin() && w.ActiveGroupID != group.ID && !a.focusPending {
		a.deferAction(func() {
			// Keyboard focus already belongs to this header's controls. Update
			// the active group without redirecting it into the panel content.
			a.focusedPanel = group.ActivePanelID
			a.changed(w.FocusGroup(group.ID))
		})
	}
}

// Every panel is a direct child of the window root, including inactive panels.
// Docking changes geometry, never widget ancestry; Hidden keeps each control's state.
func (a *shell) panelContents(c *ui.Context, active *workspace.Workspace) {
	visible := make(map[string]workspace.Rect)
	for _, leaf := range a.geometry.Leaves {
		visible[leaf.NodeID] = leaf.Rect
	}
	var visit func(*workspace.Workspace, *workspace.Node)
	visit = func(w *workspace.Workspace, group *workspace.Node) {
		if group == nil {
			return
		}
		if !group.IsLeaf() {
			visit(w, group.First)
			visit(w, group.Second)
			return
		}
		rect, shown := visible[group.ID]
		shown = shown && w.ID == active.ID
		header := a.panelHeaderHeight()
		for _, tab := range group.Panels {
			isVisible := shown && tab.ID == group.ActivePanelID
			content := ui.Column(c.Key("content-"+tab.ID)).Absolute().Left(rect.X+1).Top(rect.Y+header).
				Size(max(rect.Width-2, 0), max(rect.Height-header-1, 0)).Radius(0, 0, paneRadius-1, paneRadius-1).Clip().Hidden(!isVisible).MinWidth(0).MinHeight(0)
			content.Children(func() { a.panels.View(c, tab.ID) })
			if isVisible {
				a.observeDrop(content, w, group, rect, header)
				if content.FocusWithin() && a.focusedPanel == tab.ID {
					a.focusPending = false
				}
				if content.FocusWithin() && a.focusedPanel != tab.ID && !a.focusPending {
					id := tab.ID
					a.deferAction(func() { a.focusedPanel = id; a.changed(w.SelectPanel(id)) })
				}
			}
		}
	}
	for _, w := range a.state.Workspaces {
		visit(w, w.Root)
	}
}

func (a *shell) panelHeaderHeight() float32 {
	if a.settings.Density == config.Compact {
		return 36
	}
	return 40
}

func panelIcon(kind workspace.Kind) *ui.SVG {
	switch kind {
	case workspace.Terminal:
		return terminalIcon
	case workspace.Files:
		return folderIcon
	case workspace.Changes:
		return changesIcon
	default:
		return gridIcon
	}
}

func (a *shell) panelToolbar(c *ui.Context, w *workspace.Workspace, group *workspace.Node, panel workspace.Panel, width float32, showActions bool, p palette) {
	height := a.panelHeaderHeight()
	title, resource := a.panels.Header(panel)
	actionWidth := float32(28)
	if width >= 390 {
		actionWidth = 4*28 + 3*3
	}
	// Give the title its intrinsic width, bounded only by the actual header
	// room after icons/padding. The path gives up its room first.
	titleWidth := width - 14 - 9 - 3 - actionWidth - 15 - 16
	if len(group.Panels) > 1 {
		titleWidth -= 31
	}
	ui.Row(c.Key("panel-toolbar")).Height(height).Shrink(0).Padding(0, 9, 0, 14).Gap(3).Children(func() {
		if len(group.Panels) > 1 {
			iconButton(c, "panel-selector", "Switch panel", chevronIcon).Menu(func(m *ui.Menu) {
				for index, candidate := range group.Panels {
					label := fmt.Sprintf("%s · %d", candidate.Title, index+1)
					if m.Item(label).Checked(candidate.ID == group.ActivePanelID).Chosen() {
						a.deferAction(func() { a.changed(w.SelectPanel(candidate.ID)); a.requestPanelFocus(candidate.ID) })
					}
				}
			})
		}
		grip := ui.Row(c.Key("panel-grip")).Label("Drag " + title).Height(height - 2).Grow(1).MinWidth(0).Gap(8).
			Drag(panelDrag{w.ID, panel.ID}).Cursor(ui.CursorGrab).Focusable().KeepFocus()
		grip.OnClick(func() { a.changed(w.SelectPanel(panel.ID)); a.requestPanelFocus(panel.ID) })
		grip.Children(func() {
			ui.Icon(c, panelIcon(panel.Kind)).Size(15, 15).TextColor(p.muted).Shrink(0)
			ui.Text(c, title).FontSize(13).Bold().SingleLine().Ellipsis("…").MaxWidth(max(0, titleWidth)).MinWidth(0).Shrink(0).Tooltip(title)
			ui.Text(c, panelPath(resource)).FontSize(12).TextColor(p.muted).SingleLine().Ellipsis("…").Grow(1).MinWidth(0).Tooltip(resource)
		})
		grip.ContextMenu(func(m *ui.Menu) { a.panelMenu(m, w, group, panel) })
		actions := ui.Row(c.Key("panel-actions")).Gap(3).Shrink(0).FocusGroup(ui.Horizontal)
		showActions = showActions || actions.FocusWithin()
		actions.IgnoreInput(!showActions)
		if !showActions {
			actions.Opacity(0)
		}
		actions.Children(func() {
			if width >= 390 {
				iconButton(c, "split-right", "Split right", splitRightIcon).OnClick(func() { a.split(group.ID, workspace.Right) })
				iconButton(c, "split-down", "Split down", splitDownIcon).OnClick(func() { a.split(group.ID, workspace.Bottom) })
				iconButton(c, "zoom-panel", "Zoom panel", zoomIcon).OnClick(func() { a.zoom(group.ID) })
				iconButton(c, "close-panel", "Close panel", closeIcon).OnClick(func() { a.changed(w.ClosePanel(panel.ID)) })
			} else {
				iconButton(c, "panel-menu", "Panel actions", moreIcon).Menu(func(m *ui.Menu) { a.panelMenu(m, w, group, panel) })
			}
		})
	})
}

func panelPath(resource string) string {
	clean := filepath.Clean(resource)
	if filepath.IsAbs(clean) && filepath.Dir(clean) == clean {
		return filesystemRootLabel(clean)
	}
	return shortenHome(resource)
}

func dockAt(x, y, width, height float32) workspace.Dock {
	x, y = x/max(width, 1), y/max(height, 1)
	if x > .25 && x < .75 && y > .25 && y < .75 {
		return workspace.Center
	}
	nearest, dock := x, workspace.Left
	for _, candidate := range []struct {
		distance float32
		dock     workspace.Dock
	}{{1 - x, workspace.Right}, {y, workspace.Top}, {1 - y, workspace.Bottom}} {
		if candidate.distance < nearest {
			nearest, dock = candidate.distance, candidate.dock
		}
	}
	return dock
}

type dockPreview struct {
	rect workspace.Rect
	dock workspace.Dock
}

func (a *shell) observeDrop(card ui.Element, w *workspace.Workspace, group *workspace.Node, rect workspace.Rect, offsetY float32) {
	x, y, _ := card.PointerPosition()
	dock := dockAt(x, y+offsetY, rect.Width, rect.Height)
	if drag, ok := ui.Drop[panelDrag](card); ok && drag.WorkspaceID == w.ID {
		a.deferAction(func() {
			err := a.movePanel(w, drag.PanelID, group.ID, dock)
			if !errors.Is(err, workspace.ErrSelfMove) {
				a.changed(err)
				if err == nil {
					a.requestPanelFocus(drag.PanelID)
				}
			}
		})
	}
	drag, over := ui.DragOver[panelDrag](card)
	if !over || drag.WorkspaceID != w.ID {
		return
	}
	source, _ := w.FindPanel(drag.PanelID)
	if source == nil || (source.ID == group.ID && (len(group.Panels) == 1 || dock == workspace.Center)) {
		return
	}
	a.dropPreview = &dockPreview{rect: rect, dock: dock}
}

func (a *shell) drawDropPreview(c *ui.Context, p palette) {
	if a.dropPreview == nil {
		return
	}
	rect, dock := a.dropPreview.rect, a.dropPreview.dock
	left, top, width, height := rect.X+5, rect.Y+5, rect.Width-10, rect.Height-10
	label := "Swap panels"
	switch dock {
	case workspace.Left:
		width /= 2
		label = "Split left"
	case workspace.Right:
		width /= 2
		left += width
		label = "Split right"
	case workspace.Top:
		height /= 2
		label = "Split above"
	case workspace.Bottom:
		height /= 2
		top += height
		label = "Split below"
	}
	ui.Column(c.Key("dock-preview")).Label("Drop target "+string(dock)).Absolute().Left(left).Top(top).Size(max(width, 0), max(height, 0)).
		IgnoreInput(true).Radius(paneRadius-5).Border(2, p.accent).Background(p.accent.Alpha(.14)).Center().Children(func() {
		ui.Text(c, label).FontSize(12).Bold().Padding(7, 12).Radius(8).Background(p.card).TextColor(p.accent)
	})
}

type resizeGesture struct {
	id                   string
	axis                 workspace.Axis
	start, initial, span float32
}

func (a *shell) divider(c *ui.Context, w *workspace.Workspace, d workspace.Divider, p palette) {
	r := d.Rect
	extra := (float32(12) - paneGap) / 2
	if d.Axis == workspace.Horizontal {
		r.X -= extra
		r.Width += extra * 2
	} else {
		r.Y -= extra
		r.Height += extra * 2
	}
	bar := ui.Box(c.Key("divider-"+d.NodeID)).Label("Resize split").Role(ui.RoleSplitter).
		Absolute().Left(r.X).Top(r.Y).Size(r.Width, r.Height).Focusable().KeepFocus().FocusRing(false).Range(10, 90, float64(d.Ratio*100))
	cursor := ui.CursorResizeColumn
	if d.Axis == workspace.Vertical {
		cursor = ui.CursorResizeRow
	}
	bar.Cursor(cursor).HandleInput(func(ev ui.InputEvent) bool {
		if d.Axis == workspace.Horizontal {
			ev.X -= extra
		} else {
			ev.Y -= extra
		}
		return a.resizeInput(w, d, ev)
	})
	if bar.FocusVisible() {
		bar.Children(func() {
			if d.Axis == workspace.Horizontal {
				ui.Box(c).Absolute().Left(r.Width/2-1).Top(r.Height/2-18).Size(2, 36).Radius(1).Background(p.accent).IgnoreInput(true)
			} else {
				ui.Box(c).Absolute().Top(r.Height/2-1).Left(r.Width/2-18).Size(36, 2).Radius(1).Background(p.accent).IgnoreInput(true)
			}
		})
	}
}

func (a *shell) resizeInput(w *workspace.Workspace, d workspace.Divider, ev ui.InputEvent) bool {
	point, first, span := d.Rect.X+ev.X, d.Rect.X-d.Bounds.X, d.Bounds.Width-d.Rect.Width
	if d.Axis == workspace.Vertical {
		point, first, span = d.Rect.Y+ev.Y, d.Rect.Y-d.Bounds.Y, d.Bounds.Height-d.Rect.Height
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		if ev.Clicks == 2 {
			a.changed(w.ResetRatio(d.NodeID))
			return true
		}
		a.resize = resizeGesture{id: d.NodeID, axis: d.Axis, start: point, initial: first, span: max(span, 1)}
		return true
	case ui.InputPointerMove:
		if a.resize.id != d.NodeID {
			return false
		}
		a.report(w.SetRatio(d.NodeID, (a.resize.initial+point-a.resize.start)/a.resize.span))
		return true
	case ui.InputPointerUp:
		if a.resize.id != d.NodeID {
			return false
		}
		a.report(w.SetRatio(d.NodeID, (a.resize.initial+point-a.resize.start)/a.resize.span))
		a.resize = resizeGesture{}
		a.save()
		return true
	case ui.InputKeyDown:
		if ev.Key == ui.KeyEnter {
			a.changed(w.ResetRatio(d.NodeID))
			return true
		}
		delta := float32(0)
		if (d.Axis == workspace.Horizontal && ev.Key == ui.KeyLeft) || (d.Axis == workspace.Vertical && ev.Key == ui.KeyUp) {
			delta = -.02
		}
		if (d.Axis == workspace.Horizontal && ev.Key == ui.KeyRight) || (d.Axis == workspace.Vertical && ev.Key == ui.KeyDown) {
			delta = .02
		}
		if delta != 0 {
			a.changed(w.SetRatio(d.NodeID, d.Ratio+delta))
			return true
		}
	}
	return false
}

func (a *shell) sidebarDivider(c *ui.Context, left, width, height float32, p palette) {
	rightSide := a.settings.SidebarPosition == config.SidebarRight
	edge := left + width
	if rightSide {
		edge = left
	}
	bar := ui.Box(c.Key("sidebar-divider")).Label("Resize sidebar").Absolute().Left(edge-6).Top(62).Width(12).Height(height-92).
		Cursor(ui.CursorResizeColumn).Focusable().KeepFocus().FocusRing(false).Role(ui.RoleSplitter).Range(180, 420, float64(a.settings.SidebarWidth))
	bar.HandleInput(func(ev ui.InputEvent) bool {
		point := edge - 6 + ev.X
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			if ev.Clicks == 2 {
				a.settings.SidebarWidth = 240
				a.save()
				return true
			}
			a.resize = resizeGesture{id: "sidebar", start: point, initial: width}
			return true
		case ui.InputPointerMove, ui.InputPointerUp:
			if a.resize.id != "sidebar" {
				return false
			}
			delta := point - a.resize.start
			if rightSide {
				delta = -delta
			}
			a.report(a.settings.SetSidebarWidth(a.resize.initial + delta))
			if ev.Kind == ui.InputPointerUp {
				a.resize = resizeGesture{}
				a.save()
			}
			return true
		}
		return false
	})
	step := float32(10)
	if rightSide {
		step = -step
	}
	if bar.Shortcut(0, ui.KeyLeft) {
		a.report(a.settings.SetSidebarWidth(a.settings.SidebarWidth - step))
		a.save()
	}
	if bar.Shortcut(0, ui.KeyRight) {
		a.report(a.settings.SetSidebarWidth(a.settings.SidebarWidth + step))
		a.save()
	}
	if bar.FocusVisible() {
		bar.Children(func() {
			ui.Box(c).Absolute().Left(5).Top(height/2-80).Size(2, 36).Radius(1).Background(p.accent).IgnoreInput(true)
		})
	}
}

func (a *shell) split(groupID string, dock workspace.Dock) {
	w := a.active()
	if w == nil {
		return
	}
	_, err := w.AddPanel(groupID, dock, workspace.PanelSpec{Kind: workspace.Terminal, Title: "Terminal", Resource: w.RootPath})
	a.changed(err)
}

func (a *shell) zoom(groupID string) {
	w := a.active()
	if w == nil || w.ActiveGroup() == nil {
		return
	}
	if err := w.FocusGroup(groupID); err != nil {
		a.report(err)
		return
	}
	w.ToggleZoom()
	a.changed(nil)
	if group := w.ActiveGroup(); group != nil {
		a.requestPanelFocus(group.ActivePanelID)
	}
}

func (a *shell) movePanel(w *workspace.Workspace, panelID, targetGroupID string, dock workspace.Dock) error {
	if dock == workspace.Center {
		target := w.FindNode(targetGroupID)
		if target == nil {
			return workspace.ErrNotFound
		}
		return w.SwapPanels(panelID, target.ActivePanelID)
	}
	return w.MovePanel(panelID, targetGroupID, dock)
}

func (a *shell) panelMenu(m *ui.Menu, w *workspace.Workspace, group *workspace.Node, panel workspace.Panel) {
	if a.panels.Menu(panel.ID, m) {
		m.Separator()
	}
	if a.panels.CanRefresh(panel.ID) {
		if m.Item("Refresh").Chosen() {
			a.deferAction(func() { a.panels.Refresh(panel.ID) })
		}
		m.Separator()
	}
	if m.Item("Split right").Chosen() {
		a.deferAction(func() { a.split(group.ID, workspace.Right) })
	}
	if m.Item("Split down").Chosen() {
		a.deferAction(func() { a.split(group.ID, workspace.Bottom) })
	}
	if m.Item("Zoom / restore panel").Chosen() {
		a.deferAction(func() { a.zoom(group.ID) })
	}
	if m.Item("Equalize all splits").Chosen() {
		a.deferAction(func() { w.Equalize(); a.changed(nil) })
	}
	m.Separator()
	for _, leaf := range a.geometry.Leaves {
		if leaf.NodeID == group.ID {
			continue
		}
		target := w.FindNode(leaf.NodeID)
		if target == nil {
			continue
		}
		_, other := w.FindPanel(target.ActivePanelID)
		if other == nil {
			continue
		}
		m.Submenu(fmt.Sprintf("Move to %s", other.Title), func(m *ui.Menu) {
			for _, dock := range []workspace.Dock{workspace.Center, workspace.Left, workspace.Right, workspace.Top, workspace.Bottom} {
				label := string(dock)
				if dock == workspace.Center {
					label = "Swap positions"
				}
				if m.Item(label).Chosen() {
					a.deferAction(func() { a.changed(a.movePanel(w, panel.ID, target.ID, dock)); a.requestPanelFocus(panel.ID) })
				}
			}
		})
	}
	if len(group.Panels) > 1 {
		if m.Item("Move first in selector").Chosen() {
			a.deferAction(func() { a.changed(w.ReorderPanel(panel.ID, 0)) })
		}
		if m.Item("Move last in selector").Chosen() {
			a.deferAction(func() { a.changed(w.ReorderPanel(panel.ID, len(group.Panels)-1)) })
		}
	}
	m.Separator()
	a.addMenu(m)
	m.Separator()
	if m.Item("Close panel").Chosen() {
		a.deferAction(func() { a.changed(w.ClosePanel(panel.ID)) })
	}
}

func (a *shell) addMenu(m *ui.Menu) {
	for _, kind := range []workspace.Kind{workspace.Terminal, workspace.Files, workspace.Changes, workspace.Browser} {
		if m.Item("Open " + panels.Title(kind)).Chosen() {
			a.deferAction(func() { a.addPanel(kind) })
		}
	}
}
