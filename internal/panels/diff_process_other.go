//go:build !windows

package panels

import "os/exec"

func configureDiffGitProcess(*exec.Cmd) {}
