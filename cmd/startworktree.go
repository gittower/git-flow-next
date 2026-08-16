package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// startWorktreeIntent is what 'git flow <type> start's three worktree flags
// resolved to.
//
// These flags are the ONE order-sensitive group in this CLI. Every other
// --x/--no-x pair goes through getBoolFlag, which prefers the positive flag
// whatever the order; start's worktree flags instead follow the LAST one on the
// command line, so a scripted default can be overridden by appending a flag
// rather than by editing the string that produced it.
//
// The ordering comes from pflag rather than from inspecting os.Args: pflag calls
// a flag's Set method as it walks the command line, so pointing all three flags
// at one shared intent makes the last Set the winner by construction. Shorthands
// (-w), clusters (-wq) and the explicit --worktree=false form are then handled by
// pflag's own parser instead of by a second, hand-written one.
type startWorktreeIntent struct {
	// create is nil until one of the three flags is seen, so a command line
	// that mentions none of them still falls through to the branch type's
	// gitflow.branch.<type>.worktree default.
	create *bool
	// path is the last --worktree-path value, empty when the flag was absent.
	path string
}

// set records an intent, replacing whatever an earlier flag recorded.
func (i *startWorktreeIntent) set(create bool) {
	i.create = &create
}

// worktreeIntentFlag is the pflag.Value behind --worktree and --no-worktree.
type worktreeIntentFlag struct {
	intent *startWorktreeIntent
	// sense is the intent recorded when the flag's value is true. --worktree
	// carries true; --no-worktree carries false and therefore inverts.
	sense bool
	// value is what the flag was last set to, so String reports it back.
	value bool
}

// Set records this flag's sense in the shared intent, in command-line order.
func (f *worktreeIntentFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.value = parsed
	// --worktree=false and --no-worktree=false state the negation of the flag's
	// own sense, the way pflag's own boolean flags read an explicit value.
	f.intent.set(parsed == f.sense)
	return nil
}

// Type reports "bool" so the usage line shows no value placeholder.
func (f *worktreeIntentFlag) Type() string { return "bool" }

// String reports the last value this flag was set to.
func (f *worktreeIntentFlag) String() string { return strconv.FormatBool(f.value) }

// worktreePathFlag is the pflag.Value behind --worktree-path. Naming a path is a
// creation signal, and it is recorded in order like the other two flags: so
// '--worktree-path x --no-worktree' creates no worktree, while
// '--no-worktree --worktree-path x' creates one at x.
type worktreePathFlag struct {
	intent *startWorktreeIntent
}

// Set stores the path and, when it is not blank, records the creation intent.
func (f *worktreePathFlag) Set(value string) error {
	f.intent.path = value
	// A blank path names no destination, so it signals nothing; the computed
	// path stays in force and any earlier intent stands.
	if strings.TrimSpace(value) != "" {
		f.intent.set(true)
	}
	return nil
}

// Type reports "string" so the usage line shows a value placeholder.
func (f *worktreePathFlag) Type() string { return "string" }

// String reports the path currently in effect.
func (f *worktreePathFlag) String() string { return f.intent.path }

// addStartWorktreeFlags registers start's three worktree flags against one
// shared intent, which the command's Run reads once parsing is done.
func addStartWorktreeFlags(cmd *cobra.Command, intent *startWorktreeIntent) {
	cmd.Flags().VarP(&worktreeIntentFlag{intent: intent, sense: true}, "worktree", "w", "Create a worktree for the new branch")
	cmd.Flags().Var(&worktreeIntentFlag{intent: intent, sense: false}, "no-worktree", "Do not create a worktree, even if the branch type defaults to one")
	cmd.Flags().Var(&worktreePathFlag{intent: intent}, "worktree-path", "Create the worktree at this path instead of the computed one (implies --worktree)")

	// Both booleans take their value optionally, as pflag's own bool flags do:
	// a bare --worktree means --worktree=true, and -w still clusters with -q.
	cmd.Flags().Lookup("worktree").NoOptDefVal = "true"
	cmd.Flags().Lookup("no-worktree").NoOptDefVal = "true"
}
