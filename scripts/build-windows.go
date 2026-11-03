//go:build ignore

// Run from the repository root with: go run ./scripts/build-windows.go
package main

import (
	"archive/zip"
	"debug/pe"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if err := buildWindows(); err != nil {
		fmt.Fprintln(os.Stderr, "Windows build:", err)
		os.Exit(1)
	}
}

func buildWindows() error {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		return err
	}
	var config struct{ Name, Version string }
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	// The pinned CLI supplies the native DLL, resources, GUI subsystem and
	// manifest. A plain GOOS=windows go build does not make a portable app.
	// MyGo builds in fresh staging, then replaces dist/windows-amd64 as a
	// whole on success, so files from previous builds cannot enter the ZIP.
	cmd := exec.Command("go", "tool", "mygo", "build", "-platform", "windows/amd64", "-skip-installer", "-o", "dist")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	dir := filepath.Join("dist", "windows-amd64")
	if err := checkExecutable(filepath.Join(dir, config.Name+".exe")); err != nil {
		return err
	}
	for _, name := range []string{
		"ghostty-vt.dll",
		"fonts/OFL.txt",
		"fonts/NERD-FONTS-LICENSE",
		"fonts/PROVENANCE.md",
		"fonts/UPSTREAM-README.md",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("required portable resource %s: %w", name, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("required portable resource %s is not a nonempty file", name)
		}
	}
	archive := filepath.Join("dist", fmt.Sprintf("patch-%s-windows-amd64.zip", config.Version))
	if err := zipDirectory(dir, archive); err != nil {
		return err
	}
	fmt.Println("Portable Windows x64 package:", archive)
	return nil
}

func checkExecutable(path string) error {
	f, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	header, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		return fmt.Errorf("%s is not a Windows x64 GUI executable", path)
	}
	if f.Section(".rsrc") == nil {
		return fmt.Errorf("%s is missing Windows metadata resources", path)
	}
	return nil
}

func zipDirectory(dir, destination string) error {
	f, err := os.CreateTemp(filepath.Dir(destination), ".patch-windows-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	w := zip.NewWriter(f)
	// Archive the entire MyGo output, including nested resources and any
	// additional native libraries a future imported plugin declares.
	if err := errors.Join(w.AddFS(os.DirFS(dir)), w.Close(), f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), destination)
}
