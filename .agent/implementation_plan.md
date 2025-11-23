# Tidy CLI - Cobra & Bubble Tea Integration Plan

## Project Overview
Transform the `tidy` package manager into a feature-rich CLI tool using:
- **Cobra**: For command-line argument parsing and command structure
- **Bubble Tea**: For interactive TUI when no command is provided

## Current State Analysis
- ✅ Core functionality exists: grep, install, resolve, registry interaction
- ✅ Basic package.json reading/writing
- ✅ Concurrent installation with worker pools
- ✅ Dev/prod dependency differentiation via `deps.json`
- ❌ No CLI argument parsing
- ❌ No interactive UI
- ❌ Hardcoded behavior (always scans and installs)

## Architecture Design

### 1. Command Structure
```
tidy                          # Interactive TUI (Bubble Tea)
├── install [packages...]     # Install packages
│   ├── --bun                # Use Bun package manager
│   ├── --pnpm               # Use pnpm package manager
│   ├── --npm                # Use npm package manager
│   └── (default)            # Use tidy's own installer
├── run <script>             # Run package.json scripts (dev, build, etc.)
│   └── (alias: tidy dev, tidy build, etc.)
├── add <packages...>        # Add packages (alias for install)
│   ├── -D, --dev            # Add as dev dependency
│   └── -g, --grep           # Grep codebase and add found packages
└── version                  # Show version info
```

### 2. File Structure
```
tidy/
├── main.go                   # Entry point - initializes Cobra root
├── go.mod                    # Add cobra & bubbletea dependencies
├── cmd/
│   ├── root.go              # Root command + TUI launcher
│   ├── install.go           # Install command
│   ├── run.go               # Run script command
│   └── add.go               # Add command (with -g flag)
├── internal/
│   ├── grep.go              # [EXISTING] Package detection
│   ├── install.go           # [EXISTING] Installation logic
│   ├── read_json.go         # [EXISTING] JSON operations
│   ├── registery.go         # [EXISTING] Registry interaction
│   ├── resolve.go           # [EXISTING] Dependency resolution
│   ├── installer.go         # [NEW] Package manager abstraction
│   └── runner.go            # [NEW] Script runner
├── ui/
│   ├── model.go             # Bubble Tea model
│   ├── commands.go          # Bubble Tea commands
│   ├── update.go            # Update logic
│   └── view.go              # View rendering
└── deps.json                # [EXISTING] Prod/dev package list
```

## Implementation Phases

### Phase 1: Project Setup & Dependencies
**Goal**: Add required dependencies and set up basic structure

#### Tasks:
1. **Update go.mod**
   ```bash
   go get github.com/spf13/cobra@latest
   go get github.com/charmbracelet/bubbletea@latest
   go get github.com/charmbracelet/lipgloss@latest  # For styling
   go get github.com/charmbracelet/bubbles@latest   # For components
   ```

2. **Create directory structure**
   - Create `cmd/` directory
   - Create `ui/` directory

3. **Version management**
   - Add version constant
   - Implement version command

**Deliverables**:
- Updated `go.mod` with all dependencies
- Directory structure created
- Version info accessible

---

### Phase 2: Cobra Command Structure
**Goal**: Implement CLI command parsing with Cobra

#### Tasks:

1. **Root Command (`cmd/root.go`)**
   - Initialize Cobra root command
   - If no subcommand provided → launch Bubble Tea TUI
   - Add persistent flags (--verbose, --quiet, etc.)
   - Set up command hierarchy

2. **Install Command (`cmd/install.go`)**
   ```go
   tidy install [packages...]
   Flags:
   - --bun      : Use Bun
   - --pnpm     : Use pnpm  
   - --npm      : Use npm
   - (default)  : Use tidy's installer
   ```
   - Parse package names from args
   - Detect package manager flag
   - Call appropriate installer
   - Handle empty args (install all from package.json)

3. **Run Command (`cmd/run.go`)**
   ```go
   tidy run <script>
   tidy <script>  # Direct alias
   ```
   - Read package.json scripts
   - Execute specified script
   - Stream output to terminal
   - Handle script not found errors

4. **Add Command (`cmd/add.go`)**
   ```go
   tidy add <packages...>
   Flags:
   - -D, --dev  : Add as dev dependency
   - -g, --grep : Grep codebase and add found packages
   ```
   - Parse packages or use grep
   - Determine dev vs prod
   - Add to package.json
   - Install packages

5. **Refactor main.go**
   - Remove current hardcoded logic
   - Initialize Cobra command tree
   - Execute root command

**Deliverables**:
- All Cobra commands implemented
- Flags properly parsed
- Commands route to existing internal functions

---

### Phase 3: Package Manager Abstraction
**Goal**: Support multiple package managers (Bun, npm, pnpm, tidy)

#### Tasks:

1. **Create `internal/installer.go`**
   ```go
   type PackageManager interface {
       Install(packages []string, isDev bool) error
       Run(script string) error
       Add(packages []string, isDev bool) error
   }
   ```

2. **Implement package managers**:
   - `TidyInstaller` (existing logic)
   - `BunInstaller` (shell out to bun)
   - `PnpmInstaller` (shell out to pnpm)
   - `NpmInstaller` (shell out to npm)

3. **Factory function**
   ```go
   func GetPackageManager(pmType string) PackageManager
   ```

4. **Update install command**
   - Use package manager abstraction
   - Pass flag to determine PM type

**Deliverables**:
- Package manager interface
- All 4 implementations
- Commands use abstraction

---

### Phase 4: Script Runner
**Goal**: Enable running package.json scripts

#### Tasks:

1. **Create `internal/runner.go`**
   - Read package.json scripts section
   - Execute script with proper shell
   - Stream stdout/stderr
   - Handle exit codes

2. **Script discovery**
   - List available scripts
   - Suggest similar scripts if not found

3. **Integration with run command**
   - Wire up to Cobra command
   - Add error handling

**Deliverables**:
- Script runner implementation
- `tidy run <script>` works
- `tidy <script>` alias works

---

### Phase 5: Bubble Tea TUI
**Goal**: Interactive UI when no command is provided

#### Tasks:

1. **Design TUI Layout**
   ```
   ┌─────────────────────────────────────────┐
   │  🧹 Tidy - Package Manager              │
   ├─────────────────────────────────────────┤
   │                                         │
   │  What would you like to do?             │
   │                                         │
   │  > 📦 Install dependencies              │
   │    🔍 Scan & install from codebase      │
   │    ▶️  Run a script                     │
   │    ➕ Add new packages                  │
   │    ℹ️  Show project info                │
   │    ❌ Exit                              │
   │                                         │
   ├─────────────────────────────────────────┤
   │  ↑/↓: Navigate  Enter: Select  q: Quit  │
   └─────────────────────────────────────────┘
   ```

2. **Create `ui/model.go`**
   ```go
   type Model struct {
       choices     []string
       cursor      int
       selected    map[int]struct{}
       state       AppState
       packages    []string
       installing  bool
       progress    string
   }
   ```

3. **Create `ui/update.go`**
   - Handle keyboard input (up/down/enter/q)
   - State transitions
   - Trigger background operations

4. **Create `ui/view.go`**
   - Render main menu
   - Render installation progress
   - Render script selection
   - Use lipgloss for styling

5. **Create `ui/commands.go`**
   - Async operations (install, scan, etc.)
   - Progress updates
   - Completion messages

6. **Menu Options Implementation**:

   **a) Install dependencies**
   - Show package.json dependencies
   - Install all with progress bar
   - Show completion status

   **b) Scan & install from codebase**
   - Run grep functionality
   - Show found packages
   - Confirm and install
   - Show progress

   **c) Run a script**
   - List available scripts from package.json
   - Select script to run
   - Execute and show output
   - Return to menu

   **d) Add new packages**
   - Text input for package names
   - Toggle dev/prod
   - Add to package.json
   - Install packages

   **e) Show project info**
   - Display package.json info
   - Show installed packages
   - Show available scripts

7. **Progress Indicators**
   - Use `bubbles/progress` for installation
   - Use `bubbles/spinner` for operations
   - Real-time updates during install

**Deliverables**:
- Fully functional TUI
- All menu options working
- Beautiful, styled interface
- Smooth user experience

---

### Phase 6: Grep Integration with -g Flag
**Goal**: Allow `tidy add -g` to scan and add packages

#### Tasks:

1. **Update add command**
   - Add `-g, --grep` flag
   - If flag present, run grep logic
   - Show found packages
   - Add to package.json
   - Install

2. **Grep output formatting**
   - Pretty print found packages
   - Show which files they were found in
   - Confirm before adding

**Deliverables**:
- `tidy add -g` scans codebase
- Found packages added to package.json
- Packages installed

---

### Phase 7: Polish & Error Handling
**Goal**: Production-ready CLI

#### Tasks:

1. **Error handling**
   - Graceful error messages
   - Helpful suggestions
   - Exit codes

2. **Help text**
   - Comprehensive help for each command
   - Examples in help text
   - ASCII art banner

3. **Configuration**
   - Optional config file (~/.tidyrc)
   - Default package manager preference
   - Color scheme preferences

4. **Logging**
   - Verbose mode (-v)
   - Debug mode (--debug)
   - Log file option

5. **Testing**
   - Unit tests for commands
   - Integration tests
   - TUI interaction tests

**Deliverables**:
- Robust error handling
- Comprehensive help
- Optional configuration
- Test coverage

---

## Technical Decisions

### 1. Package Manager Detection
- Check for lock files (bun.lockb, pnpm-lock.yaml, package-lock.json)
- Use detected PM as default
- Allow override with flags

### 2. Tidy's Own Installer
- Continue using existing logic (grep, resolve, install)
- Concurrent installation with worker pools
- Use deps.json for prod/dev classification

### 3. Script Execution
- Use `os/exec` to run scripts
- Inherit parent process environment
- Stream output in real-time

### 4. TUI State Management
- Use Bubble Tea's Elm architecture
- Immutable state updates
- Commands for async operations

### 5. Styling
- Use lipgloss for consistent styling
- Color scheme: 
  - Primary: Cyan (#00D9FF)
  - Success: Green (#00FF00)
  - Error: Red (#FF0000)
  - Warning: Yellow (#FFFF00)

---

## Dependencies to Add

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/bubbles v0.17.1
)
```

---

## Migration Strategy

### Current main.go behavior:
1. Scan codebase (grep)
2. Resolve dependencies
3. Install packages

### New behavior:
- **With command**: Execute command directly
  - `tidy install` → install from package.json
  - `tidy add -g` → scan and install (old behavior)
  - `tidy run dev` → run dev script
  
- **Without command**: Launch TUI
  - Interactive menu
  - User selects action
  - Execute selected action

### Backward Compatibility:
- `tidy add -g` replicates old behavior
- Can add alias: `tidy` → `tidy add -g` (optional)

---

## Example Usage

```bash
# Interactive mode
$ tidy
# Shows TUI menu

# Install all dependencies
$ tidy install

# Install with specific PM
$ tidy install --bun
$ tidy install --pnpm

# Install specific packages
$ tidy install react react-dom
$ tidy add react react-dom

# Install dev dependencies
$ tidy add -D typescript @types/node

# Scan and add packages from codebase
$ tidy add -g

# Run scripts
$ tidy run dev
$ tidy dev  # alias

# Show version
$ tidy version
```

---

## Success Criteria

- ✅ All commands work as specified
- ✅ TUI is intuitive and beautiful
- ✅ Multiple package managers supported
- ✅ Script running works
- ✅ Grep integration with -g flag
- ✅ Existing functionality preserved
- ✅ Good error handling
- ✅ Comprehensive help text
- ✅ Tests pass

---

## Timeline Estimate

- Phase 1: 1-2 hours
- Phase 2: 3-4 hours
- Phase 3: 2-3 hours
- Phase 4: 2 hours
- Phase 5: 4-6 hours (most complex)
- Phase 6: 1-2 hours
- Phase 7: 2-3 hours

**Total**: ~15-22 hours

---

## Next Steps

1. Review this plan
2. Approve or request changes
3. Begin Phase 1 implementation
4. Iterate through phases
5. Test and refine

---

## Notes

- Keep existing `internal/` functions intact
- Refactor only where necessary
- Maintain backward compatibility where possible
- Focus on user experience in TUI
- Make CLI intuitive and helpful
