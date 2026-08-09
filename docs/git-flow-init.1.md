# GIT-FLOW-INIT(1)

## NAME

git-flow-init - Initialize git-flow in a repository

## SYNOPSIS

**git-flow init** [**-f**|**--force**] [**--preset**=*preset*] [**--custom**] [**--defaults**] [**--shared**|**--local**|**--global**|**--system**|**--file**=*path*] [*options*]

## DESCRIPTION

Initialize git-flow configuration in the current Git repository. This command sets up the branch structure and configuration needed for git-flow operations.

**git-flow init** supports three initialization modes:

1. **Interactive Mode** (default) - Prompts for branch names and prefixes one question at a time, each showing its default in brackets that Enter accepts
2. **Preset Mode** - Automatically applies a predefined workflow configuration  
3. **Custom Mode** - Sets up only the trunk branch and shows configuration commands

## OPTIONS

### General Options

**-f**, **--force**
: Force reconfiguration of git-flow even if already initialized. Without this option, **git flow init** will fail if configuration already exists (in non-interactive mode) or prompt for confirmation (in interactive mode).

### Preset Options

**--preset**=*preset*, **-p** *preset*
: Apply a predefined workflow preset. Valid values: **classic**, **github**, **gitlab**

**--custom**
: Enable custom configuration mode. Prompts for trunk branch and displays configuration commands.

**--defaults**, **-d**
: Use default branch naming conventions without prompting for customization.

**--no-create-branches**
: Don't create branches even if they don't exist in the repository.

### Configuration Scope Options

These options control where git-flow configuration is stored. Only one scope option may be specified at a time. When no scope option is given, git-flow reads from merged config (local > global > system precedence) and writes to local config.

**--shared**
: Author the configuration into a committable **.gitflow** file at the repository top level, then copy the **gitflow.*** keys into the repository's local **.git/config**. Committing **.gitflow** lets teammates share one git-flow configuration: on a fresh clone, git-flow offers to activate it (see **gitflow-config**(5), FIRST-RUN ACTIVATION). **--shared** is mutually exclusive with **--local**, **--global**, **--system**, and **--file**. Without **--force**, running **--shared** again when a **.gitflow** already exists fails and leaves the file untouched; with **--force** the file is rewritten and re-copied (removing any stale managed keys from local config).

**--local**
: Read and write configuration only in the repository's **.git/config** file. This is the default for writes when no scope option is specified.

**--global**
: Read and write configuration in the user's global **~/.gitconfig** file. Useful for setting up defaults that apply to all repositories.

**--system**
: Read and write configuration in the system-wide **/etc/gitconfig** file. Typically requires administrator privileges.

**--file**=*path*
: Read and write configuration in the specified file. The parent directory must exist and be writable. Paths may be absolute or relative to the current working directory. Useful for managing shared configuration files.

### Branch Name Overrides

**--main**=*name*, **-m** *name*
: Override main branch name (default: main)

**--develop**=*name*, **-e** *name*
: Override develop branch name (default: develop)

### Prefix Overrides

**--feature**=*prefix*
: Override feature branch prefix (default: feature/)

**--bugfix**=*prefix*, **-b** *prefix*
: Override bugfix branch prefix (default: bugfix/)

**--release**=*prefix*, **-r** *prefix*
: Override release branch prefix (default: release/)

**--hotfix**=*prefix*, **-x** *prefix*
: Override hotfix branch prefix (default: hotfix/)

**--support**=*prefix*, **-s** *prefix*
: Override support branch prefix (default: support/)

**--tag**=*prefix*, **-t** *prefix*
: Override version tag prefix (default: none)

A prefix is stored exactly as given; no separator is appended. The trailing slash in the defaults is a naming convention, not a requirement — `feature/`, `feature_` and `feature-` are all valid and produce the branches `feature/login`, `feature_login` and `feature-login` respectively.

## PRESETS

### Classic GitFlow

Traditional git-flow workflow with the following structure:

- **main** - Production releases (trunk)
- **develop** - Integration branch (auto-updates from main)  
- **feature/** - New features (parent: develop)
- **release/** - Release preparation (parent: main, starts from develop, creates tags)
- **hotfix/** - Emergency fixes (parent: main, creates tags)
- **support/** - Long-term support (parent: main)

### GitHub Flow

Simplified workflow for continuous deployment:

- **main** - Production branch (trunk)
- **feature/** - All development work (parent: main)

### GitLab Flow

Multi-environment workflow for staged deployments:

- **production** - Production environment (trunk)
- **staging** - Staging environment (parent: production)
- **main** - Development integration (parent: staging)
- **feature/** - Development work (parent: main)
- **hotfix/** - Production fixes (parent: production)

## INTERACTIVE MODE

When run without options, **git-flow init** asks eight questions in sequence, each showing its default in brackets. Press Enter to keep the default:

```
Branch name for production releases [main]: 
Branch name for development [develop]: 
Feature branch prefix [feature/]: 
Bugfix branch prefix [bugfix/]: 
Release branch prefix [release/]: 
Hotfix branch prefix [hotfix/]: 
Support branch prefix [support/]: 
Version tag prefix []: 
```

Answers are trimmed of surrounding whitespace and then stored verbatim. No separator is appended to a prefix, so answering the feature prompt with `feature_` stores exactly that prefix, and `git flow feature start login` then creates the branch `feature_login`.

An answer that is empty after trimming leaves the default in place.

If a git-flow-avh configuration is already present, **git-flow init** imports it instead of prompting.

## CUSTOM MODE

With **--custom**, only prompts for the trunk branch:

```
? What's your trunk branch (holds production code)? [main] production
✓ Trunk branch: production

Configuration commands:
  git-flow config add base <name> [<parent>] [options...]
  git-flow config add topic <name> <parent> [options...]
  [... full command reference displayed ...]
```

## EXAMPLES

Initialize with Classic GitFlow using defaults:
```bash
git flow init --preset=classic
```

Initialize with defaults without prompting:
```bash
git flow init --defaults
```

Initialize with preset and defaults:
```bash
git flow init --preset=classic --defaults
```

Initialize GitHub Flow with custom main branch:
```bash
git flow init --preset=github --main=master
```

Initialize Classic GitFlow with custom branch names:
```bash
git flow init --preset=classic --main=master --develop=dev --feature=feat/
```

Initialize with short flags:
```bash
git flow init -p classic -d -m master -b bug/ -r rel/
```

Custom configuration mode:
```bash
git flow init --custom
```

Interactive initialization:
```bash
git flow init
```

Reconfigure git-flow with new settings:
```bash
git flow init --force --feature=feat/
```

Force reinitialize with github preset:
```bash
git flow init --preset=github --force
```

Reconfigure with short flag:
```bash
git flow init -f --defaults
```

Initialize with global scope (user-wide defaults):
```bash
git flow init --defaults --global
```

Initialize with local scope (repository-specific):
```bash
git flow init --defaults --local
```

Initialize with configuration file:
```bash
git flow init --defaults --file=/path/to/custom-gitflow.config
```

Initialize a shared, committable configuration:
```bash
git flow init --defaults --shared
git add .gitflow && git commit -m "Add shared git-flow configuration"
```

Create local config when global config already exists:
```bash
git flow init --defaults --local
```

## CONFIGURATION

By default, git-flow stores configuration in the repository's **.git/config** file under the **gitflow.*** namespace. The **--global**, **--system**, or **--file** options can be used to store configuration in alternate locations.

```
[gitflow]
    version = 1.0
    initialized = true
[gitflow "branch.main"]
    type = base
[gitflow "branch.develop"]
    type = base
    parent = main
    autoupdate = true
[gitflow "branch.feature"]
    type = topic
    parent = develop
    prefix = feature/
```

## EXIT STATUS

**0**
: Successful initialization

**1**
: Repository not found or not a git repository

**2**
: Repository already initialized (use config commands to modify)

**3**
: Invalid preset or configuration options

**6**
: A required precondition failed — for example, the repository has no commits and branch creation is requested but no git identity (**user.name** and **user.email**) is configured

## SEE ALSO

**git-flow**(1), **git-flow-config**(1), **gitflow-config**(5)

## NOTES

- **git-flow init** requires **--force** to reconfigure an already initialized repository in non-interactive mode
- In interactive mode without **--force**, users are prompted for confirmation before reconfiguring
- Existing branches are preserved during initialization
- When initializing a repository with no existing commits, **git-flow init** creates an empty initial commit to enable branch creation. No files are added to the working directory
- Creating that initial commit requires a configured git identity. When the repository has no commits and branch creation is requested, **git-flow init** verifies that both **user.name** and **user.email** are set (in local, global, or system config) before writing any configuration. If the identity is missing, init fails fast with an actionable error (exit status **6**) and leaves the repository untouched, so it can be re-run after configuring the identity. Repositories that already have commits, or runs with **--no-create-branches**, do not require an identity
- Compatible with repositories previously initialized with git-flow-avh
- Configuration scope options (**--local**, **--global**, **--system**, **--file**) only affect the **init** command. All other git-flow commands (start, finish, update, etc.) always read from merged config using Git's standard precedence (local > global > system)
- When checking initialization status without an explicit scope flag, git-flow checks merged config and reports which scope the configuration was found in
- When initialized via global or system config, attempting to initialize again without a scope flag will suggest using **--local** to create repository-specific config
