# GIT-FLOW-INIT(1)

## NAME

git-flow-init - Initialize git-flow in a repository

## SYNOPSIS

**git-flow init** [**-f**|**--force**|**--no-force**] [**--preset**=*preset*] [**--custom**] [**--defaults**] [*options*]

## DESCRIPTION

Initialize git-flow configuration in the current Git repository. This command sets up the branch structure and configuration needed for git-flow operations.

**git-flow init** supports three initialization modes:

1. **Interactive Mode** (default) - Presents a menu to choose between presets or custom configuration
2. **Preset Mode** - Automatically applies a predefined workflow configuration  
3. **Custom Mode** - Sets up only the trunk branch and shows configuration commands

## OPTIONS

### Preset Options

**--preset**=*preset*
: Apply a predefined workflow preset. Valid values: **classic**, **github**, **gitlab**

**--custom**
: Enable custom configuration mode. Prompts for trunk branch and displays configuration commands.

**--defaults**, **-d**
: Use default branch naming conventions without prompting for customization.

**--no-create-branches**
: Don't create branches even if they don't exist in the repository.

### Initialization Control

**--force**, **-f**
: Force reinitialization of an already initialized repository. Existing git-flow configuration will be overwritten.

**--no-force**
: Don't allow reinitialization if already initialized (default behavior). Exits with error if git-flow is already configured.

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
: Override version tag prefix (default: v)

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

Reinitialize with different settings:
```bash
git flow init --force --preset=github
```

## CONFIGURATION

After initialization, git-flow stores configuration in **.git/config** under the **gitflow.*** namespace:

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
: Repository already initialized (use **--force** to reinitialize)

**3**
: Invalid preset or configuration options

## SEE ALSO

**git-flow**(1), **git-flow-config**(1), **gitflow-config**(5)

## NOTES

- **git-flow init** requires **--force** to reinitialize an already initialized repository
- Existing git-flow-avh configuration is automatically imported without requiring **--force**
- Existing branches are preserved during initialization
- Compatible with repositories previously initialized with git-flow-avh
- All configuration is stored locally in the repository