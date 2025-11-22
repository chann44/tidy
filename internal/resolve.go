package internal

import "fmt"

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
	var queue Queue

	for _, pkg := range transformPackageJson(pkgs) {
		queue = append(queue, pkg)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, exists := resolved[current.name]; exists {
			continue
		}

		manifest, err := FetchManifest(current.name, current.vesrion)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch %s@%s: %v\n", current.name, current.vesrion, err)
			continue
		}

		resolved[current.name] = Deps{
			Version:      manifest.Version,
			Tarball:      manifest.Dist.Tarball,
			Dependencies: make(map[string]Deps),
		}

		for depName, depVersion := range manifest.Dependencies {
			if _, ok := resolved[depName]; ok {
				continue
			}
			queue = append(queue, pkg{
				name:    depName,
				vesrion: depVersion,
			})
		}
	}

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
	return pkgsList
}
