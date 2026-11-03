// Package config stores user preferences separately from workspace session state.
package config

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/darylcecile/patch/internal/config/storage"
)

const Version = 1

const (
	MinSidebarWidth float32 = 180
	MaxSidebarWidth float32 = 420
)

type Navigation string

const (
	Sidebar Navigation = "sidebar"
	Tabs    Navigation = "tabs"
	// Both is accepted for legacy config compatibility; Load normalizes it to Sidebar.
	// Deprecated: use Sidebar or Tabs.
	Both Navigation = "both"
)

type Appearance string

type SidebarPosition string

const (
	SidebarLeft  SidebarPosition = "left"
	SidebarRight SidebarPosition = "right"
)

const (
	Light  Appearance = "light"
	Dark   Appearance = "dark"
	System Appearance = "system"
)

type Density string

const (
	Comfortable Density = "comfortable"
	Compact     Density = "compact"
)

// SidebarVisible and SidebarWidth are independent preferences; switching to Tabs
// need not destroy the user's chosen sidebar state. Accent is an optional renderer token.
type Config struct {
	Version         int             `json:"version"`
	SidebarVisible  bool            `json:"sidebarVisible"`
	SidebarWidth    float32         `json:"sidebarWidth"`
	SidebarPosition SidebarPosition `json:"sidebarPosition"`
	Navigation      Navigation      `json:"navigation"`
	Appearance      Appearance      `json:"appearance"`
	Density         Density         `json:"density"`
	Accent          string          `json:"accent,omitempty"`
}

func Default() *Config {
	return &Config{
		Version: Version, SidebarVisible: true, SidebarWidth: 240, SidebarPosition: SidebarLeft,
		Navigation: Sidebar, Appearance: System, Density: Comfortable,
	}
}

func (c *Config) SetSidebarWidth(width float32) error {
	if math.IsNaN(float64(width)) || math.IsInf(float64(width), 0) {
		return fmt.Errorf("sidebar width must be finite")
	}
	c.SidebarWidth = max(MinSidebarWidth, min(MaxSidebarWidth, width))
	return nil
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Version != Version {
		return fmt.Errorf("unsupported config version %d (expected %d)", c.Version, Version)
	}
	if c.Navigation != Sidebar && c.Navigation != Tabs && c.Navigation != Both {
		return fmt.Errorf("invalid navigation mode %q", c.Navigation)
	}
	if c.SidebarPosition != SidebarLeft && c.SidebarPosition != SidebarRight {
		return fmt.Errorf("invalid sidebar position %q", c.SidebarPosition)
	}
	if c.Appearance != Light && c.Appearance != Dark && c.Appearance != System {
		return fmt.Errorf("invalid appearance %q", c.Appearance)
	}
	if c.Density != Comfortable && c.Density != Compact {
		return fmt.Errorf("invalid density %q", c.Density)
	}
	if math.IsNaN(float64(c.SidebarWidth)) || c.SidebarWidth < MinSidebarWidth || c.SidebarWidth > MaxSidebarWidth {
		return fmt.Errorf("sidebar width must be within [%g, %g]", MinSidebarWidth, MaxSidebarWidth)
	}
	return nil
}

// Load overlays provided fields on defaults, requires an explicit supported
// version, and clamps sidebar width. Valid legacy Both navigation is normalized
// to Sidebar in memory. Loading never rewrites the file; missing/corrupt files
// return an error. The entrypoint decides whether to use defaults in memory.
func Load(path string) (*Config, error) {
	c := Default()
	c.Version = 0
	if err := storage.ReadJSON(path, c); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := c.SetSidebarWidth(c.SidebarWidth); err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	if c.Navigation == Both {
		c.Navigation = Sidebar
	}
	return c, nil
}

// Save refuses to replace corrupt or unsupported config. Explicit recovery can
// first back up/remove that file; ordinary autosave cannot clobber the original.
func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if _, err := Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("preserve existing config: %w", err)
	}
	if err := storage.WriteJSON(path, c); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
