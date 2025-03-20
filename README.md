# Git Flow Next Implementation Guide

## 1. Project Objectives

### Primary Goal
Build "git-flow-next", a modern reimplementation of git-flow in Go that offers greater flexibility while maintaining backward compatibility with git-flow-avh.

### Success Criteria
- Feature parity with core git-flow-avh commands (init, start, finish)
- Successful execution of all test cases
- Proper handling of git-flow-avh configuration import
- Implementation of the new, more flexible configuration system

## 2. Technical Requirements

### Go Implementation
- Use Go 1.19 or newer
- Follow Go project standard directory structure
- Implement proper error handling with meaningful error messages
- Use Go modules for dependency management

### External Dependencies
- Use only the standard library and minimal external dependencies
- For Git operations, use a suitable Go Git library (e.g., go-git) or shell out to git commands
- For CLI interface, use a robust Go command-line library (e.g., cobra)

### Output Binary
- The resulting binary must be named `git-flow`
- Must be installable in user's PATH
- Should detect when run as a Git subcommand (`git flow`)

## 3. Implementation Sequence

### Phase 1: Core Framework
1. Set up project structure
2. Implement configuration loading/parsing system
3. Create Git operations wrapper
4. Build command-line interface skeleton

### Phase 2: Basic Commands
1. Implement `init` command
2. Implement dynamic topic branch command generation system
3. Implement generic topic branch `start` command
4. Implement generic topic branch `finish` command
5. Implement backward compatibility with git-flow-avh

### Phase 3: Testing & Refinement
1. Implement comprehensive test suite
2. Add configuration validation
3. Implement conflict resolution system
4. Create documentation

## 4. Code Organization

### Directory Structure
```
git-flow-next/
├── cmd/                    # Command implementation
│   ├── root.go             # Root command
│   ├── init.go             # Init command
│   ├── topicbranch.go      # Dynamic topic branch command generator
│   ├── start.go            # Generic start command implementation
│   └── finish.go           # Generic finish command implementation
├── config/                 # Configuration handling
│   ├── loader.go           # Load configuration
│   ├── writer.go           # Write configuration
│   ├── model.go            # Configuration data structures
│   └── compat.go           # Compatibility with git-flow-avh
├── git/                    # Git operations wrapper
│   ├── branch.go           # Branch operations
│   ├── config.go           # Git config operations
│   ├── merge.go            # Merge operations
│   └── repo.go             # Repository operations
├── model/                  # Data models
│   ├── branch.go           # Branch types
│   └── workflow.go         # Workflow definitions
├── util/                   # Utility functions
│   ├── validation.go       # Input validation
│   └── state.go            # State management
├── main.go                 # Application entry point
└── test/                   # Integration tests
```

### Module Responsibilities

#### Main Module
- Parse command-line arguments
- Dynamically discover and route to topic branch commands based on configuration
- Handle global flags
- Generate dynamic help text based on configured branch types

#### Config Package
- Read git-flow configuration from Git config
- Parse and validate configuration
- Import git-flow-avh configuration
- Write updated configuration

#### Git Package
- Provide abstraction over Git operations
- Handle branch creation, checkout, deletion
- Manage merges with different strategies
- Access and modify Git configuration

#### Model Package
- Define data structures for branch types
- Define workflow relationships
- Define merge strategies

#### Util Package
- Provide input validation
- Handle state persistence during operations
- Implement common utility functions

## 5. Configuration System Specification

### Format
The configuration uses Git's config system with the following structure:

```
[gitflow]
    version = 1.0

[gitflow.branch "main"]
    type = base
    parent = 
    upstreamStrategy = none
    downstreamStrategy = none
    autoUpdate = false

[gitflow.branch "develop"]
    type = base
    parent = main
    upstreamStrategy = merge
    downstreamStrategy = merge
    autoUpdate = true

[gitflow.branch "feature"]
    type = topic
    parent = develop
    startPoint = develop
    upstreamStrategy = rebase
    downstreamStrategy = squash
    prefix = feature/
```

### Configuration Loading
1. Read all gitflow config at once using: `git config --get-regexp "^gitflow\..*"`
2. Parse into structured in-memory representation
3. Cache configuration for the duration of the command execution

### Backward Compatibility
1. Check for existing git-flow-avh configuration
2. Convert from old format to new format during init or first use
3. Maintain compatible config entries for branch tracking

## 6. Command Structure and Dynamic Command Generation

### Dynamic Command Discovery
The application should dynamically generate commands based on the configured branch types:

1. During initialization, read the configuration to discover all topic branch types
2. For each topic branch type, create a corresponding command (e.g., `feature`, `release`, `hotfix`, `integration`)
3. Each topic branch command should have subcommands for operations (`start`, `finish`)
4. All topic branch commands should delegate to the generic implementation with their branch type as a parameter

### Command Mapping Example
```
git flow feature start my-feature
  |     |       |      └─ Branch name (passed to generic start)
  |     |       └─ Subcommand (mapped to generic start command)
  |     └─ Topic branch type (dynamically generated from config)
  └─ Main command
```

### Dynamic Help Generation
The application must dynamically generate help text based on configured topic branch types:
- Root help should list all available commands, including dynamically discovered topic branch types
- Each topic branch command should have appropriate help text describing its purpose based on configuration
- Example when a custom "integration" branch type is configured:
  ```
  usage: git flow <command> [<args>]
  
  Available commands:
    init         Initialize git-flow in a repository
    feature      Manage feature branches
    release      Manage release branches
    hotfix       Manage hotfix branches
    integration  Manage integration branches
    version      Show version information
    help         Show help for a specific command
  
  Run 'git flow <command> -h' for command-specific help.
  ```

## 7. Command Implementation Details

### Init Command
```go
// Pseudocode for Init Command
func InitCommand() {
    // Check if git-flow-avh config exists
    if GitFlowAVHConfigExists() {
        // Import and translate to new format
        oldConfig := ReadGitFlowAVHConfig()
        newConfig := TranslateConfig(oldConfig)
        WriteNewConfig(newConfig)
        PrintImportSuccessMessage()
    } else {
        // Prompt for configuration
        mainBranch := PromptForBranch("main branch", "main")
        developBranch := PromptForBranch("develop branch", "develop")
        featurePrefix := PromptForPrefix("feature prefix", "feature/")
        releasePrefix := PromptForPrefix("release prefix", "release/")
        hotfixPrefix := PromptForPrefix("hotfix prefix", "hotfix/")
        
        // Create and write config
        config := CreateDefaultConfig(mainBranch, developBranch, featurePrefix, releasePrefix, hotfixPrefix)
        WriteNewConfig(config)
        PrintConfigCreatedMessage()
    }
    
    // Mark repo as initialized
    MarkRepoInitialized()
}
```

### Dynamic Topic Branch Command Generation
```go
// Pseudocode for Dynamic Command Registration
func RegisterTopicBranchCommands(rootCmd *cobra.Command) {
    // Get all topic branch types from configuration
    config := LoadConfiguration()
    topicBranchTypes := GetTopicBranchTypes(config)
    
    // For each topic branch type, create a command
    for _, branchType := range topicBranchTypes {
        branchConfig := GetBranchTypeConfig(branchType)
        
        // Create command for this branch type
        branchCmd := &cobra.Command{
            Use:   branchType,
            Short: fmt.Sprintf("Manage %s branches", branchType),
            Long:  fmt.Sprintf("Manage %s branches - %s", branchType, branchConfig.Description),
        }
        
        // Add start subcommand
        startCmd := &cobra.Command{
            Use:   "start [name]",
            Short: fmt.Sprintf("Start a new %s branch", branchType),
            Args:  cobra.ExactArgs(1),
            Run: func(cmd *cobra.Command, args []string) {
                // Delegate to generic implementation
                StartCommand(branchType, args[0])
            },
        }
        branchCmd.AddCommand(startCmd)
        
        // Add finish subcommand
        finishCmd := &cobra.Command{
            Use:   "finish [name]",
            Short: fmt.Sprintf("Finish a %s branch", branchType),
            Args:  cobra.ExactArgs(1),
            Run: func(cmd *cobra.Command, args []string) {
                // Delegate to generic implementation
                FinishCommand(branchType, args[0])
            },
        }
        branchCmd.AddCommand(finishCmd)
        
        // Add custom flags if specified in configuration
        if branchConfig.CustomFlags != nil {
            AddCustomFlags(startCmd, finishCmd, branchConfig.CustomFlags)
        }
        
        // Add the branch command to the root command
        rootCmd.AddCommand(branchCmd)
    }
}

### Generic Topic Branch Start
```go
// Pseudocode for Generic Start Command
func StartCommand(branchType string, name string) {
    // Validate inputs
    if !IsValidBranchType(branchType) {
        ExitWithError("Invalid branch type")
    }
    
    if BranchExists(GetFullBranchName(branchType, name)) {
        ExitWithError("Branch already exists")
    }
    
    // Get configuration for branch type
    config := GetBranchTypeConfig(branchType)
    startPoint := config.StartPoint
    
    // Validate start point exists
    if !BranchExists(startPoint) {
        ExitWithError("Start point branch does not exist")
    }
    
    // Create branch
    fullBranchName := config.Prefix + name
    CreateBranch(fullBranchName, startPoint)
    
    // Checkout branch
    Checkout(fullBranchName)
    
    // Store parent relationship
    StoreBranchParent(fullBranchName, config.Parent)
    
    PrintSuccessMessage(branchType, name)
}
```

### Generic Topic Branch Finish
```go
// Pseudocode for Generic Finish Command
func FinishCommand(branchType string, name string) {
    // Validate inputs and state
    fullBranchName := GetFullBranchName(branchType, name)
    
    if !BranchExists(fullBranchName) {
        ExitWithError("Branch does not exist")
    }
    
    // Get configuration
    config := GetBranchTypeConfig(branchType)
    parentBranch := GetBranchParent(fullBranchName)
    mergeStrategy := config.UpstreamStrategy
    
    // Execute merge based on strategy
    success := false
    switch mergeStrategy {
    case "rebase":
        success = RebaseAndMerge(fullBranchName, parentBranch)
    case "squash":
        success = SquashMerge(fullBranchName, parentBranch)
    case "merge":
        success = StandardMerge(fullBranchName, parentBranch)
    default:
        ExitWithError("Invalid merge strategy")
    }
    
    if !success {
        // Store state for later resume
        StoreFinishState(branchType, name, parentBranch, mergeStrategy)
        ExitWithError("Merge conflicts detected. Resolve conflicts and run 'git flow finish --continue'")
    }
    
    // Update child branches if needed
    UpdateChildBranches(parentBranch)
    
    // Delete branch
    DeleteBranch(fullBranchName)
    
    PrintSuccessMessage(branchType, name)
}
```

## 7. Testing Requirements

### Unit Tests
Implement unit tests for all core functions:
- Configuration parsing and validation
- Git operation wrappers
- Command implementation logic
- Dynamic command generation and routing

### Integration Tests
1. **Init Command Tests**
   - Test init on new repository
   - Test init with existing git-flow-avh config

2. **Dynamic Command Generation Tests**
   - Test command discovery from configuration
   - Test help text generation with custom branch types
   - Test routing from dynamic commands to generic implementations

3. **Start Command Tests**
   - Test start with valid branch type and name
   - Test start with non-existent branch type
   - Test start when branch already exists
   - Test start with custom branch type from configuration

4. **Finish Command Tests**
   - Test finish with valid branch (no conflicts)
   - Test finish with merge conflicts
   - Test finish with different merge strategies
   - Test finish when branch has no commits
   - Test finish with custom branch type from configuration

### Test Coverage
- Aim for minimum 80% code coverage
- 100% coverage for critical paths (configuration, merge strategies)

## 8. Error Handling and Logging

### Error Handling Guidelines
1. Use meaningful error messages that suggest corrective action
2. Return appropriate exit codes for different error types
3. Preserve Git state on error when possible

### Logging Requirements
1. Log command execution and results
2. Provide verbose output option for debugging
3. Display progress for long-running operations

## 9. State Management

### Conflict Resolution State
Store state in `.git/gitflow/state/` directory:
```json
{
  "action": "finish",
  "branchType": "feature",
  "branchName": "example",
  "currentStep": "merge",
  "parentBranch": "develop",
  "mergeStrategy": "rebase",
  "remainingSteps": ["delete_branch"]
}
```

### State Resume
Implement logic to check for and resume interrupted operations

## 10. Future Enhancements (Post-Initial Version)

1. Additional commands (publish, pull, track)
2. Custom workflow definitions
3. Support for more complex branch relationships
4. Interactive conflict resolution
5. Integration with Git hosting platforms

## 11. Documentation Requirements

1. Generate command help text from code
2. Create README with installation and basic usage
3. Document configuration options and merge strategies
4. Include migration guide for git-flow users

### Installation

#### macOS
Using Homebrew:
```bash
brew install git-flow-next
```

Manual installation:
```bash
# For Intel Macs
curl -Lo git-flow.tar.gz https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-darwin-amd64.tar.gz
# For Apple Silicon Macs
curl -Lo git-flow.tar.gz https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-darwin-arm64.tar.gz

tar xzf git-flow.tar.gz
sudo mv git-flow /usr/local/bin/
rm git-flow.tar.gz
```

#### Linux
Using package managers (coming soon):
```bash
# For Ubuntu/Debian
sudo apt install git-flow-next

# For Fedora
sudo dnf install git-flow-next

# For Arch Linux
yay -S git-flow-next
```

Manual installation:
```bash
# For x86_64 systems
curl -Lo git-flow.tar.gz https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-linux-amd64.tar.gz
# For ARM64 systems
curl -Lo git-flow.tar.gz https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-linux-arm64.tar.gz
# For 32-bit systems
curl -Lo git-flow.tar.gz https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-linux-386.tar.gz

tar xzf git-flow.tar.gz
sudo mv git-flow /usr/local/bin/
rm git-flow.tar.gz
```

#### Windows
Using Scoop:
```powershell
scoop install git-flow-next
```

Manual installation:
1. Download the latest release for your system:
   - [64-bit Windows](https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-windows-amd64.zip)
   - [32-bit Windows](https://github.com/gittower/git-flow-next/releases/latest/download/git-flow-next-latest-windows-386.zip)
2. Extract the ZIP file
3. Move `git-flow.exe` to a directory in your PATH (e.g., `C:\Program Files\Git\cmd\`)

#### Verifying the Installation
After installation, verify that git-flow is installed correctly:
```bash
git flow version
```

#### Building from Source
If you have Go 1.19 or later installed, you can build from source:
```bash
git clone https://github.com/gittower/git-flow-next.git
cd git-flow-next
go build -o git-flow
```