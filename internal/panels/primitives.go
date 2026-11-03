package panels

import (
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo/ui"
)

type primitives struct {
	host     Host
	query    string
	checked  bool
	switched bool
	segment  int
	value    float64
	choice   string
	scroll   ui.ScrollState
	focus    ui.Handle
}

func newPrimitives(host Host, _ workspace.Panel, _ string) Instance {
	return &primitives{host: host, value: 50}
}

func (p *primitives) Close() {}
func (p *primitives) View(c *ui.Context) {
	ui.Scroll(c).TrackScroll(&p.scroll).Fill().Padding(24).Gap(20).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "Native building blocks").FontSize(20).Bold()
			ui.Text(c, "Reusable Go controls with keyboard focus, selection and system text input.").TextColor(c.Theme().TextMuted)
		})
		ui.TextInput(c, &p.query).Bind(&p.focus).Label("Sample text field").Placeholder("Try native text selection…")
		ui.Segmented(c, &p.segment, "First", "Second", "Third").Label("Sample segmented control")
		ui.Row(c).Wrap().Gap(20).Children(func() {
			ui.Checkbox(c, &p.checked, "Checkbox")
			ui.Switch(c, &p.switched).Label("Switch")
		})
		ui.Slider(c, &p.value, 0, 100).Label("Sample slider")
		ui.Row(c).Wrap().Gap(8).Children(func() {
			ui.PrimaryButton(c, "Open settings").OnClick(p.host.Settings)
			ui.Button(c, "New terminal").OnClick(func() { p.host.Open(workspace.Terminal) })
			ui.Button(c, "Disabled").Disabled(true)
		})
		ui.Divider(c)
		ui.Text(c, "These examples use mygo widgets directly. Panel factories can compose the same primitives without HTML or a JavaScript runtime.").TextColor(c.Theme().TextMuted)
	})
}

func (p *primitives) Focus()       { p.focus.Focus() }
func (p *primitives) CancelFocus() { p.focus.CancelFocus() }

type welcome struct {
	host  Host
	root  string
	focus ui.Handle
}

func (p *welcome) Close()       {}
func (p *welcome) Focus()       { p.focus.Focus() }
func (p *welcome) CancelFocus() { p.focus.CancelFocus() }
func (p *welcome) View(c *ui.Context) {
	ui.Column(c.Key("welcome-root")).Bind(&p.focus).Focusable().FocusRing(false).Fill().Center().Padding(28).Gap(16).Children(func() {
		ui.Text(c, "Workspace is empty").FontSize(22).Bold()
		ui.Text(c, p.root).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Text(c, "Open a terminal or explore the project files.").MaxWidth(390).TextColor(c.Theme().TextMuted)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			ui.PrimaryButton(c, "Open terminal").OnClick(func() { p.host.Open(workspace.Terminal) })
			ui.Button(c, "Explore files").OnClick(func() { p.host.Open(workspace.Files) })
		})
	})
}
