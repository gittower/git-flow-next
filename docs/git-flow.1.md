# GIT-FLOW(1)

## NAME

git-flow - A modern implementation of the git-flow branching model

## SYNOPSIS

**git-flow** [**--verbose**|**-v**] *command* [*args*]

## DESCRIPTION

git-flow-next is a modern, maintainable implementation of the git-flow branching model written in Go. It provides a set of Git extensions to provide high-level repository operations for Vincent Driessen's branching model.

This implementation maintains compatibility with existing git-flow repositories and supports both preset workflows (Classic GitFlow, GitHub Flow, GitLab Flow) and fully customizable branch configurations.

## GLOBAL OPTIONS

**--verbose**, **-v**
: Enable verbose output showing detailed operation information

**--help**, **-h**
: Show help information for any command

## COMMANDS

### Core Commands

**init**
: Initialize git-flow in the current repository. See **git-flow-init**(1).

**config**
: Manage git-flow configuration for base branches and topic branch types. See **git-flow-config**(1).

**overview**
: Display repository workflow overview. See **git-flow-overview**(1).

**integrate** [*branch*]
: Integrate a base branch into its parent (e.g. develop into main). See **git-flow-integrate**(1).

**version**
: Show version information. Prints `<version> (git-flow-next)`, so tooling that parses the first whitespace-separated token reads a bare version number.

### Topic Branch Commands

Topic branch commands are dynamically generated based on your configuration. Default types include **feature**, **release**, **hotfix**, **support**, plus any custom types you define.

Each topic branch type supports these subcommands:

**start** *name* [*base*]
: Create new topic branch. See **git-flow-start**(1).

**finish** [*name*]
: Complete and merge topic branch. See **git-flow-finish**(1).

**list** [*pattern*]
: List existing topic branches. See **git-flow-list**(1).

**update** [*name*]
: Update topic branch from parent. See **git-flow-update**(1).

**delete** *name*
: Delete topic branch. See **git-flow-delete**(1).

**rename** *old-name* *new-name*
: Rename topic branch. See **git-flow-rename**(1).

**checkout** *name*|*prefix*
: Switch to topic branch. See **git-flow-checkout**(1).

**track** *name*
: Create local branch tracking a remote topic branch. See **git-flow-track**(1).

### Shorthand Commands

**delete** [*name*]
: Delete current or specified topic branch. See **git-flow-delete**(1).

**update** [*name*]
: Update current or specified topic branch from parent. See **git-flow-update**(1).

**rebase** [*name*]
: Rebase current or specified topic branch (alias for update --rebase).

**rename** [*new-name*]
: Rename current topic branch. See **git-flow-rename**(1).

**finish** [*name*]
: Finish current or specified topic branch. See **git-flow-finish**(1).

**publish**
: Publish current topic branch to remote (planned feature).

## WORKFLOW PRESETS

git-flow-next supports three workflow presets:

**Classic GitFlow**
: Traditional git-flow with main, develop, feature/, release/, and hotfix/ branches.

**GitHub Flow**
: Simplified workflow with main and feature/ branches only.

**GitLab Flow**
: Multi-environment workflow with production, staging, main, feature/, and hotfix/ branches.

## CONFIGURATION

git-flow-next uses Git's configuration system to store all settings under the **gitflow.*** namespace. Configuration follows a three-layer hierarchy, but some options intentionally skip branch defaults:

1. **Branch Type Defaults** (gitflow.branch.*) - Default behavior for branch types (when applicable)
2. **Command Overrides** (gitflow.*type*.*command*.*) - Override defaults for specific operations  
3. **Command-line Flags** - Always take highest precedence

See **git-flow-config**(1) and **gitflow-config**(5) for detailed configuration reference.

## GIT-FLOW-AVH COMPATIBILITY

git-flow-next automatically detects and translates git-flow-avh configuration at runtime without modifying existing settings. Legacy configuration is mapped to the new format transparently.

## EXAMPLES

Initialize a repository with Classic GitFlow:
```bash
git flow init --preset=classic
```

Start a new feature:
```bash  
git flow feature start my-feature
```

Finish current branch (shorthand):
```bash
git flow finish
```

Configure custom branch type:
```bash
git flow config add topic bugfix develop --prefix=bug/
```

## FILES

**.git/config**
: Repository-specific git-flow configuration stored under gitflow.* keys

## SEE ALSO

**git-flow-init**(1), **git-flow-config**(1), **git-flow-completion**(1), **git-flow-start**(1), **git-flow-finish**(1), **git-flow-integrate**(1), **git-flow-update**(1), **git-flow-delete**(1), **git-flow-track**(1), **gitflow-config**(5), **git**(1)

## AUTHORS

git-flow-next is maintained by the Tower team at GitTower GmbH.

Original git-flow by Vincent Driessen. git-flow AVH Edition by Peter van der Does.

## BUGS

Report bugs at https://github.com/gittower/git-flow-next/issues

## COPYRIGHT

This is free software; see the source for copying conditions.
