# 🧹 Tidy

```
         ╭─────────────────╮
        ╱│                 │╲
       ╱ │     ╭─────╮     │ ╲
      ╱  │    ╱  • •  ╲    │  ╲
     ╱   │    │   ◡   │    │   ╲
    ╱    │    ╲_____╱│    │    ╲
   ╱     │           │    │     ╲
  ╱      │   ╱│ ││││╲│    │      ╲
 ╱       │   │ ││││ ││    │       ╲
╱        │   ╲╰─────╯╱│    │        ╲
│══════════════════════════════════│
│  📦   🎯   ✨   🧹   ✨   🎯   📦  │
│══════════════════════════════════│
       \           /
        \         /
         \_______/
```

A friendly helper that automatically finds and installs missing npm/bun dependencies.

A Go tool that automatically finds and installs missing npm/bun dependencies by scanning your TypeScript and JavaScript files.

## What It Does

`tidy` scans your project for TypeScript/JavaScript files, extracts external package imports, and automatically installs any missing packages that aren't already in your `package.json`.

## Features

- 🔍 Scans `.ts`, `.tsx`, `.js`, and `.jsx` files
- 🚫 Automatically ignores common directories (`node_modules`, `.git`, `public`)
- 📋 Checks against both `dependencies` and `devDependencies`
- 🚀 Installs missing packages using `bun add`
- ✨ Clean and simple output

## Installation

```bash
go install github.com/chann44/tidy@latest
```

Or clone and build:

```bash
git clone https://github.com/chann44/tidy.git
cd tidy
go build -o tidy
```

## Usage

Navigate to your project directory and run:

```bash
tidy
```

The tool will:
1. Scan all TypeScript/JavaScript files in the current directory
2. Extract external package imports
3. Check which packages are missing from `package.json`
4. Install any missing packages using `bun add`

### Example Output

```
Missing packages found:
  - react
  - zod
  - @types/node

Installing packages...

Packages installed successfully!
```

## How It Works

1. **File Scanning**: Recursively walks through your project directory, ignoring common build/dependency folders
2. **Import Extraction**: Uses regex patterns to find import statements and require calls
3. **Package Detection**: Identifies external packages (skips relative imports, path aliases, and Node.js built-ins)
4. **Dependency Check**: Compares found packages against your `package.json`
5. **Auto-Install**: Runs `bun add` for any missing packages

## Requirements

- Go 1.25.3 or later
- Bun (for package installation)

## License

MIT

