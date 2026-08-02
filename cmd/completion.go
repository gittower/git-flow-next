package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	// Disable Cobra's default completion command; we provide our own
	// that adds git-subcommand integration for "git flow".
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:       "completion {bash|zsh|fish|powershell}",
	Short:     "Generate shell completion scripts",
	Long:      completionLong,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return genBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return genFishCompletion(os.Stdout)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

const completionLong = `Generate shell completion scripts for git-flow.

For bash, zsh, and fish the generated scripts provide tab completion for
both "git-flow" (direct) and "git flow" (git subcommand) invocation styles.
PowerShell completion supports "git-flow" only.

Bash:

  source <(git flow completion bash)

  # To load completions for each session, execute once:
  # Linux:
  git flow completion bash > /etc/bash_completion.d/git-flow
  # macOS (Homebrew):
  git flow completion bash > $(brew --prefix)/etc/bash_completion.d/git-flow

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it first:
  echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  git flow completion zsh > "${fpath[1]}/_git-flow"

  # You will need to start a new shell for this setup to take effect.

Fish:

  git flow completion fish | source

  # To load completions for each session, execute once:
  git flow completion fish > ~/.config/fish/completions/git-flow.fish

PowerShell:

  git flow completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, add the output of
  # the above command to your PowerShell profile.
`

// genBashCompletion generates bash completion with a bridge for "git flow" usage.
// Git's bash completion calls _git_<subcommand>() when completing "git <subcommand>",
// so we append a _git_flow() function that remaps the completion context and
// delegates to Cobra's __start_git-flow.
func genBashCompletion(w io.Writer) error {
	if err := rootCmd.GenBashCompletionV2(w, true); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, bashBridge)
	return err
}

const bashBridge = `
# Bridge for "git flow" subcommand completion.
# Git's bash completion calls _git_flow() when the user types "git flow <tab>".
# This remaps the completion context so Cobra sees "git-flow ..." and delegates.
_git_flow()
{
    # Rewrite "git flow" to "git-flow" so Cobra recognises the binary name.
    # Both forms are 8 characters, so COMP_POINT stays unchanged.
    COMP_LINE="${COMP_LINE/git flow/git-flow}"
    COMP_WORDS=("git-flow" "${COMP_WORDS[@]:2}")
    ((COMP_CWORD--))

    # Reset compopt to match the defaults Cobra expects when compopt is available
    # ("complete -o default", no nospace). Git's completion context may have set
    # options that interfere with trailing-space behaviour.
    if [[ $(type -t compopt) = "builtin" ]]; then
        compopt -o default +o nospace
    fi

    __start_git-flow
}
`

// genFishCompletion generates fish completion with support for "git flow" usage.
// Fish has no automatic git-subcommand discovery, so we append registrations
// that delegate to git-flow's __complete when the user types "git flow <tab>".
func genFishCompletion(w io.Writer) error {
	if err := rootCmd.GenFishCompletion(w, true); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, fishBridge)
	return err
}

const fishBridge = `
# Bridge for "git flow" subcommand completion.
# Registers completions for "git flow ..." by delegating to git-flow's
# Cobra-generated __complete command.
function __fish_git_flow_complete
    set -l tokens (commandline -opc)
    set -l current (commandline -ct)
    # Skip "git" and "flow", pass the rest to git-flow __complete
    set -l args $tokens[3..-1]

    set -l results (GIT_FLOW_ACTIVE_HELP=0 git-flow __complete $args $current 2>/dev/null)
    if test (count $results) -eq 0
        return
    end

    # The last line is the completion directive; strip it
    set -e results[-1]
    string join \n -- $results
end

# Register "flow" as a git subcommand
complete -c git -f -n '__fish_use_subcommand' -a flow -d 'Manage branches with git-flow'

# Delegate deeper completions to git-flow
complete -c git -f -n '__fish_seen_subcommand_from flow' -a '(__fish_git_flow_complete)'
`
