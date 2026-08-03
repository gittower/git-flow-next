# git-flow-next Documentation Index

Comprehensive manpage-style documentation for git-flow-next commands and configuration.

## Quick Reference

| Command | Purpose | Documentation |
|---------|---------|---------------|
| **git-flow** | Main command overview | [git-flow(1)](git-flow.1.md) |
| **git-flow init** | Initialize git-flow | [git-flow-init(1)](git-flow-init.1.md) |
| **git-flow config** | Manage configuration | [git-flow-config(1)](git-flow-config.1.md) |
| **git-flow overview** | Repository status | [git-flow-overview(1)](git-flow-overview.1.md) |
| **git-flow integrate** | Integrate a base branch into its parent | [git-flow-integrate(1)](git-flow-integrate.1.md) |
| **git-flow completion** | Generate shell completion scripts | [git-flow-completion(1)](git-flow-completion.1.md) |

## Topic Branch Commands

| Command | Purpose | Documentation |
|---------|---------|---------------|
| **git-flow \<topic\> start** | Create topic branches | [git-flow-start(1)](git-flow-start.1.md) |
| **git-flow \<topic\> finish** | Complete topic branches | [git-flow-finish(1)](git-flow-finish.1.md) |
| **git-flow \<topic\> list** | List topic branches | [git-flow-list(1)](git-flow-list.1.md) |
| **git-flow \<topic\> update** | Update topic branches | [git-flow-update(1)](git-flow-update.1.md) |
| **git-flow \<topic\> delete** | Delete topic branches | [git-flow-delete(1)](git-flow-delete.1.md) |
| **git-flow \<topic\> rename** | Rename topic branches | [git-flow-rename(1)](git-flow-rename.1.md) |
| **git-flow \<topic\> checkout** | Switch to topic branches | [git-flow-checkout(1)](git-flow-checkout.1.md) |

## Configuration Reference

| Topic | Documentation |
|-------|---------------|
| **Configuration System** | [gitflow-config(5)](gitflow-config.5.md) |
| **git-flow-avh Compatibility** | [gitflow-config(5)#git-flow-avh-compatibility](gitflow-config.5.md#git-flow-avh-compatibility) |

## Getting Started

1. **[Initialize git-flow](git-flow-init.1.md)** - Set up your workflow
2. **[Configuration guide](git-flow-config.1.md)** - Customize your setup  
3. **[Topic branch workflow](git-flow-start.1.md)** - Basic development workflow
4. **[Repository overview](git-flow-overview.1.md)** - Monitor your setup

## Workflow Guides

- **Classic GitFlow** - Traditional git-flow workflow
- **GitHub Flow** - Simplified continuous deployment workflow
- **GitLab Flow** - Multi-environment staged workflow  
- **Custom Workflows** - Build your own with config commands

## Advanced Topics

- **[Three-layer configuration](gitflow-config.5.md#configuration-hierarchy)** - Understanding precedence
- **[Merge strategies](gitflow-config.5.md#merge-strategy-reference)** - Choose the right strategy
- **[Command overrides](gitflow-config.5.md#command-overrides)** - Fine-tune behavior
- **[Dynamic commands](git-flow.1.md#topic-branch-commands)** - Auto-generated commands

## Integration

- **CI/CD Integration** - Use JSON output for automation
- **IDE Integration** - Consume workflow data  
- **Shell Integration** - Add status to prompts
- **Git Hooks** - Customize workflow behavior

All documentation follows Unix manpage conventions and includes practical examples.