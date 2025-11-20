package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

//go:embed deps.json
var embeddedDepsJSON []byte

var ignoreDirs = map[string]bool{
	"node_modules": true,
	"public":       true,
	".git":         true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"out":          true,
}

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type DepsConfig struct {
	Prod []string `json:"prod"`
	Dev  []string `json:"dev"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pathsChan := make(chan string)
	packagesChan := make(chan string)
	wg := &sync.WaitGroup{}

	go func() {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() && ignoreDirs[d.Name()] {
				return filepath.SkipDir
			}

			if !d.IsDir() {
				ext := filepath.Ext(d.Name())
				name := d.Name()
				isStandardFile := ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
				isConfigFile := strings.Contains(name, ".config.") &&
					(ext == ".js" || ext == ".ts" || ext == ".mjs" || ext == ".cjs")

				if isStandardFile || isConfigFile {
					pathsChan <- path
				}
			}

			return nil
		})

		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
		}
		close(pathsChan)
	}()

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(pathsChan, packagesChan, wg)
	}

	go func() {
		wg.Wait()
		close(packagesChan)
	}()

	externalPackages := make(map[string]bool)
	for pkg := range packagesChan {
		externalPackages[pkg] = true
	}

	packageJSONPath := filepath.Join(root, "package.json")
	packageJSON, err := readPackageJSON(packageJSONPath)
	if err != nil {
		fmt.Printf("Warning: Could not read package.json: %v\n", err)
		packageJSON = &PackageJSON{
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		}
	}

	depsConfig, err := loadDepsConfig(root)
	if err != nil {
		fmt.Printf("Warning: Could not read deps.json: %v\n", err)
		depsConfig = &DepsConfig{
			Prod: []string{},
			Dev:  []string{},
		}
	}

	prodDeps := make(map[string]bool)
	devDeps := make(map[string]bool)
	for _, pkg := range depsConfig.Prod {
		prodDeps[pkg] = true
	}
	for _, pkg := range depsConfig.Dev {
		devDeps[pkg] = true
	}

	var missingProdPackages []string
	var missingDevPackages []string
	for pkg := range externalPackages {
		_, inDeps := packageJSON.Dependencies[pkg]
		_, inDevDeps := packageJSON.DevDependencies[pkg]
		if !inDeps && !inDevDeps {
			if devDeps[pkg] {
				missingDevPackages = append(missingDevPackages, pkg)
			} else {
				missingProdPackages = append(missingProdPackages, pkg)
			}
		}
	}

	totalMissing := len(missingProdPackages) + len(missingDevPackages)
	if totalMissing == 0 {
		fmt.Println("All packages are already installed!")
		return
	}

	fmt.Println("Missing packages found:")
	if len(missingProdPackages) > 0 {
		fmt.Println("\nProduction dependencies:")
		for _, pkg := range missingProdPackages {
			fmt.Printf("  - %s\n", pkg)
		}
	}
	if len(missingDevPackages) > 0 {
		fmt.Println("\nDev dependencies:")
		for _, pkg := range missingDevPackages {
			fmt.Printf("  - %s\n", pkg)
		}
	}

	if len(missingProdPackages) > 0 {
		fmt.Printf("\nInstalling production packages...\n")
		cmd := exec.Command("bun", append([]string{"add"}, missingProdPackages...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error installing production packages: %v\n", err)
			os.Exit(1)
		}
	}

	if len(missingDevPackages) > 0 {
		fmt.Printf("\nInstalling dev packages...\n")
		cmd := exec.Command("bun", append([]string{"add", "-d"}, missingDevPackages...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error installing dev packages: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\nAll packages installed successfully!")
}

func worker(paths <-chan string, packages chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for filePath := range paths {
		pkgs := extractPackagesFromFile(filePath)
		for _, pkg := range pkgs {
			packages <- pkg
		}
	}
}

func extractPackagesFromFile(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	content := string(data)
	var packages []string
	seen := make(map[string]bool)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`import\s+.*\s+from\s+["']([^"']+)["']`),
		regexp.MustCompile(`import\s+["']([^"']+)["']`),
		regexp.MustCompile(`require\(["']([^"']+)["']\)`),
		regexp.MustCompile(`import\(["']([^"']+)["']\)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pkg := match[1]
				if !strings.HasPrefix(pkg, ".") && !strings.HasPrefix(pkg, "/") && !strings.HasPrefix(pkg, "@/") && !strings.HasPrefix(pkg, "node:") {
					packageName := extractPackageName(pkg)
					if packageName != "" && !seen[packageName] {
						packages = append(packages, packageName)
						seen[packageName] = true
					}
				}
			}
		}
	}

	return packages
}

func loadDepsConfig(root string) (*DepsConfig, error) {
	fmt.Printf("Loading embedded deps.json (size: %d bytes)...\n", len(embeddedDepsJSON))

	var config DepsConfig
	if err := json.Unmarshal(embeddedDepsJSON, &config); err != nil {
		return nil, err
	}

	fmt.Printf("Loaded %d production packages and %d dev packages from embedded config\n",
		len(config.Prod), len(config.Dev))

	return &config, nil
}

func readPackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	if pkg.Dependencies == nil {
		pkg.Dependencies = make(map[string]string)
	}
	if pkg.DevDependencies == nil {
		pkg.DevDependencies = make(map[string]string)
	}

	return &pkg, nil
}

func extractPackageName(importPath string) string {
	if idx := strings.Index(importPath, "?"); idx != -1 {
		importPath = importPath[:idx]
	}
	if idx := strings.Index(importPath, "#"); idx != -1 {
		importPath = importPath[:idx]
	}

	if strings.HasPrefix(importPath, "@") {
		parts := strings.Split(importPath, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}

	parts := strings.Split(importPath, "/")
	return parts[0]
}
