package workspace

import (
	"math"
	"testing"
)

func TestGeometryNestedMinimumsAndSmallViewport(t *testing.T) {
	w := New("/projects/patch").ActiveWorkspace()
	left := w.Root
	right := addPanel(t, w, left.ID, Right, PanelSpec{Kind: Files})
	rightGroup, _ := w.FindPanel(right.ID)
	addPanel(t, w, rightGroup.ID, Right, PanelSpec{Kind: Terminal})
	must(t, w.SetRatio(w.Root.ID, 0.9))
	before := snapshot(t, w)
	large := w.Layout(Rect{X: 20, Y: 30, Width: 1000, Height: 600}, 10, 120)
	if len(large.Leaves) != 3 || len(large.Dividers) != 2 {
		t.Fatal("unexpected geometry counts")
	}
	// First subtree gets at most 740px so the two nested leaves retain 120px each.
	if large.Leaves[0].Rect.Width != 740 || large.Leaves[1].Rect.Width != 120 || large.Leaves[2].Rect.Width != 120 {
		t.Fatalf("nested minimums were not respected: %+v", large.Leaves)
	}
	if large.Dividers[0].Rect != (Rect{X: 760, Y: 30, Width: 10, Height: 600}) {
		t.Fatalf("incorrect divider geometry: %+v", large.Dividers[0])
	}
	for _, width := range []float32{0, 4, 100, 1000} {
		bounds := Rect{X: 20, Y: 30, Width: width, Height: 80}
		geometry := w.Layout(bounds, 10, 120)
		for _, leaf := range geometry.Leaves {
			assertBounded(t, leaf.Rect, bounds)
		}
		for _, divider := range geometry.Dividers {
			assertBounded(t, divider.Rect, bounds)
		}
	}
	if snapshot(t, w) != before {
		t.Fatal("geometry changed persisted ratios")
	}
}

func TestGeometryVerticalSplit(t *testing.T) {
	w := New("/projects/patch").ActiveWorkspace()
	addPanel(t, w, w.Root.ID, Bottom, PanelSpec{Kind: Terminal})
	must(t, w.SetRatio(w.Root.ID, 0.25))
	g := w.Layout(Rect{X: 5, Y: 10, Width: 400, Height: 804}, 4, 100)
	if g.Leaves[0].Rect != (Rect{X: 5, Y: 10, Width: 400, Height: 200}) ||
		g.Leaves[1].Rect != (Rect{X: 5, Y: 214, Width: 400, Height: 600}) ||
		g.Dividers[0].Axis != Vertical || g.Dividers[0].Ratio != 0.25 {
		t.Fatalf("incorrect vertical geometry: %+v", g)
	}
}

func assertBounded(t *testing.T, rect, bounds Rect) {
	t.Helper()
	const epsilon = 0.001
	if !finite(rect.Width) || !finite(rect.Height) || rect.Width < 0 || rect.Height < 0 ||
		rect.X < bounds.X-epsilon || rect.Y < bounds.Y-epsilon ||
		rect.X+rect.Width > bounds.X+bounds.Width+epsilon || rect.Y+rect.Height > bounds.Y+bounds.Height+epsilon {
		t.Fatalf("rectangle %+v escapes %+v", rect, bounds)
	}
}

// BenchmarkComputeLayout measures only model geometry for 16 panes, not rendering
// or native frame time. Setup and identifier generation are outside the measurement.
func BenchmarkComputeLayout(b *testing.B) {
	w := New("/projects/patch").ActiveWorkspace()
	for i := 1; i < 16; i++ {
		dock := Right
		if i%2 == 0 {
			dock = Bottom
		}
		if _, err := w.AddPanel("", dock, PanelSpec{Kind: Terminal}); err != nil {
			b.Fatal(err)
		}
	}
	bounds := Rect{Width: 1600, Height: 1000}
	if len(w.Layout(bounds, 6, 80).Leaves) != 16 {
		b.Fatal("fixture must have 16 leaves")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := w.Layout(bounds, 6, 80)
		if len(result.Leaves) != 16 || math.IsNaN(float64(result.Leaves[0].Rect.Width)) {
			b.Fatal("invalid layout")
		}
	}
}
