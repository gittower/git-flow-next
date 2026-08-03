# GIT-FLOW-COMPLETION(1)

## NAME

git-flow-completion - Generate shell completion scripts

## SYNOPSIS

**git-flow completion** *shell*

## DESCRIPTION

Generate shell completion scripts for git-flow. For bash, zsh, and fish the generated scripts provide tab completion for both **git-flow** (direct invocation) and **git flow** (git subcommand) command forms. PowerShell completion supports **git-flow** only.

Supported shells: **bash**, **zsh**, **fish**, **powershell**.

## SHELL-SPECIFIC BEHAVIOUR

### Bash

The generated script includes Cobra's standard completion for the **git-flow** binary plus a `_git_flow()` bridge function. Git's bash completion system calls `_git_<subcommand>()` when completing **git \<subcommand\>**, so this bridge remaps the completion context and delegates to Cobra's `__start_git-flow`.

### Zsh

No bridge is needed. Cobra's output defines a `_git-flow` function, and zsh's built-in **_git** completion auto-discovers `_git-<subcommand>` functions. Both **git-flow** and **git flow** completion work once the script is sourced or placed in **fpath**.

### Fish

The generated script includes Cobra's standard completion for **git-flow** plus a bridge that registers completions for the **git** command when the subcommand is **flow**. Deeper completions are delegated to git-flow's built-in `__complete` mechanism.

### PowerShell

Uses Cobra's standard PowerShell completion for the **git-flow** binary. No subcommand bridge is provided, so **git flow** completion is not supported in PowerShell.

## INSTALLATION

### Bash

```bash
# Load in the current session
source <(git flow completion bash)

# Persist for all sessions (Linux)
git flow completion bash > /etc/bash_completion.d/git-flow

# Persist for all sessions (macOS with Homebrew)
git flow completion bash > $(brew --prefix)/etc/bash_completion.d/git-flow
```

### Zsh

```bash
# Enable completion if not already configured
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Install the completion function
git flow completion zsh > "${fpath[1]}/_git-flow"

# Restart your shell
```

### Fish

```bash
# Load in the current session
git flow completion fish | source

# Persist for all sessions
git flow completion fish > ~/.config/fish/completions/git-flow.fish
```

### PowerShell

```powershell
# Load in the current session
git flow completion powershell | Out-String | Invoke-Expression

# Persist: add the above to your PowerShell profile
```

## EXAMPLES

### Generate and preview bash completion

```bash
git flow completion bash
```

### Install bash completion system-wide

```bash
sudo git flow completion bash > /etc/bash_completion.d/git-flow
```

### Install zsh completion for the current user

```bash
mkdir -p ~/.zsh/completions
echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
git flow completion zsh > ~/.zsh/completions/_git-flow
```

## EXIT STATUS

**0**
: Completion script generated successfully

## SEE ALSO

**git-flow**(1), **bash**(1), **zshcompsys**(1), **fish**(1)
