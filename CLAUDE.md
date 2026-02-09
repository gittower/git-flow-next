# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

git-flow-next is a modern Go implementation of the git-flow branching model. It's a CLI tool that helps manage Git workflows with feature, release, and hotfix branches.

**GitHub Repository**: [gittower/git-flow-next](https://github.com/gittower/git-flow-next)

## Development Commands

### Version Updates

When updating version information, ensure both locations are updated:

- `version/version.go` - Core version constant used by the application
- `cmd/version.go` - Version variable used by the version command

Both files must have matching version numbers to maintain consistency.

### Building
```bash
go build -o git-flow main.go              # Build local binary
./scripts/build.sh                        # Multi-platform build script
./scripts/build.sh v1.0.0                 # Build with specific version
```

### Testing
```bash
go test ./...                             # Run all tests
go test ./test/cmd/                       # Run command tests
go test ./test/internal/                  # Run internal package tests
go test -v ./test/cmd/init_test.go        # Run specific test file
```

### Running
```bash
go run main.go                            # Run directly
./git-flow                                # Run built binary
```

## Additional Documentation

For comprehensive development information, see:
- **[CODE_REFERENCE.md](CODE_REFERENCE.md)** - Quick codebase reference and navigation guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture and design overview
- **[CODING_GUIDELINES.md](CODING_GUIDELINES.md)** - Coding standards and conventions
- **[CONFIGURATION.md](CONFIGURATION.md)** - Complete configuration reference and examples
- **[TESTING_GUIDELINES.md](TESTING_GUIDELINES.md)** - Testing methodology and practices
- **[COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md)** - Commit message standards and best practices
- **[RELEASING.md](RELEASING.md)** - Release process and versioning
- **[.github/copilot-instructions.md](.github/copilot-instructions.md)** - GitHub Copilot context and patterns

## Architecture

### Command Structure
- **Root Command**: `cmd/root.go` - Main CLI setup with Cobra
- **Dynamic Commands**: `cmd/topicbranch.go` - Registers branch type commands based on config
- **Core Commands**: `cmd/init.go`, `cmd/start.go`, `cmd/finish.go`, etc.

### Key Internal Packages
- **config**: Git configuration management, branch type definitions
- **git**: Git command wrapper with error handling
- **mergestate**: Merge conflict state persistence
- **errors**: Custom error types and exit codes
- **util**: Validation and utility functions

### Configuration System
- Branch types defined in Git config under `gitflow.*`
- Each branch type has: prefix, parent, start point, merge strategies
- Supports custom branch types beyond standard feature/release/hotfix
- Configuration imported from git-flow-avh for compatibility

### Git Operations
- All Git operations go through `internal/git/repo.go`
- Handles conflict detection and resolution
- Supports different merge strategies (merge, rebase, squash)
- Configurable fetch behavior before operations

## Testing Architecture

### Test Structure
- Tests mirror source structure in `test/` directory
- `testutil/` contains Git repository helpers and test utilities
- Integration tests use temporary Git repositories
- Command tests verify CLI behavior and error handling

### Test Utilities
- `testutil.CreateTestRepo()` - Creates temporary Git repository
- `testutil.InitGitFlow()` - Initializes git-flow in test repo
- Git operation mocks for unit tests

### Testing Best Practices
- **Use git-flow defaults**: Initialize test repos with `git flow init --defaults`
- **Use feature branches**: Feature branches for general topic branch testing
- **Modify config when needed**: Use `git config` for specific test scenarios

## Default Configuration

git-flow-next provides sensible defaults that work for most teams:

### Branch Structure
- **main/master**: Production releases
- **develop**: Integration branch (auto-updated from main)
- **feature/**: New features (parent: develop)
- **release/**: Release preparation (parent: main, starts from develop)
- **hotfix/**: Emergency fixes (parent: main)

### Default Merge Strategies
- **Feature finish**: `merge` into develop
- **Release finish**: `merge` into main (then auto-update develop)
- **Hotfix finish**: `merge` into main (then auto-update develop)
- **Feature update**: `rebase` from develop
- **Release/Hotfix update**: `merge`/`rebase` from main

### Default Tag Settings
- **Feature**: No tags created
- **Release/Hotfix**: Tags created on finish

## Configuration Examples

### Branch Configuration
```bash
# Custom branch prefixes
git config gitflow.branch.feature.prefix "feat/"
git config gitflow.branch.release.prefix "rel/"

# Merge strategies (upstream - finish operations)
git config gitflow.feature.finish.merge rebase
git config gitflow.release.finish.merge squash

# Merge strategies (downstream - update operations)
git config gitflow.feature.downstreamStrategy rebase
git config gitflow.release.downstreamStrategy merge

# Base branch relationships
git config gitflow.branch.develop.parent main
git config gitflow.branch.develop.autoUpdate true

# Fetch behavior
git config gitflow.feature.finish.fetch true
```

### Branch Types
- **feature**: Topic branches for new features
- **release**: Release preparation branches
- **hotfix**: Emergency fixes for production
- **support**: Long-term support branches
- Custom types can be added via configuration

## Documentation Requirements

**⚠️ MANDATORY**: Always update documentation when modifying commands or configuration:

- **New commands**: Create manpage documentation in `docs/`
- **Modified commands**: Update existing command documentation  
- **New/changed options**: Update relevant manpage files
- **Configuration changes**: Update `docs/gitflow-config.5.md`
- **Behavior changes**: Update descriptions and examples

All documentation follows Unix manpage standards. See `docs/README.md` for details.

## Development Philosophy

**CRITICAL**: Follow a **pragmatic, anti-over-engineering approach**:
- Use patterns wisely, but don't let them dictate your code
- Complex functions with many parameters are acceptable when they reflect real problems
- Encapsulate meaningfully, not artificially
- Option structs are good when they group related concepts (like `TagOptions`)
- Prefer explicit over implicit, readable over clever

## Code Conventions

### Error Handling
- Use custom error types from `internal/errors`
- Return specific exit codes for different error conditions
- Provide clear error messages with context

### Git Integration
- Always use `internal/git/repo.go` for Git operations
- Handle merge conflicts gracefully
- Check for uncommitted changes before operations

### Command Implementation
- Use Cobra command structure consistently
- Implement both long and short flag variants
- Add command examples and usage information
- Validate inputs before executing operations
- **Follow three-layer configuration precedence**: Layer 1 branch type definition (identity & process characteristics — what the type *is*) → Layer 2 command-specific config (operational settings — how commands *execute*) → Layer 3 CLI flags (always win). Layer 1 is for essential branch-type properties only; many options intentionally skip it (e.g., publish push-options, sign, keep).

## Commit Guidelines

Follow the project's commit message standards as defined in **[COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md)**. Use structured format with appropriate types (feat, fix, refactor, test, docs) and clear, imperative subjects ≤50 characters.

## Branching

When needing to create branches for development work, use git-flow commands:

```bash
git flow feature start <feature-name>   # Create a new feature branch
```

## Development Notes

### Compatibility
- Maintains compatibility with existing git-flow repositories
- Imports configuration from git-flow-avh
- Uses same branch naming conventions by default

### Performance Considerations
- Minimize Git operations in large repositories
- Cache configuration lookups
- Use efficient conflict detection
