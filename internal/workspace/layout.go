package workspace

import (
	"fmt"
	"math"
	"strings"
)

const (
	MinRatio float32 = 0.1
	MaxRatio float32 = 0.9
)

func validDock(d Dock) bool {
	return d == Center || d == Left || d == Right || d == Top || d == Bottom
}

// AddPanel adds a tab at Center, or a new group on an edge. An empty target means
// the active group; in an empty workspace any valid dock creates the root group.
func (w *Workspace) AddPanel(targetGroupID string, dock Dock, spec PanelSpec) (*Panel, error) {
	if strings.TrimSpace(string(spec.Kind)) == "" {
		return nil, fmt.Errorf("panel kind must not be empty")
	}
	target, err := w.dockTarget(targetGroupID, dock)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(spec.Title) == "" {
		spec.Title = string(spec.Kind)
	}
	p := Panel{ID: newID("panel"), Kind: spec.Kind, Title: spec.Title, Resource: spec.Resource}
	w.dockPanel(target, dock, p)
	_, result := w.FindPanel(p.ID)
	return result, nil
}

func (w *Workspace) dockTarget(id string, dock Dock) (*Node, error) {
	if !validDock(dock) {
		return nil, fmt.Errorf("invalid docking position %q", dock)
	}
	if w.Root == nil && id == "" {
		return nil, nil
	}
	if id == "" {
		id = w.ActiveGroupID
	}
	target := w.FindNode(id)
	if target == nil || !target.IsLeaf() {
		return nil, fmt.Errorf("dock target %q: %w", id, ErrNotFound)
	}
	return target, nil
}

// MovePanel preserves the panel descriptor and ID. Center appends to the target's
// tabs. Moving to an edge of the same group splits a tab out when other tabs remain;
// moving the only tab onto itself, or centering onto its own group, returns ErrSelfMove.
// All input validation occurs before the source is detached.
func (w *Workspace) MovePanel(panelID, targetGroupID string, dock Dock) error {
	source, panel := w.FindPanel(panelID)
	if panel == nil {
		return fmt.Errorf("move panel %q: %w", panelID, ErrNotFound)
	}
	target, err := w.dockTarget(targetGroupID, dock)
	if err != nil {
		return err
	}
	if source == target && (dock == Center || len(source.Panels) == 1) {
		return ErrSelfMove
	}
	moved := *panel
	removePanel(source, panelID)
	w.Root = collapse(w.Root)
	w.dockPanel(target, dock, moved)
	return nil
}

// SwapPanels exchanges complete panel descriptors at their existing positions,
// preserving panel identities/resources and the layout tree. Focus follows the
// source panel to its destination. Within one group, it exchanges tab positions.
func (w *Workspace) SwapPanels(sourcePanelID, targetPanelID string) error {
	sourceGroup, source := w.FindPanel(sourcePanelID)
	if source == nil {
		return fmt.Errorf("swap source panel %q: %w", sourcePanelID, ErrNotFound)
	}
	targetGroup, target := w.FindPanel(targetPanelID)
	if target == nil {
		return fmt.Errorf("swap target panel %q: %w", targetPanelID, ErrNotFound)
	}
	if sourcePanelID == targetPanelID {
		return ErrSelfMove
	}
	*source, *target = *target, *source
	if sourceGroup != targetGroup && sourceGroup.ActivePanelID == sourcePanelID {
		sourceGroup.ActivePanelID = targetPanelID
	}
	targetGroup.ActivePanelID = sourcePanelID
	w.ActiveGroupID = targetGroup.ID
	if w.ZoomedGroupID != targetGroup.ID {
		w.RestoreZoom()
	}
	return nil
}

func (w *Workspace) dockPanel(target *Node, dock Dock, panel Panel) {
	group := target
	if target == nil {
		group = newGroup(panel)
		w.Root = group
	} else if dock == Center {
		target.Panels = append(target.Panels, panel)
		target.ActivePanelID = panel.ID
	} else {
		group = newGroup(panel)
		branch := &Node{ID: newID("split"), Axis: Horizontal, Ratio: 0.5, First: target, Second: group}
		if dock == Top || dock == Bottom {
			branch.Axis = Vertical
		}
		if dock == Left || dock == Top {
			branch.First, branch.Second = group, target
		}
		w.Root = replaceNode(w.Root, target.ID, branch)
	}
	w.ActiveGroupID = group.ID
	if w.ZoomedGroupID != group.ID {
		w.RestoreZoom()
	}
}

func replaceNode(n *Node, id string, replacement *Node) *Node {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return replacement
	}
	if !n.IsLeaf() {
		n.First = replaceNode(n.First, id, replacement)
		n.Second = replaceNode(n.Second, id, replacement)
	}
	return n
}

// ReorderPanel moves a panel to its final zero-based index inside its group.
func (w *Workspace) ReorderPanel(panelID string, index int) error {
	group, panel := w.FindPanel(panelID)
	if panel == nil {
		return fmt.Errorf("reorder panel %q: %w", panelID, ErrNotFound)
	}
	if index < 0 || index >= len(group.Panels) {
		return fmt.Errorf("panel index %d out of range", index)
	}
	for i := range group.Panels {
		if group.Panels[i].ID == panelID {
			moveItem(group.Panels, i, index)
			break
		}
	}
	return nil
}

// ClosePanel collapses empty groups and their parent split. Closing the focused
// group selects the first leaf of its sibling subtree. Closing the final panel
// leaves Root nil; AddPanel can then repopulate the empty workspace.
func (w *Workspace) ClosePanel(id string) error {
	group, panel := w.FindPanel(id)
	if panel == nil {
		return fmt.Errorf("close panel %q: %w", id, ErrNotFound)
	}
	removePanel(group, id)
	if group.ID == w.ActiveGroupID && len(group.Panels) == 0 {
		if neighbor := siblingLeaf(w.Root, group); neighbor != nil {
			w.ActiveGroupID = neighbor.ID
		}
	}
	w.Root = collapse(w.Root)
	w.repairFocus()
	return nil
}

func siblingLeaf(n, group *Node) *Node {
	if n == nil || n.IsLeaf() {
		return nil
	}
	if n.First == group {
		return firstLeaf(n.Second)
	}
	if n.Second == group {
		return firstLeaf(n.First)
	}
	if neighbor := siblingLeaf(n.First, group); neighbor != nil {
		return neighbor
	}
	return siblingLeaf(n.Second, group)
}

func removePanel(group *Node, id string) {
	for i, panel := range group.Panels {
		if panel.ID != id {
			continue
		}
		group.Panels = append(group.Panels[:i], group.Panels[i+1:]...)
		if len(group.Panels) == 0 {
			group.ActivePanelID = ""
		} else if group.ActivePanelID == id {
			group.ActivePanelID = group.Panels[min(i, len(group.Panels)-1)].ID
		}
		return
	}
}

func collapse(n *Node) *Node {
	if n == nil {
		return nil
	}
	if n.IsLeaf() {
		if len(n.Panels) == 0 {
			return nil
		}
		return n
	}
	n.First, n.Second = collapse(n.First), collapse(n.Second)
	if n.First == nil {
		return n.Second
	}
	if n.Second == nil {
		return n.First
	}
	return n
}

func finite(value float32) bool { return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0) }

func (w *Workspace) SetRatio(nodeID string, ratio float32) error {
	n := w.FindNode(nodeID)
	if n == nil || n.IsLeaf() {
		return fmt.Errorf("resize split %q: %w", nodeID, ErrNotFound)
	}
	if !finite(ratio) {
		return fmt.Errorf("split ratio must be finite")
	}
	n.Ratio = max(MinRatio, min(MaxRatio, ratio))
	return nil
}

func (w *Workspace) ResetRatio(nodeID string) error { return w.SetRatio(nodeID, 0.5) }

// Equalize resets each binary split to equal halves; it does not flatten the tree.
func (w *Workspace) Equalize() { equalize(w.Root) }

func equalize(n *Node) {
	if n == nil || n.IsLeaf() {
		return
	}
	n.Ratio = 0.5
	equalize(n.First)
	equalize(n.Second)
}

func (w *Workspace) ToggleZoom() {
	if w.ZoomedGroupID == w.ActiveGroupID {
		w.RestoreZoom()
	} else if group := w.ActiveGroup(); group != nil {
		w.ZoomedGroupID = group.ID
	}
}

func (w *Workspace) RestoreZoom() { w.ZoomedGroupID = "" }
