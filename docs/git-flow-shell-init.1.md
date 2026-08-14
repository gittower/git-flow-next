# GIT-FLOW-SHELL-INIT(1)

## NAME

git-flow-shell-init - Print the shell integration script

## SYNOPSIS

**git-flow shell-init** *shell*

## DESCRIPTION

Print a shell script that makes git-flow change your shell's working directory.

git-flow runs as a subprocess and cannot change the directory of the shell that started it, so a command that would move you — **checkout** to a branch that has a worktree, **worktree add**, **worktree remove** — writes its absolute destination to the file named by **GIT_FLOW_CD_FILE** instead. The script printed here installs a wrapper that supplies that file, one command at a time, changes directory when the command came back with a destination, and removes the file afterwards.

For each invocation the wrapper creates a temporary file with **mktemp**(1), points **GIT_FLOW_CD_FILE** at it for that single command, runs git-flow with its output going straight to the terminal, changes directory if the file came back non-empty, and removes the file on every path — success, failure, or no destination at all — while returning git-flow's own exit code. Because git-flow's output is never captured, colors, progress and interactive prompts behave exactly as they do without the wrapper.

The variable is set **per invocation rather than exported**, so it does not leak into other programs, subshells, or scripts that happen to call git-flow from the same shell.

Supported shells: **bash**, **zsh**, **fish**. PowerShell and `cmd` are not supported.

## ARGUMENTS

**bash**
: Print the wrapper for bash, to be evaluated with `eval`. Requires bash 3.2 or newer.

**zsh**
: Print the wrapper for zsh, to be evaluated with `eval`.

**fish**
: Print the wrapper for fish, to be piped into `source`. Requires fish 3.0 or newer.

## OPTIONS

None.

## WHAT THE SCRIPT DEFINES

Each script defines **two** shell functions, both delegating to one shared helper:

**git**
: Intercepts **git flow ...** and hands it to the helper; every other git invocation is passed through to the real binary verbatim, with git's own exit code. This is the function that makes the documented **git flow ...** form navigate — a **git-flow** function alone never could, because git executes the `git-flow` binary as a subprocess of its own, and a directory change there dies with the subprocess.

**git-flow**
: The same behaviour for the direct **git-flow ...** form.

## ENVIRONMENT

**GIT_FLOW_CD_FILE**
: The navigation channel the wrapper supplies. When set to a writable path, a command that navigates writes its absolute destination there; git-flow only ever reads this variable, never sets it globally. See **git-flow-worktree**(1) for the channel's full contract, and **git-flow-checkout**(1) for what **checkout** writes to it.

: A command that navigates also prints a tip naming this command, but only when the channel is unused — with the wrapper installed the tip would be advice you have already taken. **--quiet** on **checkout** and **worktree add** suppresses it outright.

## EXAMPLES

### Bash

```bash
eval "$(git flow shell-init bash)"

# To load it for each session:
echo 'eval "$(git flow shell-init bash)"' >> ~/.bashrc
```

### Zsh

```bash
eval "$(git flow shell-init zsh)"

# To load it for each session:
echo 'eval "$(git flow shell-init zsh)"' >> ~/.zshrc
```

### Fish

```bash
git flow shell-init fish | source

# To load it for each session:
echo 'git flow shell-init fish | source' >> ~/.config/fish/config.fish
```

### Using it

```bash
$ git flow feature checkout user-auth
Worktree for branch 'feature/user-auth' at /home/you/my-project-worktrees/feature/user-auth
To switch to it: cd /home/you/my-project-worktrees/feature/user-auth
$ pwd
/home/you/my-project-worktrees/feature/user-auth
```

## EXIT STATUS

**0**
: The script was printed

**1**
: The shell argument is missing or names an unsupported shell

## SEE ALSO

**git-flow**(1), **git-flow-checkout**(1), **git-flow-worktree**(1), **git-flow-completion**(1), **bash**(1), **zsh**(1), **fish**(1)

## NOTES

- The script defines **both** a `git` and a `git-flow` function. Only the `git` one makes the advertised **git flow ...** form navigate.
- The `git` function **shadows an existing git alias or function** — the last definition wins. Source git-flow's script **last** in your startup file, or fold your own wrapper into it.
- Only **git flow ...** with `flow` as the **first** argument is intercepted. `git -C somewhere flow ...` runs the binary directly and does not navigate; parsing git's global options in shell would be fragile and is deliberately not attempted.
- **bash --posix** cannot define a function whose name contains a hyphen. Sourcing still succeeds and installs the `git` wrapper; `git-flow` then resolves to the binary as usual, so the direct form does not navigate in POSIX mode.
- If the destination cannot be entered — it was removed between the write and the change of directory — the wrapper reports it on standard error and still returns git-flow's own exit code, rather than reporting the failure of the `cd`.
- The wrapper is transparent to **set -e**: it never aborts a caller's script before removing its temporary file, and it never swallows a failure the caller is not asking about.
- Pressing Ctrl-C during a wrapped command may leave a single empty temporary file in **TMPDIR**. No `trap` prevents this reliably, and a predictable name would trade the leak for a symlink attack in a shared `/tmp`.
- The variable is in git-flow's environment for the wrapped command, so hooks git-flow spawns inherit it.
- This command is unrelated to **git-flow-completion**(1), which generates tab completion; the two are installed independently.
