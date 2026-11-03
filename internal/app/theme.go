package app

import (
	"github.com/darylcecile/patch/internal/config"
	"github.com/egoist/mygo/ui"
)

const (
	paneGap    float32 = 9
	paneRadius float32 = 12
)

type palette struct {
	card, border, muted, selection, accent ui.Color
	backdrop                               ui.LinearGradient
}

func (a *shell) theme(c *ui.Context) palette {
	t := *c.Theme()
	if a.settings.Appearance == config.Light {
		t = *ui.LightTheme()
	} else if a.settings.Appearance == config.Dark {
		t = *ui.DarkTheme()
	}
	t.FontSize, t.Radius = 13, 8
	t.Accent = ui.Hex("#746493")
	if a.settings.Accent == "blue" {
		t.Accent = ui.Hex("#386f99")
	} else if a.settings.Accent == "green" {
		t.Accent = ui.Hex("#41785f")
	}
	t.AccentHover, t.AccentPressed, t.Focus = t.Accent.Mix(ui.RGB(0, 0, 0), .08), t.Accent.Mix(ui.RGB(0, 0, 0), .17), t.Accent
	t.FocusRingOffset = -2
	if a.settings.Density == config.Compact {
		t.Spacing = 3
	}
	p := palette{card: ui.Hex("#fdfcfb"), border: ui.Hex("#dcd6de"), muted: ui.Hex("#77747b"), selection: ui.Hex("#ffffff"), accent: t.Accent,
		backdrop: ui.LinearGradient{Angle: 180, Oklab: true, Stops: []ui.GradientStop{
			{Color: ui.Hex("#ebe3f2"), At: 0}, {Color: ui.Hex("#f1e9ed"), At: .42}, {Color: ui.Hex("#f5efd9"), At: 1},
		}}}
	if t.Dark {
		p.card, p.border, p.muted, p.selection = ui.Hex("#323039"), ui.Hex("#48414f"), ui.Hex("#bcb6c5"), ui.Hex("#423e4b")
		p.backdrop.Stops = []ui.GradientStop{{Color: ui.Hex("#211d28"), At: 0}, {Color: ui.Hex("#262321"), At: 1}}
		t.AccentText = ui.Hex("#ffffff")
		t.Background, t.TextMuted, t.Border = p.card, p.muted, p.border
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.Hex("#3d3945"), ui.Hex("#484250"), ui.Hex("#51495a")
	} else {
		t.Text, t.TextMuted, t.Background, t.Border = ui.Hex("#312e36"), p.muted, p.card, p.border
		t.Surface, t.SurfaceHover, t.SurfacePressed = ui.Hex("#f5f2f5"), ui.Hex("#eae5ee"), ui.Hex("#dfd8e6")
	}
	c.SetTheme(&t)
	return p
}

// Small SVGs use mygo's native icon primitive, sharing the chrome's inherited ink.
func icon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

var (
	plusIcon       = icon(`<path d="M12 5v14M5 12h14"/>`)
	closeIcon      = icon(`<path d="m6 6 12 12M6 18 18 6"/>`)
	sidebarIcon    = icon(`<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M9 4v16"/>`)
	terminalIcon   = icon(`<rect x="3" y="4" width="18" height="16" rx="2.5"/><path d="m7 9 3 3-3 3m6 0h4"/>`)
	folderIcon     = icon(`<path d="M3 7a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v9H3Z"/>`)
	changesIcon    = icon(`<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="7" r="2"/><path d="M6 7v10m2-3h4a6 6 0 0 0 6-5"/>`)
	splitRightIcon = icon(`<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M12 4v16"/>`)
	splitDownIcon  = icon(`<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M3 12h18"/>`)
	zoomIcon       = icon(`<path d="M14 3h7v7m0-7-7 7M10 21H3v-7m0 7 7-7"/>`)
	moreIcon       = icon(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`)
	settingsIcon   = icon(`<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3"/><circle cx="15" cy="17" r="3"/>`)
	gridIcon       = icon(`<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>`)
	chevronIcon    = icon(`<path d="m7 10 5 5 5-5"/>`)
)

func iconButton(c *ui.Context, key, label string, image *ui.SVG) ui.Element {
	return sizedIconButton(c, key, label, image, 28, 17)
}

func smallIconButton(c *ui.Context, key, label string, image *ui.SVG) ui.Element {
	return sizedIconButton(c, key, label, image, 22, 14)
}

func sizedIconButton(c *ui.Context, key, label string, image *ui.SVG, size, iconSize float32) ui.Element {
	b := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Size(size, size).Shrink(0).Padding(0).Center().Radius(7).
		Background(ui.Transparent).Border(0, ui.Transparent).TextColor(c.Theme().TextMuted).KeepFocus()
	if b.Pressed() {
		b.Background(c.Theme().TextMuted.Alpha(.12))
	} else if b.Hovered() {
		b.Background(c.Theme().TextMuted.Alpha(.07))
	}
	b.Children(func() { ui.Icon(c, image).Size(iconSize, iconSize).Shrink(0) })
	return b
}
