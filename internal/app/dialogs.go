package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/workspace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *shell) showNewWorkspace() {
	a.workspaceName, a.dialogError = "", ""
	a.workspacePath = a.root
	if w := a.active(); w != nil {
		a.workspacePath = w.RootPath
	}
	a.newWorkspace = true
}

func (a *shell) openFolder() { a.chooseFolder(true) }

func (a *shell) chooseFolder(open bool) {
	if a.choosingFolder {
		return
	}
	a.choosingFolder = true
	start := a.workspacePath
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "Open project folder", ButtonLabel: "Open", Directory: true, DefaultPath: start})
		a.win.Update(func() {
			a.choosingFolder = false
			if a.closed {
				return
			}
			if err != nil {
				a.report(err)
				return
			}
			if len(paths) == 0 {
				return
			}
			a.workspacePath = paths[0]
			if open {
				a.workspaceName = ""
				a.createWorkspace()
			}
		})
	}()
}

func (a *shell) createWorkspace() {
	if a.creatingWorkspace {
		return
	}
	path := strings.TrimSpace(a.workspacePath)
	if path == "" {
		a.dialogError = "Choose a workspace directory."
		return
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			a.dialogError = err.Error()
			return
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}
	root, err := filepath.Abs(path)
	if err != nil {
		a.dialogError = err.Error()
		return
	}
	name := strings.TrimSpace(a.workspaceName)
	a.creatingWorkspace = true
	go func() {
		info, err := os.Stat(root)
		if err == nil && !info.IsDir() {
			err = fmt.Errorf("%s is not a directory", root)
		}
		a.win.Update(func() {
			a.creatingWorkspace = false
			if a.closed {
				return
			}
			if err != nil {
				a.dialogError = err.Error()
				return
			}
			if a.state == nil {
				a.state = &workspace.State{Version: workspace.Version}
			}
			w := a.state.CreateWorkspace(name, root)
			a.newWorkspace = false
			a.changed(defaultPanels(w))
		})
	}()
}

func (a *shell) dialogs(c *ui.Context) {
	ui.DialogBase(c.Key("new-workspace-dialog"), &a.newWorkspace, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(35, 25, 45, .25))
		panel.Width(460).MaxWidthPercent(90).Padding(24).Gap(14).Radius(18).Background(c.Theme().Background)
		ui.Text(c, "New workspace").FontSize(20).Bold()
		ui.Text(c, "A project directory with its own panel arrangement.").TextColor(c.Theme().TextMuted)
		ui.TextInput(c.Key("workspace-name"), &a.workspaceName).Label("Workspace name").Placeholder("Name (defaults to the directory name)").AutoFocus()
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c.Key("workspace-root"), &a.workspacePath).Label("Workspace directory").Grow(1).MinWidth(0).OnSubmit(a.createWorkspace)
			ui.Button(c.Key("browse-project"), "Browse…").Disabled(a.choosingFolder).OnClick(func() { a.chooseFolder(false) })
		})
		if a.dialogError != "" {
			ui.Text(c, a.dialogError).FontSize(12).TextColor(c.Theme().Danger)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			ui.Button(c, "Cancel").OnClick(func() { a.newWorkspace = false })
			ui.PrimaryButton(c, "Create workspace").Disabled(a.creatingWorkspace).OnClick(a.createWorkspace)
		})
	})
	ui.DialogBase(c.Key("settings-dialog"), &a.settingsOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(35, 25, 45, .25))
		panel.Width(480).MaxWidthPercent(90).MaxHeightPercent(90).Padding(24).Gap(18).Radius(18).Background(c.Theme().Background)
		ui.Row(c).Children(func() {
			ui.Text(c, "Settings").FontSize(20).Bold().Grow(1)
			iconButton(c, "close-settings", "Close settings", closeIcon).OnClick(func() { a.settingsOpen = false })
		})
		ui.Scroll(c).MaxHeight(550).Gap(18).Children(func() {
			a.settingsFields(c)
		})
		ui.Text(c, "Saved automatically to "+filepath.Join(a.home, "config.json")).FontSize(10).TextColor(c.Theme().TextMuted)
	})
}

func (a *shell) settingsFields(c *ui.Context) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Tab style").Bold()
		ui.Text(c, "Choose Above, Left Side or Right Side in View → Customize View → Tab Style.").FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Text(c, "Hide or show the sidebar separately from the View menu.").FontSize(11).TextColor(c.Theme().TextMuted)
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Appearance").Bold()
		options := []config.Appearance{config.System, config.Light, config.Dark}
		index := slices.Index(options, a.settings.Appearance)
		if ui.Segmented(c.Key("appearance-setting"), &index, "System", "Light", "Dark").Label("Appearance").Changed() {
			a.settings.Appearance = options[index]
			a.save()
		}
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Density").Bold()
		options := []config.Density{config.Comfortable, config.Compact}
		index := slices.Index(options, a.settings.Density)
		if ui.Segmented(c.Key("density-setting"), &index, "Comfortable", "Compact").Label("Density").Changed() {
			a.settings.Density = options[index]
			a.save()
		}
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Accent").Bold()
		options := []string{"", "blue", "green"}
		index := max(0, slices.Index(options, a.settings.Accent))
		if ui.Segmented(c.Key("accent-setting"), &index, "Lavender", "Blue", "Green").Label("Accent color").Changed() {
			a.settings.Accent = options[index]
			a.save()
		}
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, fmt.Sprintf("Sidebar width · %.0f pt", a.settings.SidebarWidth)).Bold()
		width := float64(a.settings.SidebarWidth)
		if ui.Slider(c.Key("sidebar-width-setting"), &width, 180, 420).Label("Sidebar width").Changed() {
			a.report(a.settings.SetSidebarWidth(float32(width)))
			a.save()
		}
	})
}
