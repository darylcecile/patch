package app

import (
	"math"
	"slices"
	"testing"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/panels"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
)

func testShell(t *testing.T) (*shell, *ui.Tester, string, string) {
	t.Helper()
	state := workspace.New("/project")
	w := state.ActiveWorkspace()
	first := w.Root.ActivePanelID
	w.Root.Panels[0].Title = "First"
	second, err := w.AddPanel(w.ActiveGroupID, workspace.Right, workspace.PanelSpec{Kind: workspace.Welcome, Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	a := &shell{state: state, settings: config.Default()}
	a.panels = panels.NewRegistry(panels.Host{})
	a.panels.Sync(state)
	return a, ui.NewTester(a.view, 1200, 800), first, second.ID
}

func TestHeaderDragPreviewCancelAndSwap(t *testing.T) {
	a, tt, first, second := testShell(t)
	w := a.active()
	a.changed(w.SelectPanel(first))
	tt.Frame()
	sourceGroup, _ := w.FindPanel(first)
	sourceGroupID := sourceGroup.ID
	grip, ok := tt.Find("Drag First")
	if !ok {
		t.Fatal("first panel header not found")
	}
	targetGroup, _ := w.FindPanel(second)
	var target workspace.Rect
	for _, leaf := range a.geometry.Leaves {
		if leaf.NodeID == targetGroup.ID {
			target = leaf.Rect
		}
	}
	x, y := target.X+target.Width/2, target.Y+target.Height/2
	startX, startY := grip.X+grip.W/2, grip.Y+grip.H/2
	tt.Press(startX, startY)
	tt.Move(x, y)
	if a.dropPreview == nil || a.dropPreview.dock != workspace.Center {
		t.Fatal("center swap preview not visible during drag")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Release(x, y)
	if group, _ := w.FindPanel(first); group.ID != sourceGroupID {
		t.Fatal("Escape committed the pending move")
	}
	tt.Press(startX, startY)
	tt.Move(x, y)
	tt.Release(x, y)
	if group, _ := w.FindPanel(first); group.ID != targetGroup.ID {
		t.Fatal("release did not swap the source panel into the target")
	}
	if group, _ := w.FindPanel(second); group.ID != sourceGroupID {
		t.Fatal("swap did not preserve the other panel's identity")
	}
	if a.focusedPanel != first || a.focusPending {
		t.Fatal("active panel did not regain content focus after the drag")
	}
	if err := a.state.Validate(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50 && !tt.Focused("Drag First"); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Drag First") {
		t.Fatal("panel header is not reachable with the keyboard")
	}
	tt.Key(ui.Shift, ui.KeyF10)
	if !slices.Contains(tt.Menu(), "Split right") {
		t.Fatal("keyboard context menu did not expose host panel actions")
	}
	tt.CloseMenu()
}

func TestDividerTracksPointerAndSettingsAdapt(t *testing.T) {
	a, tt, _, _ := testShell(t)
	d := a.geometry.Dividers[0]
	x, y := d.Rect.X+d.Rect.Width/2, d.Rect.Y+d.Rect.Height/2
	tt.Press(x, y)
	tt.Move(x+70, y)
	want := d.Ratio + 70/(d.Bounds.Width-d.Rect.Width)
	if diff := math.Abs(float64(a.active().Root.Ratio - want)); diff > .002 {
		t.Fatalf("resize ratio %v, want %v", a.active().Root.Ratio, want)
	}
	tt.Release(x+70, y)
	for i := 0; i < 40 && !tt.Focused("Resize split"); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Resize split") {
		t.Fatal("divider is not reachable with the keyboard")
	}
	tt.Key(0, ui.KeyEnter)
	if a.active().Root.Ratio != .5 {
		t.Fatal("focused divider Enter did not reset the split")
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Tab style") || !tt.HasText("Appearance") {
		t.Fatal("settings controls are not visible")
	}
	if err := tt.Click("Close settings"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Toggle sidebar"); err != nil {
		t.Fatal(err)
	}
	if a.settings.Navigation != config.Sidebar || a.settings.SidebarVisible {
		t.Fatal("collapsing the sidebar changed its navigation mode")
	}
	tt.SetSize(680, 500)
	if err := tt.Click("Toggle sidebar"); err != nil {
		t.Fatal(err)
	}
	if a.settings.Navigation != config.Sidebar || !a.sidebarShown() {
		t.Fatal("narrow sidebar did not expand in the selected mode")
	}
	if _, ok := tt.Find("Panel actions"); !ok {
		t.Fatal("narrow panel has no overflow actions")
	}
}

func TestPlacementVisibilityAndRightSidebar(t *testing.T) {
	a, tt, _, _ := testShell(t)
	a.toggleSidebar()
	a.setTabStyle(config.Tabs, config.SidebarLeft)
	a.toggleSidebar()
	if a.settings.SidebarVisible || a.settings.Navigation != config.Tabs {
		t.Fatal("Above mode toggle changed persisted sidebar preferences")
	}
	a.setTabStyle(config.Sidebar, config.SidebarRight)
	if a.settings.SidebarVisible || a.settings.SidebarPosition != config.SidebarRight {
		t.Fatal("placement changed collapsed visibility")
	}
	a.toggleSidebar()
	tt.Frame()
	divider, ok := tt.Find("Resize sidebar")
	if !ok || divider.X <= a.canvasBounds.X+a.canvasBounds.Width {
		t.Fatal("right sidebar divider is not beside the canvas")
	}
	before := a.settings.SidebarWidth
	x, y := divider.X+divider.W/2, divider.Y+divider.H/2
	tt.Press(x, y)
	tt.Move(x-20, y)
	tt.Release(x-20, y)
	if math.Abs(float64(a.settings.SidebarWidth-before-20)) > .01 {
		t.Fatal("right-edge drag did not track the pointer")
	}
	tt.SetSize(680, 500)
	if a.settings.Navigation != config.Sidebar || a.settings.SidebarPosition != config.SidebarRight || !a.sidebarShown() {
		t.Fatal("viewport resize changed selected placement")
	}
	a.setTabStyle(config.Tabs, config.SidebarLeft)
	if a.settings.SidebarPosition != config.SidebarRight || !a.settings.SidebarVisible {
		t.Fatal("Above mode discarded sidebar side or visibility")
	}
}
