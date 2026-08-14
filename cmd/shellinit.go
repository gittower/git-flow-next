package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(shellInitCmd)
}

var shellInitCmd = &cobra.Command{
	Use:       "shell-init {bash|zsh|fish}",
	Short:     "Print the shell integration script",
	Long:      shellInitLong,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var script string
		switch args[0] {
		case "bash":
			script = bashShellInit
		case "zsh":
			script = zshShellInit
		case "fish":
			script = fishShellInit
		default:
			// Unreachable while ValidArgs and these cases agree. It is an error
			// rather than a silent no-op so that adding a fourth shell to one
			// without the other fails on the spot instead of printing nothing and
			// exiting 0, which reads to the user as a broken install.
			return fmt.Errorf("unsupported shell %q", args[0])
		}
		// io.WriteString rather than a Fprint: the scripts contain printf
		// directives of their own, which a formatting call would misread.
		_, err := io.WriteString(os.Stdout, script)
		return err
	},
}

const shellInitLong = `Print a shell script that makes git-flow change your shell's directory.

git-flow runs as a subprocess and cannot change the directory of the shell
that started it, so a command that would move you writes its destination to
the file named by GIT_FLOW_CD_FILE. The script printed here installs a
wrapper that supplies that file for one command at a time, changes directory
when the command came back with a destination, and removes the file
afterwards. git-flow's own output is never captured, so colors, progress and
prompts behave exactly as they do without the wrapper.

Both a "git" and a "git-flow" shell function are defined: the "git" function
is what makes the documented "git flow ..." form navigate, since git would
otherwise run the git-flow binary as a subprocess of its own.

Bash:

  eval "$(git flow shell-init bash)"

  # To load it for each session, add that line to ~/.bashrc

Zsh:

  eval "$(git flow shell-init zsh)"

  # To load it for each session, add that line to ~/.zshrc

Fish:

  git flow shell-init fish | source

  # To load it for each session, add that line to
  # ~/.config/fish/config.fish

Source the script LAST in your startup file: the "git" function replaces any
git alias or function defined before it.
`

// posixNavCore is the part of the wrapper bash and zsh share verbatim: the
// navigation contract in __git_flow_nav, plus the git() function that routes
// "git flow ..." into it. The two scripts differ ONLY in how git-flow() is
// defined, so the core lives here once — a fix applied to a copy would otherwise
// leave the other shell silently behind.
//
// Every construct is bash 3.2-compatible, the version macOS still ships. The
// pieces that look fussy are the ones that were measured:
//
//   - The single __git_flow_nav holds the whole contract and has a POSIX-valid
//     name; git() and git-flow() are thin wrappers around it.
//   - The assignment prefix, never export, keeps GIT_FLOW_CD_FILE scoped to the
//     single command, so it cannot leak into other programs or subshells.
//   - "command" prevents the functions from recursing into themselves.
//   - "|| __gf_status=$?" rather than "; __gf_status=$?" keeps the wrapper
//     transparent to a caller's set -e: an unguarded failure would abort the
//     function before the temp file is removed.
//   - local is declared and assigned separately, so mktemp's exit status is not
//     masked by local's always-zero one.
//   - "${TMPDIR:-/tmp}" falls back for an unset AND for a set-but-empty TMPDIR;
//     the fish script's test -n is the equivalent.
//   - The mktemp template is the one form both GNU coreutils and BSD accept.
//   - A failed cd is reported rather than swallowed, and the wrapper still
//     returns git-flow's own status.
const posixNavCore = `__git_flow_nav() {
    local __gf_file=""
    __gf_file=$(mktemp "${TMPDIR:-/tmp}/git-flow-cd.XXXXXX") || {
        command git-flow "$@"
        return $?
    }

    local __gf_status=0
    GIT_FLOW_CD_FILE="$__gf_file" command git-flow "$@" || __gf_status=$?

    if [ -s "$__gf_file" ]; then
        local __gf_dest=""
        IFS= read -r __gf_dest < "$__gf_file" || :
        if [ -n "$__gf_dest" ]; then
            cd -- "$__gf_dest" || printf 'git-flow: cannot enter %s\n' "$__gf_dest" >&2
        fi
    fi

    rm -f "$__gf_file"
    return $__gf_status
}

git() {
    if [ "${1:-}" = "flow" ]; then
        shift
        __git_flow_nav "$@"
        return $?
    fi
    command git "$@"
}
`

// bashShellInit is the shared core plus bash's guarded git-flow().
//
// git-flow() is skipped outright in POSIX mode and otherwise defined from a
// quoted eval. "bash --posix" rejects git-flow as a function name, and that error
// is FATAL inside eval — the shell exits and neither 2>/dev/null nor "|| :"
// intercepts it — which would take the git() function down with it.
const bashShellInit = posixNavCore + `
case ":$SHELLOPTS:" in
    *:posix:*) : ;;
    *) eval '
git-flow() {
    __git_flow_nav "$@"
}
' ;;
esac
`

// zshShellInit is the shared core plus git-flow() defined directly and
// unconditionally.
//
// The POSIX guard is deliberately absent. SHELLOPTS is normally unset in zsh, so
// expanding it aborts sourcing under a caller's set -u; and when zsh does
// inherit an exported SHELLOPTS containing posix from a parent bash, the guard
// would skip git-flow() for a restriction zsh does not have — zsh accepts the
// name perfectly well.
const zshShellInit = posixNavCore + `
git-flow() {
    __git_flow_nav "$@"
}
`

// fishShellInit is the wrapper for fish, which needs fish 3.0 or newer.
//
// Fish-specific points, each of which the wrapper tests pin:
//
//   - There is no $?; $status must be captured in the statement IMMEDIATELY
//     after the command, or an intervening test, if or echo replaces it.
//   - "env VAR=... git-flow" scopes the variable to one command; the inline
//     "VAR=... cmd" prefix needs fish 3.1, and env execs the binary, which also
//     sidesteps the function shadowing without needing "command".
//   - "set -e argv[1]" rather than a slice, which errors on an out-of-range
//     range in older fish.
//   - "if not cd ..." rather than "cd ...; or ...", so the diagnostic does not
//     itself clobber $status before the return.
//   - cd inside a function changes the shell's directory, because fish
//     functions are not subshells.
//   - "test -n $TMPDIR" rather than "set -q TMPDIR", which is TRUE for a
//     set-but-EMPTY variable: that took the branch, ran mktemp against /, and
//     lost navigation behind a raw mkstemp error while bash and zsh fell back
//     cleanly through "${TMPDIR:-/tmp}".
//   - cd takes no "--" here, where the bash and zsh core writes 'cd -- "$dest"'.
//     The guard is unnecessary in every shell — navigate.WriteDestinationTo only
//     ever writes an absolute path, so a destination cannot begin with "-" — and
//     fish's floor for this script is 3.0, whose handling of "--" in cd no test
//     here can exercise. The divergence is deliberate, not an oversight.
const fishShellInit = `function __git_flow_nav --description 'Run git-flow and act on its navigation channel'
    set -l gf_dir /tmp
    if test -n "$TMPDIR"
        set gf_dir $TMPDIR
    end
    set -l gf_file (mktemp "$gf_dir/git-flow-cd.XXXXXX")
    if test -z "$gf_file"
        command git-flow $argv
        return $status
    end

    env GIT_FLOW_CD_FILE="$gf_file" git-flow $argv
    set -l gf_status $status

    if test -s "$gf_file"
        read -l gf_dest <"$gf_file"
        if test -n "$gf_dest"
            if not cd "$gf_dest"
                echo "git-flow: cannot enter $gf_dest" >&2
            end
        end
    end

    rm -f "$gf_file"
    return $gf_status
end

function git-flow --description 'git-flow with automatic directory switching'
    __git_flow_nav $argv
end

function git --description 'git with git-flow directory switching'
    if test (count $argv) -gt 0; and test "$argv[1]" = flow
        set -e argv[1]
        __git_flow_nav $argv
        return $status
    end
    command git $argv
end
`
