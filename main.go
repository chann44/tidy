package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/chann44/tidy/internal"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
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

	fmt.Println("Scanning codebase for packages...")
	internal.Grep(jsn)

	if _, err := os.Stat("package.json"); !os.IsNotExist(err) {
		jsn, _ = internal.ReadJson(wd)
	}

	fmt.Println("\nResolving dependencies...")
	resolved, err := internal.Resolve(jsn)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nInstalling packages...")

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	installing := make(map[string]bool)
	var installingMu sync.Mutex

	for name, deps := range resolved {
		if internal.IsInstalled(name) {
			fmt.Printf("Skipping %s@%s (already installed)\n", name, deps.Version)
			continue
		}

		installingMu.Lock()
		if installing[name] {
			installingMu.Unlock()
			fmt.Printf("Skipping %s@%s (installation in progress)\n", name, deps.Version)
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

			fmt.Printf("Installing %s@%s\n", name, deps.Version)
			err := internal.Install(name, deps.Tarball)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
				mu.Unlock()
				fmt.Printf("Error installing %s: %v\n", name, err)
				return
			}
		}(name, deps)
	}

	wg.Wait()

	if len(errors) > 0 {
		fmt.Printf("\nCompleted with %d error(s):\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	} else {
		fmt.Println("\nAll packages installed successfully!")
	}
}
