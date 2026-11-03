package workspace

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
)

func addPanel(t *testing.T, w *Workspace, target string, dock Dock, spec PanelSpec) Panel {
	t.Helper()
	p, err := w.AddPanel(target, dock, spec)
	if err != nil {
		t.Fatal(err)
	}
	return *p
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	must(t, err)
	return string(data)
}

func TestDockEdgesPreserveTargetAndResources(t *testing.T) {
	for _, dock := range []Dock{Left, Right, Top, Bottom} {
		t.Run(string(dock), func(t *testing.T) {
			s := New("/projects/patch")
			w := s.ActiveWorkspace()
			target := w.Root
			p := addPanel(t, w, target.ID, dock, PanelSpec{Kind: Browser, Title: "Go docs", Resource: "https://go.dev/doc/?tab=1"})
			group, found := w.FindPanel(p.ID)
			if *found != p || w.FindNode(target.ID) != target || group.ID != w.ActiveGroupID {
				t.Fatal("docking lost the original group, descriptor, or focus")
			}
			first, second, axis := target, group, Horizontal
			if dock == Left || dock == Top {
				first, second = group, target
			}
			if dock == Top || dock == Bottom {
				axis = Vertical
			}
			if w.Root.First != first || w.Root.Second != second || w.Root.Axis != axis || w.Root.Ratio != 0.5 {
				t.Fatalf("unexpected split orientation: %+v", w.Root)
			}
			must(t, s.Validate())
		})
	}
}

func TestMoveMergeCollapseAndSplitTab(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	originalGroup := w.Root
	welcome := originalGroup.Panels[0]
	files := addPanel(t, w, originalGroup.ID, Right, PanelSpec{Kind: Files, Title: "Files", Resource: "/projects/patch"})
	filesGroup, _ := w.FindPanel(files.ID)
	terminal := addPanel(t, w, filesGroup.ID, Bottom, PanelSpec{Kind: Terminal, Title: "Shell", Resource: "/projects/patch/internal"})
	terminalGroup, _ := w.FindPanel(terminal.ID)
	survivingSplitID := w.Root.Second.ID
	removedSplitID := w.Root.ID
	must(t, w.MovePanel(welcome.ID, terminalGroup.ID, Center))
	if w.Root.ID != survivingSplitID || w.FindNode(removedSplitID) != nil || w.FindNode(originalGroup.ID) != nil {
		t.Fatal("empty source and its split were not collapsed to the surviving subtree")
	}
	if !reflect.DeepEqual(terminalGroup.Panels, []Panel{terminal, welcome}) || terminalGroup.ActivePanelID != welcome.ID {
		t.Fatal("center docking did not append and select the unchanged descriptor")
	}
	// Splitting a tab out of its own multi-tab group is a real rearrangement.
	must(t, w.MovePanel(terminal.ID, terminalGroup.ID, Left))
	movedGroup, moved := w.FindPanel(terminal.ID)
	if *moved != terminal || movedGroup.ID == terminalGroup.ID || terminalGroup.Panels[0] != welcome {
		t.Fatal("splitting a tab changed its identity or the remaining group")
	}
	// Moving a sole panel to its sibling collapses the old split before inserting.
	must(t, w.MovePanel(files.ID, movedGroup.ID, Top))
	if w.FindNode(filesGroup.ID) != nil {
		t.Fatal("empty files group survived move")
	}
	_, movedFiles := w.FindPanel(files.ID)
	if *movedFiles != files {
		t.Fatal("resource changed during move")
	}
	must(t, s.Validate())
}

func TestRejectedMovesPreserveState(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	p := w.Root.Panels[0]
	before := snapshot(t, s)
	cases := []struct {
		panel, target string
		dock          Dock
		want          error
	}{
		{p.ID, w.Root.ID, Center, ErrSelfMove},
		{p.ID, w.Root.ID, Left, ErrSelfMove},
		{p.ID, "missing", Center, ErrNotFound},
		{"missing", w.Root.ID, Right, ErrNotFound},
		{p.ID, w.Root.ID, Dock("diagonal"), nil},
	}
	for _, c := range cases {
		err := w.MovePanel(c.panel, c.target, c.dock)
		if err == nil || (c.want != nil && !errors.Is(err, c.want)) {
			t.Fatalf("move %+v: got %v", c, err)
		}
		if snapshot(t, s) != before {
			t.Fatal("rejected move mutated the session")
		}
	}
}

func TestSwapPanelsPreservesDescriptorsAndFocus(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	left := w.Root
	welcome := left.Panels[0]
	source := addPanel(t, w, left.ID, Center, PanelSpec{Kind: Terminal, Title: "Shell", Resource: "/projects/patch/internal"})
	target := addPanel(t, w, left.ID, Right, PanelSpec{Kind: Browser, Title: "Docs", Resource: "https://go.dev/doc/"})
	right, _ := w.FindPanel(target.ID)
	root := w.Root
	must(t, w.SelectPanel(source.ID))
	w.ToggleZoom()
	before := snapshot(t, s)
	for _, invalid := range []struct {
		source, target string
		want           error
	}{
		{"missing", target.ID, ErrNotFound},
		{source.ID, "missing", ErrNotFound},
		{source.ID, source.ID, ErrSelfMove},
	} {
		if err := w.SwapPanels(invalid.source, invalid.target); !errors.Is(err, invalid.want) {
			t.Fatalf("swap %+v: got %v", invalid, err)
		}
		if snapshot(t, s) != before {
			t.Fatal("rejected swap changed the session")
		}
	}
	must(t, w.SwapPanels(source.ID, target.ID))
	if !reflect.DeepEqual(left.Panels, []Panel{welcome, target}) || !reflect.DeepEqual(right.Panels, []Panel{source}) {
		t.Fatal("swap changed panel identities/resources or existing group contents")
	}
	if w.Root != root || root.First != left || root.Second != right ||
		left.ActivePanelID != target.ID || right.ActivePanelID != source.ID || w.ActiveGroupID != right.ID || w.ZoomedGroupID != "" {
		t.Fatal("swap changed the tree or did not follow the source panel to its destination")
	}
	must(t, w.SelectPanel(welcome.ID))
	w.ToggleZoom()
	must(t, w.SwapPanels(welcome.ID, target.ID))
	if !reflect.DeepEqual(left.Panels, []Panel{target, welcome}) ||
		left.ActivePanelID != welcome.ID || w.ActiveGroupID != left.ID || w.ZoomedGroupID != left.ID {
		t.Fatal("same-group swap did not preserve zoom and select the moved source")
	}
	must(t, s.Validate())
}

func TestTabReorderCloseAndEmptyRecovery(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	group := w.Root
	welcome := group.Panels[0]
	files := addPanel(t, w, group.ID, Center, PanelSpec{Kind: Files, Resource: w.RootPath})
	terminal := addPanel(t, w, group.ID, Center, PanelSpec{Kind: Terminal, Resource: w.RootPath})
	must(t, w.ReorderPanel(welcome.ID, 2))
	if !reflect.DeepEqual(group.Panels, []Panel{files, terminal, welcome}) || group.ActivePanelID != terminal.ID {
		t.Fatal("reordering did not preserve tab order/selection")
	}
	must(t, w.ClosePanel(terminal.ID))
	if group.ActivePanelID != welcome.ID {
		t.Fatal("closing active tab did not select the adjacent tab")
	}
	w.ToggleZoom()
	must(t, w.ClosePanel(files.ID))
	must(t, w.ClosePanel(welcome.ID))
	if w.Root != nil || w.ActiveGroupID != "" || w.ZoomedGroupID != "" {
		t.Fatal("last close did not produce a valid empty workspace")
	}
	must(t, s.Validate())
	reopened := addPanel(t, w, "", Center, PanelSpec{Kind: Files, Resource: w.RootPath})
	if w.Root.Panels[0] != reopened || w.Root.ID != w.ActiveGroupID {
		t.Fatal("cannot repopulate empty workspace")
	}
	must(t, s.Validate())
}

func TestWorkspaceOrderSelectionAndLastClose(t *testing.T) {
	s := New("/projects/first")
	first := s.ActiveWorkspace()
	second := s.CreateWorkspace("Second", "/projects/second")
	third := s.CreateWorkspace("Third", "/projects/third")
	must(t, s.ReorderWorkspace(first.ID, 2))
	if !reflect.DeepEqual(s.Workspaces, []*Workspace{second, third, first}) || s.ActiveWorkspaceID != third.ID {
		t.Fatal("workspace reorder changed selection or order")
	}
	must(t, s.SelectWorkspace(second.ID))
	must(t, s.CloseWorkspace(second.ID))
	if s.ActiveWorkspaceID != third.ID {
		t.Fatal("closing active workspace did not select its neighbour")
	}
	must(t, s.CloseWorkspace(first.ID))
	must(t, s.CloseWorkspace(third.ID))
	fresh := s.ActiveWorkspace()
	if len(s.Workspaces) != 1 || fresh.ID == third.ID || fresh.RootPath != third.RootPath || fresh.Root.Panels[0].Kind != Welcome {
		t.Fatal("last workspace close did not create usable welcome at same project root")
	}
	must(t, s.Validate())
}

func TestCloseFocusedGroupSelectsNearestSibling(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	welcome := w.Root.Panels[0]
	files := addPanel(t, w, w.Root.ID, Right, PanelSpec{Kind: Files})
	filesGroup, _ := w.FindPanel(files.ID)
	terminal := addPanel(t, w, filesGroup.ID, Bottom, PanelSpec{Kind: Terminal})
	must(t, w.ClosePanel(terminal.ID))
	if w.ActiveGroupID != filesGroup.ID || w.Root.Second != filesGroup {
		t.Fatal("closing focused terminal did not focus its surviving files neighbour")
	}
	must(t, w.ClosePanel(welcome.ID))
	if w.ActiveGroupID != filesGroup.ID || w.Root != filesGroup {
		t.Fatal("closing an inactive group changed focus")
	}
	must(t, s.Validate())
}

func TestRatioZoomAndCollapse(t *testing.T) {
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	left := w.Root
	rightPanel := addPanel(t, w, left.ID, Right, PanelSpec{Kind: Terminal})
	right, _ := w.FindPanel(rightPanel.ID)
	root := w.Root
	must(t, w.SetRatio(root.ID, -2))
	if root.Ratio != MinRatio {
		t.Fatal("minimum ratio not clamped")
	}
	must(t, w.SetRatio(root.ID, 2))
	if root.Ratio != MaxRatio {
		t.Fatal("maximum ratio not clamped")
	}
	if err := w.SetRatio(root.ID, float32(math.NaN())); err == nil || root.Ratio != MaxRatio {
		t.Fatal("nonfinite ratio was not rejected without mutation")
	}
	must(t, w.ResetRatio(root.ID))
	if root.Ratio != 0.5 {
		t.Fatal("ratio reset failed")
	}
	must(t, w.SetRatio(root.ID, 0.7))
	w.ToggleZoom()
	bounds := Rect{Width: 800, Height: 600}
	zoom := w.Layout(bounds, 6, 100)
	if len(zoom.Leaves) != 1 || zoom.Leaves[0].NodeID != right.ID || zoom.Leaves[0].Rect != bounds || len(zoom.Dividers) != 0 {
		t.Fatal("zoom layout did not fill bounds with active group")
	}
	zoomTab := addPanel(t, w, right.ID, Center, PanelSpec{Kind: Files})
	if w.ZoomedGroupID != right.ID || right.ActivePanelID != zoomTab.ID || len(w.Layout(bounds, 6, 100).Leaves) != 1 {
		t.Fatal("adding a center tab did not preserve the active group's zoom")
	}
	must(t, w.ClosePanel(zoomTab.ID))
	w.ToggleZoom()
	if w.Root != root || root.Ratio != 0.7 || len(w.Layout(bounds, 6, 100).Leaves) != 2 {
		t.Fatal("zoom restoration changed underlying layout")
	}
	w.Equalize()
	if root.Ratio != 0.5 {
		t.Fatal("equalize failed")
	}
	w.ToggleZoom()
	must(t, w.ClosePanel(rightPanel.ID))
	if w.Root != left || w.ActiveGroupID != left.ID || w.ZoomedGroupID != "" {
		t.Fatal("closing zoomed leaf did not collapse/focus surviving group")
	}
	must(t, s.Validate())
}
