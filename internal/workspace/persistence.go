package workspace

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/darylcecile/patch/internal/config/storage"
)

// Load returns errors for missing, corrupt, or unsupported session files. It never
// creates defaults or writes the path; the entrypoint decides how to present recovery.
func Load(path string) (*State, error) {
	var state State
	if err := storage.ReadJSON(path, &state); err != nil {
		return nil, fmt.Errorf("load workspace session: %w", err)
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("load workspace session %q: %w", path, err)
	}
	return &state, nil
}

// Save validates before publishing a same-directory temporary file. Existing
// corrupt/unsupported sessions must be explicitly backed up or removed before saving;
// accidentally saving defaults after a failed Load cannot destroy the original.
func (s *State) Save(path string) error {
	if err := s.Validate(); err != nil {
		return fmt.Errorf("save workspace session: %w", err)
	}
	if _, err := Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("preserve existing workspace session: %w", err)
	}
	if err := storage.WriteJSON(path, s); err != nil {
		return fmt.Errorf("save workspace session: %w", err)
	}
	return nil
}

// Validate checks structural invariants, all references, and global ID uniqueness.
// Unknown nonempty panel kinds are retained for extensible renderer registries.
func (s *State) Validate() error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if s.Version != Version {
		return fmt.Errorf("unsupported session version %d (expected %d)", s.Version, Version)
	}
	if len(s.Workspaces) == 0 {
		return fmt.Errorf("session must contain a workspace")
	}
	seen := make(map[string]bool)
	activeFound := false
	for _, w := range s.Workspaces {
		if w == nil {
			return fmt.Errorf("workspace is null")
		}
		if err := validateWorkspace(w, seen); err != nil {
			return fmt.Errorf("workspace %q: %w", w.ID, err)
		}
		activeFound = activeFound || w.ID == s.ActiveWorkspaceID
	}
	if !activeFound {
		return fmt.Errorf("active workspace %q does not exist", s.ActiveWorkspaceID)
	}
	return nil
}

func claimID(id string, seen map[string]bool) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("ID must not be empty")
	}
	if seen[id] {
		return fmt.Errorf("duplicate ID %q", id)
	}
	seen[id] = true
	return nil
}

func validateWorkspace(w *Workspace, seen map[string]bool) error {
	if err := claimID(w.ID, seen); err != nil {
		return err
	}
	if strings.TrimSpace(w.Name) == "" || strings.TrimSpace(w.RootPath) == "" {
		return fmt.Errorf("name and rootPath must not be empty")
	}
	if err := validateNode(w.Root, seen); err != nil {
		return err
	}
	if w.Root == nil {
		if w.ActiveGroupID != "" || w.ZoomedGroupID != "" {
			return fmt.Errorf("empty workspace cannot reference a group")
		}
		return nil
	}
	group := w.FindNode(w.ActiveGroupID)
	if group == nil || !group.IsLeaf() {
		return fmt.Errorf("active group %q is not a leaf", w.ActiveGroupID)
	}
	if w.ZoomedGroupID != "" && w.ZoomedGroupID != w.ActiveGroupID {
		return fmt.Errorf("zoomed group must be the active group")
	}
	return nil
}

func validateNode(n *Node, seen map[string]bool) error {
	if n == nil {
		return nil
	}
	if err := claimID(n.ID, seen); err != nil {
		return err
	}
	if n.IsLeaf() {
		return validateGroup(n, seen)
	}
	if n.First == nil || n.Second == nil {
		return fmt.Errorf("split %q must have two children", n.ID)
	}
	if n.Axis != Horizontal && n.Axis != Vertical {
		return fmt.Errorf("split %q has invalid axis %q", n.ID, n.Axis)
	}
	if !finite(n.Ratio) || n.Ratio < MinRatio || n.Ratio > MaxRatio {
		return fmt.Errorf("split %q ratio must be within [%g, %g]", n.ID, MinRatio, MaxRatio)
	}
	if len(n.Panels) != 0 || n.ActivePanelID != "" {
		return fmt.Errorf("split %q cannot contain panel tabs", n.ID)
	}
	if err := validateNode(n.First, seen); err != nil {
		return err
	}
	return validateNode(n.Second, seen)
}

func validateGroup(n *Node, seen map[string]bool) error {
	if n.Axis != "" || n.Ratio != 0 {
		return fmt.Errorf("group %q cannot have split properties", n.ID)
	}
	if len(n.Panels) == 0 {
		return fmt.Errorf("group %q must contain a panel", n.ID)
	}
	activeFound := false
	for _, p := range n.Panels {
		if err := claimID(p.ID, seen); err != nil {
			return err
		}
		if strings.TrimSpace(string(p.Kind)) == "" || strings.TrimSpace(p.Title) == "" {
			return fmt.Errorf("panel %q must have a kind and title", p.ID)
		}
		activeFound = activeFound || p.ID == n.ActivePanelID
	}
	if !activeFound {
		return fmt.Errorf("active panel %q does not belong to group %q", n.ActivePanelID, n.ID)
	}
	return nil
}
