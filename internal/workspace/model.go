// Package workspace owns Patch's platform-independent workspace and panel layout.
// Callers serialize mutations on their UI thread. Renderers may inspect exported
// fields, but should use methods to mutate the tree and preserve its invariants.
package workspace

import (
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const Version = 1

type Kind string

// These kinds are reserved by Patch. Other nonempty kinds may be registered by a renderer.
const (
	Terminal   Kind = "terminal"
	Files      Kind = "files"
	Changes    Kind = "changes"
	Welcome    Kind = "welcome"
	Browser    Kind = "browser"
	Primitives Kind = "primitives"
)

type Axis string

const (
	Horizontal Axis = "horizontal" // First is left; Second is right.
	Vertical   Axis = "vertical"   // First is top; Second is bottom.
)

type Dock string

const (
	Center Dock = "center"
	Left   Dock = "left"
	Right  Dock = "right"
	Top    Dock = "top"
	Bottom Dock = "bottom"
)

var (
	ErrNotFound = errors.New("workspace item not found")
	ErrSelfMove = errors.New("panel is already at the docking target")
)

type PanelSpec struct {
	Kind     Kind
	Title    string
	Resource string
}

type Panel struct {
	ID       string `json:"id"`
	Kind     Kind   `json:"kind"`
	Title    string `json:"title"`
	Resource string `json:"resource,omitempty"`
}

// Node is either a binary split (First and Second), or a leaf with ordered Panels.
// A split's Ratio is the fraction assigned to First, excluding the divider.
type Node struct {
	ID            string  `json:"id"`
	Axis          Axis    `json:"axis,omitempty"`
	Ratio         float32 `json:"ratio,omitempty"`
	First         *Node   `json:"first,omitempty"`
	Second        *Node   `json:"second,omitempty"`
	Panels        []Panel `json:"panels,omitempty"`
	ActivePanelID string  `json:"activePanelId,omitempty"`
}

func (n *Node) IsLeaf() bool { return n != nil && n.First == nil && n.Second == nil }

type Workspace struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RootPath      string `json:"rootPath"`
	Root          *Node  `json:"root"`
	ActiveGroupID string `json:"activeGroupId,omitempty"`
	ZoomedGroupID string `json:"zoomedGroupId,omitempty"`
}

type State struct {
	Version           int          `json:"version"`
	Workspaces        []*Workspace `json:"workspaces"`
	ActiveWorkspaceID string       `json:"activeWorkspaceId"`
}

// New creates one welcome workspace at root. The entrypoint should resolve root
// to the user's actual project path; the model never changes the working directory.
func New(root string) *State {
	s := &State{Version: Version}
	s.CreateWorkspace("", root)
	return s
}

func newID(prefix string) string { return prefix + "-" + rand.Text() }

func (s *State) CreateWorkspace(name, root string) *Workspace {
	root = filepath.Clean(root)
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(root)
	}
	p := Panel{ID: newID("panel"), Kind: Welcome, Title: "Welcome", Resource: root}
	group := newGroup(p)
	w := &Workspace{ID: newID("workspace"), Name: name, RootPath: root, Root: group, ActiveGroupID: group.ID}
	s.Workspaces = append(s.Workspaces, w)
	s.ActiveWorkspaceID = w.ID
	return w
}

func (s *State) FindWorkspace(id string) *Workspace {
	for _, w := range s.Workspaces {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func (s *State) ActiveWorkspace() *Workspace { return s.FindWorkspace(s.ActiveWorkspaceID) }

func (s *State) SelectWorkspace(id string) error {
	if s.FindWorkspace(id) == nil {
		return fmt.Errorf("select workspace %q: %w", id, ErrNotFound)
	}
	s.ActiveWorkspaceID = id
	return nil
}

// CloseWorkspace selects the adjacent workspace when closing the active one.
// Closing the last workspace creates a new welcome workspace at the same root.
func (s *State) CloseWorkspace(id string) error {
	for i, w := range s.Workspaces {
		if w.ID != id {
			continue
		}
		s.Workspaces = append(s.Workspaces[:i], s.Workspaces[i+1:]...)
		if len(s.Workspaces) == 0 {
			s.CreateWorkspace("", w.RootPath)
		} else if s.ActiveWorkspaceID == id {
			s.ActiveWorkspaceID = s.Workspaces[min(i, len(s.Workspaces)-1)].ID
		}
		return nil
	}
	return fmt.Errorf("close workspace %q: %w", id, ErrNotFound)
}

// ReorderWorkspace moves a workspace to its final zero-based index.
func (s *State) ReorderWorkspace(id string, index int) error {
	if index < 0 || index >= len(s.Workspaces) {
		return fmt.Errorf("workspace index %d out of range", index)
	}
	for i, w := range s.Workspaces {
		if w.ID == id {
			moveItem(s.Workspaces, i, index)
			return nil
		}
	}
	return fmt.Errorf("reorder workspace %q: %w", id, ErrNotFound)
}

func moveItem[T any](items []T, from, to int) {
	item := items[from]
	if from < to {
		copy(items[from:to], items[from+1:to+1])
	} else {
		copy(items[to+1:from+1], items[to:from])
	}
	items[to] = item
}

func newGroup(panel Panel) *Node {
	return &Node{ID: newID("group"), Panels: []Panel{panel}, ActivePanelID: panel.ID}
}

func (w *Workspace) FindNode(id string) *Node { return findNode(w.Root, id) }

func findNode(n *Node, id string) *Node {
	if n == nil || n.ID == id {
		return n
	}
	if first := findNode(n.First, id); first != nil {
		return first
	}
	return findNode(n.Second, id)
}

// FindPanel returns the containing group and panel. A panel pointer is only valid
// until the next model mutation, because tab insertion/reordering can move slices.
func (w *Workspace) FindPanel(id string) (*Node, *Panel) { return findPanel(w.Root, id) }

func findPanel(n *Node, id string) (*Node, *Panel) {
	if n == nil {
		return nil, nil
	}
	for i := range n.Panels {
		if n.Panels[i].ID == id {
			return n, &n.Panels[i]
		}
	}
	if group, panel := findPanel(n.First, id); panel != nil {
		return group, panel
	}
	return findPanel(n.Second, id)
}

func (w *Workspace) ActiveGroup() *Node { return w.FindNode(w.ActiveGroupID) }

func (w *Workspace) FocusGroup(id string) error {
	n := w.FindNode(id)
	if n == nil || !n.IsLeaf() {
		return fmt.Errorf("focus group %q: %w", id, ErrNotFound)
	}
	w.ActiveGroupID = id
	if w.ZoomedGroupID != id {
		w.RestoreZoom()
	}
	return nil
}

func (w *Workspace) SelectPanel(id string) error {
	group, panel := w.FindPanel(id)
	if panel == nil {
		return fmt.Errorf("select panel %q: %w", id, ErrNotFound)
	}
	group.ActivePanelID = id
	return w.FocusGroup(group.ID)
}

// SetPanelResource stores the resource exactly as provided. The caller owns any
// kind-specific validation, such as validating a browser navigation URL.
func (w *Workspace) SetPanelResource(panelID, resource string) error {
	_, panel := w.FindPanel(panelID)
	if panel == nil {
		return fmt.Errorf("set panel resource %q: %w", panelID, ErrNotFound)
	}
	panel.Resource = resource
	return nil
}

func firstLeaf(n *Node) *Node {
	for n != nil && !n.IsLeaf() {
		n = n.First
	}
	return n
}

func (w *Workspace) repairFocus() {
	if w.FindNode(w.ActiveGroupID) == nil {
		w.ActiveGroupID = ""
		if group := firstLeaf(w.Root); group != nil {
			w.ActiveGroupID = group.ID
		}
	}
	if w.FindNode(w.ZoomedGroupID) == nil {
		w.RestoreZoom()
	}
}
