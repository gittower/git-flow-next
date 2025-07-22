# git-flow-next

A modern, maintainable implementation of the git-flow branching model, written in Go.

## About

git-flow-next is a modern reimplementation of the popular git-flow branching model. It's built with Go, focusing on reliability, extensibility, and developer experience.

## Why This Project?

This project is maintained by the team behind [Tower](https://www.git-tower.com), one of the most popular Git clients for Mac and Windows. Having integrated git-flow into Tower over many years, we've gained deep insights into its strengths and areas for improvement.

As developers of version control tools, we're passionate about creating better developer experiences. While the original git-flow has served the community well, we saw an opportunity to build a more modern implementation that:

- Is written in Go for better maintainability and performance
- Provides a more robust and reliable experience
- Offers better error handling and conflict resolution
- Supports modern Git workflows and practices
- Maintains compatibility with existing git-flow setups

Our goal is to contribute back to the developer community with tools that make version control workflows more efficient and enjoyable.

## Features

- **Modern Implementation**: Written in Go with focus on reliability and maintainability
- **Improved Conflict Resolution**: Better handling of merge conflicts and edge cases
- **Flexible Configuration**: Customizable branch naming and merge strategies
- **Compatibility**: Works with existing git-flow repositories
- **Enhanced Error Handling**: Clear error messages and recovery options
- **Performance**: Fast and efficient operations

## Installation

### Homebrew (macOS and Linux)

```bash
brew install gittower/formula/git-flow-next
```

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/gittower/git-flow-next/releases)
2. Extract the binary to a location in your PATH
3. Make it executable: `chmod +x /path/to/git-flow`

## Quick Start

1. Initialize git-flow in your repository:
   ```bash
   git flow init
   ```

2. Start a new feature:
   ```bash
   git flow feature start my-feature
   ```

3. Finish the feature:
   ```bash
   git flow feature finish my-feature
   ```

## Shorthand Commands

git-flow-next provides convenient shorthand commands that automatically detect your current topic branch and execute the appropriate action. These aliases work similar to git-flow-avh and eliminate the need to specify the branch type manually.

### Available Shorthands

| Shorthand | Full Command | Description |
|-----------|--------------|-------------|
| `git flow delete` | `git flow <type> delete <name>` | Delete the current topic branch |
| `git flow rebase` | `git flow <type> update --rebase` | Rebase the current topic branch *(planned)* |
| `git flow update` | `git flow <type> update` | Update the current topic branch |
| `git flow rename` | `git flow <type> rename <name>` | Rename the current topic branch |
| `git flow publish` | `git flow <type> publish` | Publish the current topic branch *(planned)* |
| `git flow finish` | `git flow <type> finish` | Finish the current topic branch |

### How It Works

When you use a shorthand command, git-flow-next:

1. **Detects your current branch** - Checks which branch you're currently on
2. **Identifies the branch type** - Determines if it's a feature, release, hotfix, or support branch based on configured prefixes
3. **Executes the full command** - Runs the corresponding full command with the detected type and branch name

### Examples

```bash
# On a feature branch
git checkout feature/my-awesome-feature
git flow finish  # Executes: git flow feature finish my-awesome-feature

# On a release branch  
git checkout release/v1.2.0
git flow publish  # Executes: git flow release publish v1.2.0

# On a hotfix branch
git checkout hotfix/critical-bug
git flow finish  # Executes: git flow hotfix finish critical-bug
```

### Branch Detection

The shorthand commands automatically detect topic branches based on your git-flow configuration:

- **Feature branches**: `feature/`, `features/`, `feat/`
- **Release branches**: `release/`, `releases/`, `rel/`
- **Hotfix branches**: `hotfix/`, `hotfixes/`, `hf/`
- **Support branches**: `support/`, `supports/`, `sup/`

### Error Handling

- **Non-topic branches**: If you're not on a topic branch, you'll get a clear error message
- **Ambiguous branches**: If a branch name could be interpreted multiple ways, you'll be prompted to use the explicit command
- **Missing branches**: If the branch doesn't exist, appropriate error messages are shown

### Command Options

All options and flags are passed through to the underlying commands:

```bash
# Options work exactly like the full commands
git flow finish --keep --tag  # Keeps the branch and creates a tag
git flow update --rebase      # Forces rebase strategy for update
git flow delete --force       # Force deletes the branch
```

### Supported Branch Types

The shorthand commands work with all standard git-flow branch types:

- **Feature branches**: For new features and enhancements
- **Release branches**: For preparing new releases
- **Hotfix branches**: For critical bug fixes
- **Support branches**: For maintaining older versions

### Planned Features

Some shorthand commands are currently planned for future releases:

- **`git flow rebase`**: Will be implemented as an alias to `git flow <type> update --rebase`
- **`git flow publish`**: Will be implemented as an alias to `git flow <type> publish`

These commands currently show "not implemented" messages when used.

## Documentation

For detailed documentation, please visit our [documentation site](https://github.com/gittower/git-flow-next/wiki).

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on how to get involved.

## Development

For information about the project's architecture and how to set up a development environment, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

This project is licensed under the BSD 2-Clause License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project builds upon the work of:
- Vincent Driessen's [original git-flow](https://nvie.com/posts/a-successful-git-branching-model/)
- Peter van der Does' [git-flow (AVH Edition)](https://github.com/petervanderdoes/gitflow-avh)

## About Tower

git-flow-next is maintained by the team behind [Tower](https://www.git-tower.com), the popular Git client for Mac and Windows. With over a decade of experience in Git tooling and version control, we're committed to creating high-quality developer tools that make working with Git more efficient and enjoyable.