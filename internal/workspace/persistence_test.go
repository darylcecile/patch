package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSessionRoundTripAfterRearrangement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-state", "session.json")
	s := New("/projects/patch")
	w := s.ActiveWorkspace()
	first := w.Root
	file := addPanel(t, w, first.ID, Right, PanelSpec{Kind: Files, Title: "Source", Resource: "/projects/patch/internal/layout.go"})
	fileGroup, _ := w.FindPanel(file.ID)
	browser := addPanel(t, w, fileGroup.ID, Center, PanelSpec{Kind: Browser, Title: "Docs", Resource: "https://go.dev/doc/"})
	must(t, w.MovePanel(first.Panels[0].ID, fileGroup.ID, Bottom))
	must(t, w.SetRatio(w.Root.ID, 0.35))
	w.ToggleZoom()
	s.CreateWorkspace("Other project", "/projects/other")
	must(t, s.SelectWorkspace(w.ID))
	must(t, s.Save(path))
	loaded, err := Load(path)
	must(t, err)
	if !reflect.DeepEqual(s, loaded) {
		t.Fatal("saved session lost identity, descriptors, tree, focus or zoom")
	}
	// Replace a valid existing file and observe it through a fresh reader.
	must(t, loaded.ActiveWorkspace().SelectPanel(file.ID))
	const navigatedURL = "https://go.dev/ref/spec?section=types#Types"
	must(t, loaded.ActiveWorkspace().SetPanelResource(browser.ID, navigatedURL))
	must(t, loaded.Save(path))
	again, err := Load(path)
	must(t, err)
	if !reflect.DeepEqual(loaded, again) {
		t.Fatal("valid existing session was not replaced")
	}
	_, restoredBrowser := again.ActiveWorkspace().FindPanel(browser.ID)
	if restoredBrowser == nil || restoredBrowser.Resource != navigatedURL {
		t.Fatal("updated browser navigation resource did not survive reload")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	must(t, err)
	if len(entries) != 1 || entries[0].Name() != "session.json" {
		t.Fatal("save left temporary files behind")
	}
}

func TestCorruptSessionPreserved(t *testing.T) {
	valid := New("/projects/patch")
	encoded := snapshot(t, valid)
	for name, data := range map[string]string{
		"truncated":   `{"version":1,"workspaces":[`,
		"future":      `{"version":999,"workspaces":[],"activeWorkspaceId":""}`,
		"empty":       `{"version":1,"workspaces":[],"activeWorkspaceId":""}`,
		"null":        `null`,
		"unknown":     `{"version":1,"surprise":true}`,
		"trailing":    encoded + `{}`,
		"invalidJSON": `not json`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.json")
			must(t, os.WriteFile(path, []byte(data), 0o600))
			if state, err := Load(path); err == nil || state != nil {
				t.Fatal("invalid session accepted")
			}
			if err := valid.Save(path); err == nil {
				t.Fatal("autosave accepted corrupt original")
			}
			retained, err := os.ReadFile(path)
			must(t, err)
			if string(retained) != data {
				t.Fatal("original corrupt session was overwritten")
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file error not preserved: %v", err)
	}
}

func TestSessionRejectsBrokenTreesBeforeSaving(t *testing.T) {
	cases := map[string]func(*State){
		"dangling focus": func(s *State) { s.ActiveWorkspace().ActiveGroupID = "missing" },
		"duplicate panel": func(s *State) {
			g := s.ActiveWorkspace().Root
			g.Panels = append(g.Panels, g.Panels[0])
		},
		"one child": func(s *State) {
			w := s.ActiveWorkspace()
			w.Root = &Node{ID: "split", Axis: Horizontal, Ratio: 0.5, First: w.Root}
		},
		"ratio out of range": func(s *State) {
			w := s.ActiveWorkspace()
			addPanel(t, w, "", Right, PanelSpec{Kind: Files})
			w.Root.Ratio = 1.2
		},
		"wrong active tab": func(s *State) { s.ActiveWorkspace().Root.ActivePanelID = "missing" },
	}
	for name, corrupt := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.json")
			s := New("/projects/patch")
			must(t, s.Save(path))
			before, err := os.ReadFile(path)
			must(t, err)
			corrupt(s)
			if err := s.Save(path); err == nil {
				t.Fatal("invalid in-memory session was saved")
			}
			after, err := os.ReadFile(path)
			must(t, err)
			if string(before) != string(after) {
				t.Fatal("invalid save changed valid original")
			}
			data, err := json.Marshal(s)
			must(t, err)
			must(t, os.WriteFile(path, data, 0o600))
			if _, err := Load(path); err == nil {
				t.Fatal("invalid tree was loaded")
			}
		})
	}
}
