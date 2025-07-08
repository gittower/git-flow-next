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

## Configuration

git-flow-next supports various configuration options through Git config. Here are some key options:

### Fetch Before Finish

Control whether to fetch from remote before finishing topic branches:

```bash
# Enable fetch for feature branches
git config gitflow.feature.finish.fetch true

# Enable fetch for release branches  
git config gitflow.release.finish.fetch true

# Enable fetch for hotfix branches
git config gitflow.hotfix.finish.fetch true
```

You can also use command line options to override the configuration:
- `--fetch` - Fetch from remote before finishing branch
- `--no-fetch` - Don't fetch from remote before finishing branch

Example:
```bash
git flow feature finish my-feature --fetch
```

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