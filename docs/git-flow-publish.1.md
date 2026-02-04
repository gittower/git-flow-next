# GIT-FLOW-PUBLISH(1)

## NAME

git-flow-publish - Publish a topic branch to the remote repository

## SYNOPSIS

**git-flow** *topic* **publish** [*name*] [**-o** *option*]... [**--no-push-option**]

## DESCRIPTION

Publishes a local topic branch to the configured remote repository. This command pushes the branch and sets up tracking between the local and remote branches.

After publishing, other team members can track this branch using `git flow <type> track`.

If no name is provided, the current branch is published (if it matches the specified branch type).

The command will:

1. Verify the local branch exists
2. Fetch from remote to check current state
3. Check if the remote branch already exists
4. Push the branch to the remote with tracking enabled

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: Optional. The name of the branch to publish. If omitted, publishes the current branch. Can be specified with or without the branch prefix.

## OPTIONS

**-o** *option*, **--push-option**=*option*
: Transmit the given string to the server during push. This option can be repeated multiple times to send multiple push options. Push options are used by hosting platforms like GitLab, Gitea, and Gerrit for server-side behavior (e.g., creating merge requests, skipping CI). Config defaults and CLI options are combined additively.

**--no-push-option**
: Suppress all push options, including any configured defaults via `gitflow.<branchtype>.publish.push-option`. Use this when you want to publish without triggering any server-side behaviors that would be activated by configured push options.

## EXAMPLES

### Basic Usage

Publish a feature branch by name:
```bash
git flow feature publish user-authentication
```

Publish the current feature branch:
```bash
git checkout feature/my-feature
git flow feature publish
```

### Other Branch Types

Publish a release branch:
```bash
git flow release publish 1.0.0
```

Publish a hotfix branch:
```bash
git flow hotfix publish 1.0.1
```

### Using Full Branch Name

Publish using the full branch name (with prefix):
```bash
git flow feature publish feature/my-feature
```

### Using Push Options

Publish with a push option to skip CI:
```bash
git flow feature publish my-feature -o ci.skip
```

Publish and create a GitLab merge request:
```bash
git flow feature publish my-feature -o merge_request.create -o merge_request.target=main
```

Publish with multiple push options for Gerrit:
```bash
git flow feature publish my-feature -o %submit -o %topic=my-topic
```

Publish without any push options (override config defaults):
```bash
git flow feature publish my-feature --no-push-option
```

## CONFIGURATION

**gitflow.origin**
: Specifies the remote repository to push to. Defaults to "origin" if not set.

### Setting Custom Remote
```bash
# Use 'upstream' instead of 'origin'
git config gitflow.origin upstream
```

**gitflow.*branchtype*.publish.push-option**
: Default push options to transmit when publishing branches of this type. This is a multi-value key; multiple options can be configured by using `git config --add`. CLI options are combined with config defaults (additive). Use `--no-push-option` to suppress these defaults.

### Configuring Default Push Options
```bash
# Always skip CI when publishing feature branches
git config gitflow.feature.publish.push-option "ci.skip"

# Create a merge request when publishing release branches
git config gitflow.release.publish.push-option "merge_request.create"
git config --add gitflow.release.publish.push-option "merge_request.target=main"
```

## WORKFLOW INTEGRATION

### Team Collaboration Workflow

The `publish` and `track` commands are complementary for team collaboration:

```bash
# Developer A: Start and publish a feature
git flow feature start my-feature
# ... make some commits ...
git flow feature publish my-feature

# Developer B: Track the published feature
git flow feature track my-feature
```

### Publishing Before Finish

When working with remote repositories, publish your branch before finishing to share your work:

```bash
# Start feature locally
git flow feature start collaborative-feature

# Work on the feature
# ... make commits ...

# Publish to share with team
git flow feature publish collaborative-feature

# Later, finish the feature
git flow feature finish collaborative-feature
```

## ERROR HANDLING

### Branch Not Found Locally

If the specified branch doesn't exist locally:
```
Error: local branch 'feature/non-existent' does not exist
```

### Branch Already Exists on Remote

If the branch already exists on the remote:
```
Error: branch 'feature/my-feature' already exists on remote 'origin'
```

Use `git push` directly if you need to update an existing remote branch.

### Wrong Branch Type

If publishing current branch with wrong type specified:
```
Error: current branch 'feature/my-feature' is not a release branch
```

## EXIT STATUS

**0**
: Successful execution.

**1**
: git-flow is not initialized.

**2**
: Invalid input (branch type mismatch, invalid branch name).

**3**
: Git operation failed (push failed, connectivity issues, etc.).

**4**
: Branch already exists on remote.

**5**
: Branch not found locally.

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-finish**(1), **gitflow-config**(5)

## NOTES

- Publishing sets up a tracking relationship between local and remote branches
- Use `git push` for subsequent updates to the remote branch after publishing
- If the remote branch already exists, the publish will fail to prevent accidental overwrites
- After publishing, team members can track the branch with `git flow <type> track <name>`
- The fetch operation before publishing may show warnings if the remote is unreachable, but this won't prevent the publish if the remote branch doesn't exist
