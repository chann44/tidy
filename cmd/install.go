package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/chann44/tidy/internal"
	"github.com/spf13/cobra"
)

var (
	useBun  bool
	usePnpm bool
	useNpm  bool
)

var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install packages",
	Long: `Install packages from package.json or install specific packages.

Examples:
  tidy install              # Install all dependencies from package.json
  tidy install react        # Install react
  tidy install --bun        # Install using Bun
  tidy install --pnpm       # Install using pnpm
  tidy install --npm        # Install using npm`,
	Aliases: []string{"i"},
	Run: func(cmd *cobra.Command, args []string) {
		pm := getPackageManager()

		if len(args) > 0 {
			installSpecificPackages(pm, args)
		} else {
			installAllPackages()
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&useBun, "bun", false, "use Bun package manager")
	installCmd.Flags().BoolVar(&usePnpm, "pnpm", false, "use pnpm package manager")
	installCmd.Flags().BoolVar(&useNpm, "npm", false, "use npm package manager")
}

func getPackageManager() string {
	if useBun {
		return "bun"
	}
	if usePnpm {
		return "pnpm"
	}
	if useNpm {
		return "npm"
	}
	return "tidy"
}

func installSpecificPackages(pm string, packages []string) {
	if pm != "tidy" {
		installer := internal.GetPackageManager(pm)
		if err := installer.Install(packages, false); err != nil {
			fmt.Printf("Error installing packages: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Packages installed successfully!")
		return
	}

	fmt.Printf("Installing %d package(s) using Tidy...\n", len(packages))

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	var jsn internal.PackageJson
	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		jsn = internal.PackageJson{
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		}
	} else {
		jsn, _ = internal.ReadJson(wd)
	}

	for _, pkg := range packages {
		jsn.Dependencies[pkg] = "latest"
	}

	resolved, err := internal.Resolve(jsn)
	if err != nil {
		fmt.Printf("Error resolving dependencies: %v\n", err)
		os.Exit(1)
	}

	installPackages(resolved)
}

func installAllPackages() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		fmt.Println("No package.json found. Nothing to install.")
		return
	}

	jsn, err := internal.ReadJson(wd)
	if err != nil {
		fmt.Printf("Error reading package.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📦 Installing dependencies from package.json...")

	resolved, err := internal.Resolve(jsn)
	if err != nil {
		fmt.Printf("Error resolving dependencies: %v\n", err)
		os.Exit(1)
	}

	installPackages(resolved)
}

func installPackages(resolved map[string]internal.Deps) {
	fmt.Printf("Installing %d package(s)...\n\n", len(resolved))

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	installing := make(map[string]bool)
	var installingMu sync.Mutex

	for name, deps := range resolved {
		if internal.IsInstalled(name) {
			if !IsQuiet() {
				fmt.Printf("⏭️  Skipping %s@%s (already installed)\n", name, deps.Version)
			}
			continue
		}

		installingMu.Lock()
		if installing[name] {
			installingMu.Unlock()
			if !IsQuiet() {
				fmt.Printf("⏭️  Skipping %s@%s (installation in progress)\n", name, deps.Version)
			}
			continue
		}
		installing[name] = true
		installingMu.Unlock()

		wg.Add(1)
		go func(name string, deps internal.Deps) {
			defer wg.Done()
			defer func() {
				installingMu.Lock()
				delete(installing, name)
				installingMu.Unlock()
			}()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if IsVerbose() {
				fmt.Printf("📥 Installing %s@%s\n", name, deps.Version)
			}

			err := internal.Install(name, deps.Tarball)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				mu.Unlock()
				fmt.Printf("❌ Error installing %s: %v\n", name, err)
				return
			}

			if !IsQuiet() {
				fmt.Printf("✓ Installed %s@%s\n", name, deps.Version)
			}
		}(name, deps)
	}

	wg.Wait()

	fmt.Println()
	if len(errors) > 0 {
		fmt.Printf("⚠️  Completed with %d error(s):\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	} else {
		fmt.Println("✅ All packages installed successfully!")
	}
}
