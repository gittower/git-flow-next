# GIT-FLOW-INIT(1)

## NAME

git-flow-init - Initialize git-flow in a repository

## SYNOPSIS

**git-flow init** [**-f**|**--force**] [**--preset**=*preset*] [**--custom**] [**--defaults**] [**--local**|**--global**|**--system**|**--file**=*path*] [*options*]

## DESCRIPTION

Initialize git-flow configuration in the current Git repository. This command sets up the branch structure and configuration needed for git-flow operations.

**git-flow init** supports three initialization modes:

1. **Interactive Mode** (default) - Presents a menu to choose between presets or custom configuration
2. **Preset Mode** - Automatically applies a predefined workflow configuration  
3. **Custom Mode** - Sets up only the trunk branch and shows configuration commands

## OPTIONS

### General Options

**-f**, **--force**
: Force reconfiguration of git-flow even if already initialized. Without this option, **git flow init** will fail if configuration already exists (in non-interactive mode) or prompt for confirmation (in interactive mode).

### Preset Options

**--preset**=*preset*
: Apply a predefined workflow preset. Valid values: **classic**, **github**, **gitlab**

**--custom**
: Enable custom configuration mode. Prompts for trunk branch and displays configuration commands.

**--defaults**, **-d**
: Use default branch naming conventions without prompting for customization.

**--no-create-branches**
: Don't create branches even if they don't exist in the repository.

### Configuration Scope Options

These options control where git-flow configuration is stored. Only one scope option may be specified at a time. When no scope option is given, git-flow reads from merged config (local > global > system precedence) and writes to local config.

**--local**
: Read and write configuration only in the repository's **.git/config** file. This is the default for writes when no scope option is specified.

**--global**
: Read and write configuration in the user's global **~/.gitconfig** file. Useful for setting up defaults that apply to all repositories.

**--system**
: Read and write configuration in the system-wide **/etc/gitconfig** file. Typically requires administrator privileges.

**--file**=*path*
: Read and write configuration in the specified file. The parent directory must exist and be writable. Paths may be absolute or relative to the current working directory. Useful for managing shared configuration files.

### Branch Name Overrides

**--main**=*name*
: Override main branch name (default: main)

**--develop**=*name*  
: Override develop branch name (default: develop)

**--production**=*name*
: Override production branch name for GitLab flow (default: production)

**--staging**=*name*
: Override staging branch name for GitLab flow (default: staging)

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

When run without options, **git-flow init** presents an interactive menu:

```
? Choose initialization method:
  ❯ Use preset workflow
    Custom configuration

? Choose a preset:
  ❯ Classic GitFlow
    GitHub Flow  
    GitLab Flow
```

After preset selection, you can customize branch names and prefixes.

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

## SEE ALSO

**git-flow**(1), **git-flow-config**(1), **gitflow-config**(5)

## NOTES

- **git-flow init** requires **--force** to reconfigure an already initialized repository in non-interactive mode
- In interactive mode without **--force**, users are prompted for confirmation before reconfiguring
- Existing branches are preserved during initialization
- When initializing a repository with no existing commits, **git-flow init** creates an empty initial commit to enable branch creation. No files are added to the working directory
- Compatible with repositories previously initialized with git-flow-avh
- Configuration scope options (**--local**, **--global**, **--system**, **--file**) only affect the **init** command. All other git-flow commands (start, finish, update, etc.) always read from merged config using Git's standard precedence (local > global > system)
- When checking initialization status without an explicit scope flag, git-flow checks merged config and reports which scope the configuration was found in
- When initialized via global or system config, attempting to initialize again without a scope flag will suggest using **--local** to create repository-specific config
