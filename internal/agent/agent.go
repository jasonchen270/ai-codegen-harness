// Package agent runs the AI that writes code into the project. It shells out to
// the `claude` CLI in headless mode and lets it edit files freely.
//
// The harness deliberately does NOT rely on claude obeying any restriction: the
// allowed paths are passed to claude only as a hint in the prompt. The real
// constraint is enforced afterward by the harness against git. If claude ignores
// the hint and writes out of bounds, the run is rejected and reverted. That
// proves the constraint is structural, not advisory.
package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run invokes claude in dir to act on the prompt, told it should only write
// under allow. Returns claude's stdout (for logging) or an error if the process
// itself failed to run.
func Run(dir, prompt string, allow []string) (string, error) {
	full := fmt.Sprintf(
		"%s\n\nYou may ONLY create or modify files under these paths: %s. "+
			"Do not touch anything else. When done, stop.",
		prompt, strings.Join(allow, ", "))

	cmd := exec.Command("claude",
		"-p", full,
		"--permission-mode", "acceptEdits",
		"--add-dir", dir,
	)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr // surface claude's progress/errors to the user
	out, err := cmd.Output()
	if err != nil {
		return string(out), fmt.Errorf("claude run failed: %w", err)
	}
	return string(out), nil
}
