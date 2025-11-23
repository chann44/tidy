package internal

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var ignoreDirs = map[string]bool{
	"node_modules": true,
	"public":       true,
	".git":         true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"out":          true,
}

type DepsConfig struct {
	Prod []string `json:"prod"`
	Dev  []string `json:"dev"`
}

func Grep(packageJSON PackageJson) {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pathsChan := make(chan string, 10)
	packagesChan := make(chan string, 100)
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

	if packageJSON.Dependencies == nil {
		packageJSON.Dependencies = make(map[string]string)
	}
	if packageJSON.DevDependencies == nil {
		packageJSON.DevDependencies = make(map[string]string)
	}

	depsConfig, err := loadDepsConfig(root)
	if err != nil {
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
		return
	}

	if len(missingProdPackages) > 0 {
		for _, pkg := range missingProdPackages {
			packageJSON.Dependencies[pkg] = "*"
		}
	}
	if len(missingDevPackages) > 0 {
		for _, pkg := range missingDevPackages {
			packageJSON.DevDependencies[pkg] = "*"
		}
	}

	packageJSONPath := filepath.Join(root, "package.json")
	if err := writePackageJSON(packageJSONPath, packageJSON); err != nil {
		os.Exit(1)
	}

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
	depsPath := filepath.Join(root, "deps.json")
	data, err := os.ReadFile(depsPath)
	if err != nil {
		return nil, err
	}

	var config DepsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func writePackageJSON(path string, pkg PackageJson) error {
	data, err := os.ReadFile(path)
	var existing map[string]interface{}
	if err == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			existing = make(map[string]interface{})
		}
	} else {
		existing = make(map[string]interface{})
	}

	existing["dependencies"] = pkg.Dependencies
	existing["devDependencies"] = pkg.DevDependencies

	updatedData, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedData, 0644)
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
