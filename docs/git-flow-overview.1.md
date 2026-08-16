# GIT-FLOW-OVERVIEW(1)

## NAME

git-flow-overview - Display repository workflow overview

## SYNOPSIS

**git-flow overview** [**--format**=*format*] [**--verbose**]

## DESCRIPTION

Display a comprehensive overview of the current repository's git-flow configuration, branch structure, and workflow status. This command provides a visual representation of your git-flow setup, active branches, and workflow health.

The overview command is useful for understanding the current state of a git-flow repository and identifying any configuration or workflow issues.

## OPTIONS

**--format**=*format*
: Output format. Valid values: **text** (default), **json**, **yaml**

**--verbose**, **-v**
: Show detailed information including configuration values and branch metadata

**--no-color**
: Disable colored output

## OUTPUT SECTIONS

### Configuration Summary
- **Workflow Type**: Detected preset or custom configuration
- **Branch Types**: Configured base and topic branch types  
- **Remote**: Configured remote repository

### Branch Structure
- **Base Branches**: Long-living branches with parent relationships
- **Topic Branch Types**: Configured topic branch templates
- **Active Branches**: Currently existing topic branches

Trunk branches, child base branches and topic branch types are each listed alphabetically by branch type name within their own group, so identical runs on an unchanged repository produce the same output.

### Workflow Status
- **Health**: Configuration validation status
- **Warnings**: Potential issues or misconfigurations
- **Statistics**: Branch counts and activity metrics

## EXAMPLES

### Basic Overview

```bash
git flow overview
```

Output:
```
Git-Flow Repository Overview
============================

Configuration:
  Workflow: Classic GitFlow
  Version: 1.0
  Remote: origin

Base Branches:
  • main (trunk)
  • develop (parent: main, auto-update: true)

Topic Branch Types:
  • feature (parent: develop, prefix: feature/)
  • release (parent: main, start: develop, tags: yes)
  • hotfix (parent: main, tags: yes)

Active Branches:
  • feature/user-auth (ahead: 3, behind: 0)
  • feature/api-docs (ahead: 1, behind: 2)
  • release/1.2.0 (ahead: 5, behind: 0)

Workflow Status: ✓ Healthy
```

### Verbose Overview

```bash
git flow overview --verbose
```

Shows additional details:
- Full configuration values
- Merge strategies for each branch type
- Git commit information for active branches
- Remote tracking status

### JSON Output

```bash
git flow overview --format=json
```

Outputs structured data suitable for tooling integration:
```json
{
  "configuration": {
    "workflow": "classic",
    "version": "1.0",
    "remote": "origin"
  },
  "baseBranches": [
    {
      "name": "main",
      "type": "base",
      "parent": null,
      "autoUpdate": false
    }
  ],
  "topicBranchTypes": [...],
  "activeBranches": [...],
  "status": "healthy"
}
```

## WORKFLOW DETECTION

The overview command automatically detects your workflow type:

**Classic GitFlow**
: main + develop + feature/release/hotfix branches

**GitHub Flow**
: main + feature branches only

**GitLab Flow**  
: production + staging + main + feature/hotfix branches

**Custom**
: Non-standard configuration

## HEALTH CHECKS

The overview performs several health checks:

### Configuration Validation
- **Valid branch relationships**: No circular dependencies
- **Existing parents**: All parent branches exist
- **Valid prefixes**: Branch prefixes are valid Git references

### Branch Status
- **Sync status**: How far ahead/behind branches are
- **Merge conflicts**: Potential merge conflicts between branches
- **Stale branches**: Old branches that might need cleanup

### Workflow Compliance
- **Naming conventions**: Branches follow configured prefixes
- **Branch relationships**: Branches have correct parent relationships

## STATUS INDICATORS

**✓ Healthy**
: No issues detected, workflow is properly configured

**⚠ Warning**  
: Minor issues that should be addressed

**✗ Error**
: Critical configuration problems that prevent proper operation

**? Unknown**
: Unable to determine status (repository not initialized)

## BRANCH INFORMATION

For each active branch, the overview shows:

- **Name**: Full branch name
- **Type**: Branch type (feature, release, etc.)
- **Status**: Ahead/behind commit counts relative to parent
- **Age**: How long since last commit
- **Author**: Last committer (in verbose mode)

## INTEGRATION

The overview command is designed for integration with other tools:

### CI/CD Integration
```bash
# Check workflow health in CI
if git flow overview --format=json | jq -r '.status' == "healthy"; then
  echo "Workflow is healthy"
else  
  echo "Workflow issues detected"
  exit 1
fi
```

### Shell Prompts
```bash
# Add workflow status to prompt
export PS1="$(git flow overview --format=json | jq -r '.workflow') $ "
```

### IDE Integration
Many editors can consume the JSON output to display workflow status.

## EXAMPLES BY WORKFLOW

### Classic GitFlow Overview
```
Configuration: Classic GitFlow
Base Branches:
  • main (trunk) 
  • develop (parent: main, auto-update: true)
Topic Types:
  • feature → develop (prefix: feature/)
  • release → main (start: develop, tags: yes)
  • hotfix → main (tags: yes)
```

### GitHub Flow Overview  
```
Configuration: GitHub Flow
Base Branches:
  • main (trunk)
Topic Types:
  • feature → main (prefix: feature/)
```

### GitLab Flow Overview
```
Configuration: GitLab Flow  
Base Branches:
  • production (trunk)
  • staging (parent: production)
  • main (parent: staging)
Topic Types:
  • feature → main (prefix: feature/)
  • hotfix → production (tags: yes)
```

## EXIT STATUS

**0**
: Overview displayed successfully

**1**
: Repository not found or not a git repository

**2**
: Repository not initialized with git-flow

**3**
: Configuration errors prevent overview generation

## SEE ALSO

**git-flow**(1), **git-flow-config**(1), **git-flow-init**(1), **git-status**(1)

## NOTES

- Overview reflects current repository state, not historical information
- JSON/YAML formats are stable and suitable for automation
- Health checks are recommendations, not requirements
- Use **--verbose** for troubleshooting configuration issues
- The overview command is read-only and makes no changes to the repository