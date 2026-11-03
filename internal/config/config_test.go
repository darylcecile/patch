package config

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigRoundTripAndIndependentPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-state", "config.json")
	c := Default()
	c.SidebarVisible = false
	c.SidebarPosition = SidebarRight
	c.Navigation, c.Appearance, c.Density, c.Accent = Tabs, Dark, Compact, "teal"
	if err := c.SetSidebarWidth(333); err != nil {
		t.Fatal(err)
	}
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c, loaded) {
		t.Fatal("preferences changed during roundtrip")
	}
	loaded.Navigation = Both
	if err := loaded.Save(path); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	want := *loaded
	want.Navigation = Sidebar
	if err != nil || !reflect.DeepEqual(again, &want) {
		t.Fatalf("legacy navigation normalization lost independent preferences: %+v, %v", again, err)
	}
	retained, err := os.ReadFile(path)
	if err != nil || string(retained) != string(original) {
		t.Fatal("loading legacy navigation rewrote the original config")
	}
}

func TestPartialConfigDefaultsAndWidthClamping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"sidebarVisible":false,"sidebarWidth":999}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Navigation != Sidebar || c.SidebarPosition != SidebarLeft || c.SidebarVisible || c.SidebarWidth != MaxSidebarWidth || c.Appearance != System || c.Density != Comfortable {
		t.Fatalf("defaults or clamping not applied: %+v", c)
	}
	if err := c.SetSidebarWidth(-20); err != nil || c.SidebarWidth != MinSidebarWidth {
		t.Fatal("minimum width not clamped")
	}
	for _, invalid := range []float32{float32(math.NaN()), float32(math.Inf(1))} {
		if err := c.SetSidebarWidth(invalid); err == nil || c.SidebarWidth != MinSidebarWidth {
			t.Fatal("nonfinite width not rejected without changing existing width")
		}
	}
}

func TestInvalidConfigAndOriginalPreservation(t *testing.T) {
	for name, data := range map[string]string{
		"corrupt":        `{"version":`,
		"future":         `{"version":2}`,
		"unversioned":    `{"density":"compact"}`,
		"enum":           `{"version":1,"appearance":"sepia"}`,
		"sidebar side":   `{"version":1,"sidebarPosition":"bottom"}`,
		"legacy invalid": `{"version":1,"navigation":"both","appearance":"sepia"}`,
		"type":           `{"version":1,"sidebarVisible":"false"}`,
		"unknown":        `{"version":1,"sidebarWitdh":200}`,
		"trailing":       `{"version":1} {}`,
		"null":           `null`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if c, err := Load(path); err == nil || c != nil {
				t.Fatal("invalid config accepted")
			}
			if err := Default().Save(path); err == nil {
				t.Fatal("corrupt config was replaced by defaults")
			}
			retained, err := os.ReadFile(path)
			if err != nil || string(retained) != data {
				t.Fatal("corrupt original was not preserved")
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file error not preserved: %v", err)
	}
}

func TestSaveReturnsFilesystemError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Default().Save(filepath.Join(file, "config.json")); err == nil {
		t.Fatal("filesystem failure was hidden")
	}
	retained, err := os.ReadFile(file)
	if err != nil || string(retained) != "original" {
		t.Fatal("failed save changed original file")
	}
}
