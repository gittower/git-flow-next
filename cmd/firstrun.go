package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/spf13/cobra"
)

// firstRunActivation is the rootCmd PersistentPreRunE. On an otherwise
// unconfigured repository that carries a committable .gitflow file, it activates
// the shared config (prompt when interactive, auto-init when
// gitflow.shared.autoInit=true, hint otherwise) by copying gitflow.* into local
// .git/config before the invoked command proceeds. It is a no-op for the init
// command itself, outside a work tree / in a bare repo, when no .gitflow exists,
// and when the repo is already configured (local wins).
func firstRunActivation(cmd *cobra.Command) error {
	// Never intercept the command that configures the repository.
	if cmd.Name() == "init" {
		return nil
	}

	// Outside a git work tree (or a bare repo) there is nothing to activate.
	repo, err := openRepo()
	if err != nil {
		return nil
	}

	if !config.SharedConfigExists(repo) {
		return nil
	}

	unconfigured, err := config.IsUnconfiguredIgnoringShared(repo)
	if err != nil {
		// Be conservative: a detection failure must not block ordinary commands.
		return nil
	}
	if !unconfigured {
		// Already configured locally/globally/system: local config wins, the
		// shared file is not applied on first run.
		return nil
	}

	// The repo is unconfigured and a .gitflow is present. Validate it.
	valid, err := config.SharedConfigValid(repo)
	if err != nil {
		// Malformed / unparsable file: surface a clear error naming .gitflow.
		return err
	}
	if !valid {
		// Present but missing gitflow.version: not offered as initialized. Print a
		// hint so the user learns why it was not applied, then let the invoked
		// command report "not initialized" on its own.
		fmt.Fprintf(os.Stderr, "Note: a %s file is present but declares no gitflow.version, so it was not applied.\nRun 'git flow init --shared --force' to (re)author it.\n", config.SharedConfigFileName)
		return nil
	}

	switch {
	case config.SharedAutoInit(repo):
		return activateAutoInit(repo)
	case firstRunInteractive():
		return activatePrompt(repo)
	default:
		fmt.Fprintf(os.Stderr, "Note: this repository has a shared %s configuration that has not been activated.\nRun 'git flow init' to activate it, or set gitflow.shared.autoInit=true to activate automatically.\n", config.SharedConfigFileName)
		return nil
	}
}

// activateAutoInit performs a non-interactive activation. It refuses wholesale
// when the shared file declares an untrusted hook path, so an unattended run can
// never silently adopt executable hook configuration.
func activateAutoInit(repo *git.Repo) error {
	hookKeys, err := config.SharedConfigHookPathKeys(repo)
	if err != nil {
		return err
	}
	if len(hookKeys) > 0 && !config.SharedTrustHooks(repo) {
		return &errors.InvalidInputError{Message: fmt.Sprintf(
			"refusing to auto-initialize from %s: it declares an untrusted shared hook path (%s). Set gitflow.shared.trustHooks=true to allow it.",
			config.SharedConfigFileName, strings.Join(hookKeys, ", "))}
	}
	if _, err := config.CopySharedToLocal(repo); err != nil {
		return err
	}
	fmt.Printf("Note: auto-initialized git-flow from the shared %s file.\n", config.SharedConfigFileName)
	return nil
}

// activatePrompt asks the user (interactive stdin) whether to activate the shared
// config. Declining leaves the repo unconfigured; the invoked command then
// reports "not initialized" itself. Accepting copies the config and warns about
// any hook path skipped for lack of trust.
func activatePrompt(repo *git.Repo) error {
	fmt.Printf("This repository has a shared %s configuration that is not yet active.\nInitialize git-flow from %s now? [y/N]: ", config.SharedConfigFileName, config.SharedConfigFileName)

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return nil
	}

	result, err := config.CopySharedToLocal(repo)
	if err != nil {
		return err
	}
	for _, k := range result.SkippedHookKeys {
		fmt.Fprintf(os.Stderr, "Warning: skipped shared hook path %q; set gitflow.shared.trustHooks=true to apply it.\n", k)
	}
	fmt.Printf("Initialized git-flow from the shared %s file.\n", config.SharedConfigFileName)
	return nil
}

// firstRunInteractive reports whether the first-run activation decision should
// treat stdin as interactive. It honors the test-only GIT_FLOW_ASSUME_INTERACTIVE
// override (set to "1" to force interactive, since the subprocess test harness
// cannot allocate a PTY); otherwise it uses the real terminal check, so a pipe or
// /dev/null counts as non-interactive.
func firstRunInteractive() bool {
	if os.Getenv("GIT_FLOW_ASSUME_INTERACTIVE") == "1" {
		return true
	}
	return isTerminal(os.Stdin.Fd())
}
