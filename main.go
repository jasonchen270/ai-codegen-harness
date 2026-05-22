// Command harness runs an AI agent (the claude CLI) under an enforced write
// constraint, in an interactive prompt loop.
//
// You pass the writable paths once; the harness then reads prompts from stdin.
// For each prompt the agent writes files freely, then the harness enforces one
// rule the model is never trusted to follow on its own:
//
//	default-deny: the whole project is frozen except the allowed paths. Any
//	              create, modify, or delete outside them is a violation.
//
// An accepted prompt is committed; a rejected one is reverted to the pre-prompt
// snapshot. You decide when to stop (Ctrl-D or "exit").
//
// Usage (run inside an existing git repo with at least one commit):
//
//	harness            # prompts for the writable paths, then loops
//	harness src/       # paths given as args; skips the path prompt
//	harness src/ docs/
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jasonchen270/ai-codegen-harness/internal/agent"
	"github.com/jasonchen270/ai-codegen-harness/internal/enforce"
	"github.com/jasonchen270/ai-codegen-harness/internal/git"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(allow []string) int {
	project, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Require an existing git repo with at least one commit, so the revert
	// target (HEAD) is well-defined.
	if !git.HasCommits(project) {
		fmt.Fprintln(os.Stderr, "error: not a git repo with any commits; run this inside one")
		return 1
	}

	in := bufio.NewScanner(os.Stdin)

	// If no paths were passed as args, ask for them once up front.
	if len(allow) == 0 {
		fmt.Print("writable paths (space-separated, e.g. src/ docs/): ")
		if !in.Scan() {
			return 1 // EOF before any input
		}
		allow = strings.Fields(in.Text())
		if len(allow) == 0 {
			fmt.Fprintln(os.Stderr, "error: no paths given; nothing would be writable")
			return 1
		}
	}

	fmt.Printf("\nharness: writable paths = %s\n", strings.Join(allow, ", "))
	fmt.Println("everything else is frozen. enter a prompt (Ctrl-D or 'exit' to quit).")

	for {
		fmt.Print("\n> ")
		if !in.Scan() {
			break // EOF (Ctrl-D)
		}
		prompt := strings.TrimSpace(in.Text())
		if prompt == "" {
			continue
		}
		if prompt == "exit" || prompt == "quit" {
			break
		}
		if code := once(project, prompt, allow); code == exitFatal {
			return 1
		}
	}
	fmt.Println("\nbye.")
	return 0
}

// Outcome codes for a single prompt; only exitFatal aborts the session.
const (
	exitOK = iota
	exitRejected
	exitFatal
)

// once runs one prompt under enforcement: agent writes, harness judges, then
// commits (in bounds) or reverts (violation). Never ends the session unless a
// git/IO operation itself fails.
func once(project, prompt string, allow []string) int {
	// A clean tree makes the revert target unambiguous.
	clean, err := git.IsClean(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitFatal
	}
	if !clean {
		fmt.Fprintln(os.Stderr, "error: working tree is dirty; commit or clean it before prompting")
		return exitFatal
	}

	fmt.Println("running agent...")
	if _, err := agent.Run(project, prompt, allow); err != nil {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
		_ = git.RevertToClean(project) // leave a clean tree after a failed agent
		return exitRejected
	}

	changes, err := git.Changes(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitFatal
	}
	allowed, violations := enforce.Evaluate(changes, allow)

	if len(violations) > 0 {
		fmt.Printf("\nREJECTED: %d violation(s):\n", len(violations))
		for _, v := range violations {
			fmt.Printf("  %s: %s\n", v.Path, v.Detail)
		}
		if err := git.RevertToClean(project); err != nil {
			fmt.Fprintf(os.Stderr, "error reverting: %v\n", err)
			return exitFatal
		}
		fmt.Println("reverted to the pre-prompt state. nothing committed.")
		return exitRejected
	}

	if len(allowed) == 0 {
		fmt.Println("no changes; nothing committed.")
		return exitOK
	}

	if err := git.CommitAll(project, prompt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitFatal
	}
	fmt.Printf("accepted and committed (%d path(s)):\n", len(allowed))
	for _, p := range allowed {
		fmt.Printf("  %s\n", p)
	}
	return exitOK
}
