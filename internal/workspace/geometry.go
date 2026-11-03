package workspace

type Rect struct {
	X, Y, Width, Height float32
}

type LeafRect struct {
	NodeID string
	Rect   Rect
}

type Divider struct {
	NodeID string
	Axis   Axis
	Rect   Rect
	Bounds Rect    // Full parent bounds, used to turn pointer coordinates into a ratio.
	Ratio  float32 // Effective ratio after geometry's minimum-size constraints.
}

type Geometry struct {
	Leaves   []LeafRect
	Dividers []Divider
}

type minimum struct{ width, height float32 }

// ComputeLayout is pure. minPaneSize applies to each leaf in both dimensions,
// including nested splits. If the viewport cannot fit all minimums, space is shared
// proportionally to subtree minimums; rectangles remain bounded and nonnegative.
// Invalid/nonfinite bounds return empty geometry. The model ratio is never changed.
func ComputeLayout(root *Node, bounds Rect, dividerSize, minPaneSize float32) Geometry {
	var result Geometry
	if root == nil || !finite(bounds.X) || !finite(bounds.Y) || !finite(bounds.Width) || !finite(bounds.Height) {
		return result
	}
	bounds.Width, bounds.Height = max(0, bounds.Width), max(0, bounds.Height)
	dividerSize, minPaneSize = positive(dividerSize), positive(minPaneSize)
	minimums := make(map[*Node]minimum)
	measure(root, dividerSize, minPaneSize, minimums)
	arrange(root, bounds, dividerSize, minimums, &result)
	return result
}

func positive(value float32) float32 {
	if !finite(value) {
		return 0
	}
	return max(0, value)
}

func measure(n *Node, gap, pane float32, sizes map[*Node]minimum) minimum {
	if n == nil {
		return minimum{}
	}
	size := minimum{pane, pane}
	if !n.IsLeaf() {
		a, b := measure(n.First, gap, pane, sizes), measure(n.Second, gap, pane, sizes)
		if n.Axis == Horizontal {
			size = minimum{a.width + gap + b.width, max(a.height, b.height)}
		} else {
			size = minimum{max(a.width, b.width), a.height + gap + b.height}
		}
	}
	sizes[n] = size
	return size
}

func arrange(n *Node, bounds Rect, gap float32, sizes map[*Node]minimum, result *Geometry) {
	if n == nil {
		return
	}
	if n.IsLeaf() {
		result.Leaves = append(result.Leaves, LeafRect{NodeID: n.ID, Rect: bounds})
		return
	}
	a, b := sizes[n.First], sizes[n.Second]
	extent, minFirst, minSecond := bounds.Width, a.width, b.width
	if n.Axis == Vertical {
		extent, minFirst, minSecond = bounds.Height, a.height, b.height
	}
	effectiveGap := min(gap, extent)
	available := extent - effectiveGap
	first := splitExtent(available, minFirst, minSecond, n.Ratio)
	left, divider, right := bounds, bounds, bounds
	if n.Axis == Horizontal {
		left.Width = first
		divider.X, divider.Width = bounds.X+first, effectiveGap
		right.X, right.Width = divider.X+effectiveGap, available-first
	} else {
		left.Height = first
		divider.Y, divider.Height = bounds.Y+first, effectiveGap
		right.Y, right.Height = divider.Y+effectiveGap, available-first
	}
	ratio := float32(0.5)
	if available > 0 {
		ratio = first / available
	}
	result.Dividers = append(result.Dividers, Divider{NodeID: n.ID, Axis: n.Axis, Rect: divider, Bounds: bounds, Ratio: ratio})
	arrange(n.First, left, gap, sizes, result)
	arrange(n.Second, right, gap, sizes, result)
}

func splitExtent(available, minFirst, minSecond, ratio float32) float32 {
	if available < minFirst+minSecond {
		return available * minFirst / (minFirst + minSecond)
	}
	if !finite(ratio) {
		ratio = 0.5
	}
	return max(minFirst, min(available-minSecond, available*max(MinRatio, min(MaxRatio, ratio))))
}

// Layout applies the workspace's zoom without modifying or replacing its tree.
func (w *Workspace) Layout(bounds Rect, dividerSize, minPaneSize float32) Geometry {
	root := w.Root
	if zoom := w.FindNode(w.ZoomedGroupID); zoom != nil && zoom.IsLeaf() {
		root = zoom
	}
	return ComputeLayout(root, bounds, dividerSize, minPaneSize)
}
