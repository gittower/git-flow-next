# git-flow-next Documentation

This directory contains comprehensive manpage-style documentation for all git-flow-next commands and configuration options.

## Documentation Structure

### Command Documentation (Section 1)
- **git-flow.1.md** - Main git-flow command overview and global options
- **git-flow-init.1.md** - Repository initialization and workflow setup
- **git-flow-config.1.md** - Configuration management commands  
- **git-flow-integrate.1.md** - Integrate a base branch into its parent
- **git-flow-worktree.1.md** - Worktree management for branches
- **git-flow-feature.1.md** - Feature branch management
- **git-flow-release.1.md** - Release branch management
- **git-flow-hotfix.1.md** - Hotfix branch management
- **git-flow-overview.1.md** - Repository workflow overview
- **git-flow-completion.1.md** - Shell completion script generation
- **git-flow-shell-init.1.md** - Shell integration script for directory switching

### Configuration Documentation (Section 5)
- **gitflow-config.5.md** - Complete configuration reference and examples

## Documentation Standards

All documentation follows Unix manpage conventions:

### Structure
- **NAME** - Brief command description
- **SYNOPSIS** - Command syntax and options
- **DESCRIPTION** - Detailed explanation of functionality
- **OPTIONS** - Complete flag and parameter reference
- **EXAMPLES** - Practical usage examples
- **EXIT STATUS** - Return codes and meanings
- **SEE ALSO** - Related commands and references
- **NOTES** - Important considerations and tips

### Formatting
- **Bold** for command names, options, and important terms
- *Italics* for parameters, placeholders, and emphasis  
- `Code blocks` for configuration examples and commands
- Clear section hierarchies with proper heading levels

## Workflow-Specific Documentation

The documentation covers all supported workflows:

### Classic GitFlow
Traditional git-flow with main, develop, feature/, release/, and hotfix/ branches.

### GitHub Flow  
Simplified workflow with main and feature/ branches only.

### GitLab Flow
Multi-environment workflow with production, staging, main, feature/, and hotfix/ branches.

### Custom Workflows
Fully customizable branch configurations through the config system.

## Dynamic Commands

git-flow-next generates commands dynamically based on configuration. The documentation covers:

- **Standard topic types**: feature, release, hotfix, support
- **Custom topic types**: Any user-defined topic branch configurations
- **Shorthand commands**: Context-aware shortcuts that work with any topic branch type

## Configuration Reference

The configuration system documentation includes:

- **Three-layer hierarchy**: Branch defaults → Command overrides → CLI flags
- **Complete option reference**: All gitflow.* configuration keys
- **Merge strategy guide**: none, merge, rebase, squash strategies
- **git-flow-avh compatibility**: Automatic translation of legacy configurations
- **Workflow examples**: Complete configuration for each supported workflow

## Integration Examples

Documentation includes examples for:

- **CI/CD integration** - Using JSON output for automation
- **IDE integration** - Consuming structured workflow data
- **Shell integration** - Adding workflow status to prompts
- **Git hooks** - Customizing workflow behavior

## Keeping Documentation Updated

**⚠️ IMPORTANT**: Documentation must be updated whenever commands, options, or behavior changes.

See **CODING_GUIDELINES.md** for specific requirements about maintaining documentation currency.

## Viewing Documentation

### As Manpages (Recommended)
```bash
# Install pandoc for best rendering
brew install pandoc

# View as formatted manpage
pandoc docs/git-flow.1.md | man -l -

# Or use a manpage viewer
man docs/git-flow.1.md
```

### As Markdown
```bash
# View in terminal
cat docs/git-flow.1.md | less

# View in browser (with Markdown viewer)
open docs/git-flow.1.md
```

### Generate HTML
```bash
# Convert to HTML for web viewing
pandoc docs/git-flow.1.md -o docs/git-flow.1.html
```

## Documentation Development

When adding new commands or changing existing ones:

1. **Update relevant manpages** - Modify existing documentation
2. **Add new manpages** - Create new .1.md files for new commands
3. **Update cross-references** - Ensure SEE ALSO sections are current  
4. **Test examples** - Verify all examples work with current implementation
5. **Update this README** - Keep the overview current

## Contributing

When contributing documentation:

- Follow existing formatting and structure conventions
- Include practical examples for all major use cases
- Cross-reference related commands and concepts
- Test all command examples for accuracy
- Consider different skill levels (beginner to advanced)

## Online Documentation

The authoritative documentation is maintained in this repository. Online versions may be generated from these source files but should not be edited directly.

For the most current documentation, always refer to the files in this directory.