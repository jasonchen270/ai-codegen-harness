# ai-codegen-harness

A constrained execution harness for AI codegen, written in Go. The agent (the `claude` CLI) writes files freely; the harness then inspects the result and enforces structural path constraints itself, reverting any prompt that writes outside the allowed paths.

## Prerequisites

- A Go toolchain
- The `claude` CLI
- git

## Installation

```bash
go build -o harness .
```

## Usage

Run it **inside an existing git repo** (with at least one commit). The writable paths are set once as arguments; the harness then reads prompts interactively.

```bash
cd my-project
harness src/              # only src/ is writable this session
harness src/ docs/        # allow multiple paths
```

Then type prompts at the `>` line. In-bounds changes are committed with the prompt text as the message; out-of-bounds changes are reverted. Exit with `Ctrl-D` or `exit`.
