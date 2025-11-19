package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var ignoreDirs = map[string]bool{
	"node_modules": true,
	"public":       true,
	".git":         true,
}

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var tsFiles []string

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && ignoreDirs[d.Name()] {
			return filepath.SkipDir
		}

		ext := filepath.Ext(d.Name())
		if !d.IsDir() && (ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx") {
			tsFiles = append(tsFiles, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
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

	externalPackages := make(map[string]bool)

	for _, filePath := range tsFiles {
		packages, err := extractExternalPackages(filePath)
		if err != nil {
			fmt.Printf("Warning: Could not read %s: %v\n", filePath, err)
			continue
		}

		for _, pkg := range packages {
			externalPackages[pkg] = true
		}
	}

	var missingPackages []string
	for pkg := range externalPackages {
		_, inDeps := packageJSON.Dependencies[pkg]
		_, inDevDeps := packageJSON.DevDependencies[pkg]
		if !inDeps && !inDevDeps {
			missingPackages = append(missingPackages, pkg)
		}
	}

	if len(missingPackages) == 0 {
		fmt.Println("All packages are already installed!")
		return
	}

	fmt.Println("Missing packages found:")
	for _, pkg := range missingPackages {
		fmt.Printf("  - %s\n", pkg)
	}
	fmt.Printf("\nInstalling packages...\n")

	cmd := exec.Command("bun", append([]string{"add"}, missingPackages...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error installing packages: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nPackages installed successfully!")
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

func extractExternalPackages(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
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
				// Skip relative imports (./ or ../), path aliases (@/), and Node.js built-ins (node:)
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

	return packages, nil
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
