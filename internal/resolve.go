package internal

import (
	"context"
	"sync"
	"time"
)

type pkg struct {
	name    string
	vesrion string
}

type Queue []pkg

type Deps struct {
	Version      string
	Tarball      string
	Dependencies map[string]Deps
}

type Resolved map[string]Deps

func Resolve(pkgs PackageJson) (Resolved, error) {
	resolved := make(Resolved)
	var resolvedMu sync.RWMutex

	queue := make(chan pkg, 1000)
	var pendingMu sync.Mutex
	pendingCount := 0
	activeWorkers := 0

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	processing := make(map[string]bool)
	var processingMu sync.Mutex

	worker := func() {
		defer wg.Done()
		for current := range queue {
			pendingMu.Lock()
			activeWorkers++
			pendingMu.Unlock()

			resolvedMu.RLock()
			alreadyResolved := false
			if _, exists := resolved[current.name]; exists {
				alreadyResolved = true
			}
			resolvedMu.RUnlock()

			if alreadyResolved {
				pendingMu.Lock()
				pendingCount--
				activeWorkers--
				pendingMu.Unlock()
				continue
			}

			processingMu.Lock()
			if processing[current.name] {
				processingMu.Unlock()
				pendingMu.Lock()
				pendingCount--
				activeWorkers--
				pendingMu.Unlock()
				continue
			}
			processing[current.name] = true
			processingMu.Unlock()

			pendingMu.Lock()
			pendingCount--
			pendingMu.Unlock()

			semaphore <- struct{}{}
			manifest, err := FetchManifest(current.name, current.vesrion)
			<-semaphore

			processingMu.Lock()
			delete(processing, current.name)
			processingMu.Unlock()

			pendingMu.Lock()
			activeWorkers--
			pendingMu.Unlock()

			if err != nil {
				continue
			}

			resolvedMu.Lock()
			if _, exists := resolved[current.name]; exists {
				resolvedMu.Unlock()
				continue
			}

			resolved[current.name] = Deps{
				Version:      manifest.Version,
				Tarball:      manifest.Dist.Tarball,
				Dependencies: make(map[string]Deps),
			}
			resolvedMu.Unlock()

			for depName, depVersion := range manifest.Dependencies {
				select {
				case <-ctx.Done():
					return
				default:
				}

				resolvedMu.RLock()
				if _, ok := resolved[depName]; ok {
					resolvedMu.RUnlock()
					continue
				}
				resolvedMu.RUnlock()

				processingMu.Lock()
				if processing[depName] {
					processingMu.Unlock()
					continue
				}
				processingMu.Unlock()

				select {
				case <-ctx.Done():
					return
				case queue <- pkg{name: depName, vesrion: depVersion}:
					pendingMu.Lock()
					pendingCount++
					pendingMu.Unlock()
				default:
				}
			}
		}
	}

	numWorkers := maxConcurrency
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, pkg := range transformPackageJson(pkgs) {
		queue <- pkg
		pendingMu.Lock()
		pendingCount++
		pendingMu.Unlock()
	}

	var queueOnce sync.Once
	closeQueue := func() {
		queueOnce.Do(func() {
			close(queue)
		})
	}

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pendingMu.Lock()
				pending := pendingCount
				active := activeWorkers
				pendingMu.Unlock()

				if pending == 0 && active == 0 {
					time.Sleep(100 * time.Millisecond)
					pendingMu.Lock()
					if pendingCount == 0 && activeWorkers == 0 {
						pendingMu.Unlock()
						cancel()
						closeQueue()
						return
					}
					pendingMu.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	closeQueue()

	return resolved, nil
}

func transformPackageJson(pkgs PackageJson) []pkg {
	var pkgsList []pkg
	for name, version := range pkgs.Dependencies {
		pkgsList = append(pkgsList, pkg{
			name:    name,
			vesrion: version,
		})
	}
	for name, version := range pkgs.DevDependencies {
		pkgsList = append(pkgsList, pkg{
			name:    name,
			vesrion: version,
		})
	}
	return pkgsList
}
