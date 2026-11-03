package panels

import (
	"context"
	"os"
	"os/exec"
	"time"
)

func diffGitCommand(ctx context.Context, root string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root, "--no-pager"}, args...)...)
	// A preview must not refresh the index or fetch missing objects in a partial clone.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0")
	cmd.WaitDelay = time.Second
	configureDiffGitProcess(cmd)
	return cmd
}
