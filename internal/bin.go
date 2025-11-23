package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

func LinkBinaries() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	binDir := filepath.Join(cwd, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	nodeModulesDir := filepath.Join(cwd, "node_modules")
	entries, err := os.ReadDir(nodeModulesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".bin" {
			continue
		}

		pkgDir := filepath.Join(nodeModulesDir, entry.Name())
		pkgJsonPath := filepath.Join(pkgDir, "package.json")
		
		if _, err := os.Stat(pkgJsonPath); os.IsNotExist(err) {
			continue
		}

		binaries, err := extractBinaries(pkgJsonPath, pkgDir)
		if err != nil {
			continue // Skip packages without valid bin entries
		}

		for binName, binPath := range binaries {
			linkPath := filepath.Join(binDir, binName)
			absLinkPath, err := filepath.Abs(linkPath)
			if err != nil {
				continue
			}
			os.Remove(absLinkPath)
			
			absBinPath, err := filepath.Abs(binPath)
			if err != nil {
				continue
			}
			if err := ensureExecutable(absBinPath); err != nil {
				continue
			}
			
			if err := createBinLink(absBinPath, absLinkPath); err != nil {
				continue
			}
			
			_ = ensureExecutable(absLinkPath)
		}
	}

	return nil
}

func extractBinaries(pkgJsonPath, pkgDir string) (map[string]string, error) {
	data, err := os.ReadFile(pkgJsonPath)
	if err != nil {
		return nil, err
	}

	var pkgJson struct {
		Bin interface{} `json:"bin"`
	}
	if err := json.Unmarshal(data, &pkgJson); err != nil {
		return nil, err
	}

	if pkgJson.Bin == nil {
		return nil, nil
	}

	binaries := make(map[string]string)

	switch v := pkgJson.Bin.(type) {
	case string:
		binPath := filepath.Join(pkgDir, v)
		if _, err := os.Stat(binPath); err == nil {
			pkgName := filepath.Base(pkgDir)
			binaries[pkgName] = binPath
		}
	case map[string]interface{}:
		for binName, binPathInterface := range v {
			if binPathStr, ok := binPathInterface.(string); ok {
				binPath := filepath.Join(pkgDir, binPathStr)
				if _, err := os.Stat(binPath); err == nil {
					binaries[binName] = binPath
				}
			}
		}
	}

	return binaries, nil
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	
	mode := info.Mode()
	
	newMode := mode | 0111
	
	if mode != newMode {
		return os.Chmod(path, newMode)
	}
	
	return nil
}

func createBinLink(target, linkPath string) error {
	if runtime.GOOS == "windows" {
		if err := os.Symlink(target, linkPath); err != nil {
			if err := CopyFile(target, linkPath); err != nil {
				return err
			}
			return ensureExecutable(linkPath)
		}
		return nil
	}

	return os.Symlink(target, linkPath)
}

