# ai-codegen-harness

A constrained execution harness for AI codegen, in Go.

The idea: don't constrain the AI by *asking* it nicely in a prompt. Constrain
it **structurally**, with infrastructure the model cannot talk its way past.
The agent (the `claude` CLI) writes files freely; the harness then inspects the
result and enforces the constraint itself. A prompt that breaks it is reverted
wholesale.

## Install

```bash
go build -o harness .
```

## Usage

Run it **inside an existing git repo** (with at least one commit). The writable
paths are set once as arguments; the harness then reads prompts interactively.

```bash
cd my-project
harness src/              # only src/ is writable this session
harness src/ docs/        # allow multiple paths
```

Then type prompts at the `>` line. For each prompt:

1. claude writes files freely,
2. the harness checks every change against the allowed paths,
3. **in bounds** → committed (the prompt text is the commit message),
   **out of bounds** → the whole prompt is reverted, nothing committed.

You stay in the loop and keep prompting. Exit with `Ctrl-D` or `exit`.

```
> add a main entry point that prints the tasks
running agent...
accepted and committed (1 path(s)):
  src/main.go

> also create a top-level Makefile
running agent...
REJECTED: 1 violation(s):
  Makefile: created a file outside the allowed paths [src/]
reverted to the pre-prompt state. nothing committed.

> exit
bye.
```
