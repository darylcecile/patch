package panels

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type browserPanel struct {
	host      Host
	id        string
	web       *mygo.WebView
	address   string
	committed string
	status    string
	err       error
	closed    bool
	focus     ui.Handle
}

func newBrowser(host Host, spec workspace.Panel, _ string) Instance {
	// Even adding a Browser panel does not start WebKit until a URL is submitted.
	return &browserPanel{host: host, id: spec.ID, address: spec.Resource, committed: spec.Resource}
}

func browserURL(address string) bool {
	u, err := url.Parse(address)
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https")
}

func (p *browserPanel) rememberURL(address string, submitted bool) {
	if submitted || p.address == p.committed {
		p.address = address
	}
	p.committed = address
	if p.host.SetResource != nil && browserURL(address) {
		p.host.SetResource(p.id, address)
	}
}

func (p *browserPanel) navigate() {
	address := strings.TrimSpace(p.address)
	if !browserURL(address) {
		p.err = fmt.Errorf("enter a complete http:// or https:// URL")
		return
	}
	p.err, p.status = nil, "Loading…"
	if p.web != nil {
		p.web.Page().LoadURL(address)
		p.rememberURL(address, true)
		return
	}
	p.web, p.err = p.host.Window.NewWebView(mygo.WebViewOptions{URL: address, Page: mygo.PageOptions{Untrusted: true}})
	if p.err != nil {
		p.status = ""
		return
	}
	p.rememberURL(address, true)
	page := p.web.Page()
	page.OnDidNavigate(func(address string) {
		p.host.Window.Update(func() {
			if !p.closed {
				p.rememberURL(address, false)
			}
		})
	})
	page.OnDidFinishLoad(func() { p.status = ""; p.host.Window.Invalidate() })
	page.OnDidFailLoad(func(err *mygo.LoadError) { p.err = err; p.status = ""; p.host.Window.Invalidate() })
}

func (p *browserPanel) View(c *ui.Context) {
	ui.Column(c).Fill().MinHeight(0).Children(func() {
		ui.Row(c).Padding(8).Gap(6).Shrink(0).Children(func() {
			browserButton(c, "Back", p.web == nil).OnClick(func() { p.web.Page().GoBack() })
			browserButton(c, "Reload", p.web == nil).OnClick(func() { p.web.Page().Reload() })
			ui.TextInput(c.Key("browser-url"), &p.address).Bind(&p.focus).Label("Browser URL").Placeholder("https://example.com").Grow(1).MinWidth(0).OnSubmit(p.navigate)
			browserButton(c, "Go", false).OnClick(p.navigate)
		})
		if p.err != nil {
			ui.Text(c, p.err.Error()).Padding(8, 12).TextColor(c.Theme().Danger)
		}
		if p.status != "" {
			ui.Text(c, p.status).Padding(4, 12).FontSize(11).TextColor(c.Theme().TextMuted)
		}
		if p.web != nil {
			ui.WebView(c.Key("browser-page"), p.web).Grow(1).MinHeight(0)
		} else {
			empty(c, "Open a web page", "Enter a URL to start the system webview. This page has no access to Patch's native APIs.")
		}
	})
}

func browserButton(c *ui.Context, label string, disabled bool) ui.Element {
	button := ui.Button(c, label).Disabled(disabled).KeepFocus().Background(ui.Transparent).
		Border(0, ui.Transparent).TextColor(c.Theme().TextMuted).Radius(7)
	if disabled {
		button.Opacity(.5)
	} else if button.Pressed() {
		button.Background(c.Theme().TextMuted.Alpha(.12))
	} else if button.Hovered() {
		button.Background(c.Theme().TextMuted.Alpha(.07))
	}
	return button
}

func (p *browserPanel) Focus()       { p.focus.Focus() }
func (p *browserPanel) CancelFocus() { p.focus.CancelFocus() }

func (p *browserPanel) Close() {
	p.closed = true
	if p.web != nil {
		p.web.Destroy()
	}
}
