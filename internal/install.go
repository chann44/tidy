package internal

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func IsInstalled(name string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	packageDir := filepath.Join(cwd, "node_modules", name)
	info, err := os.Stat(packageDir)
	if err != nil || !info.IsDir() {
		return false
	}
	packageJson := filepath.Join(packageDir, "package.json")
	_, err = os.Stat(packageJson)
	return err == nil
}

const StoreDirName = ".tidy/store"

func getStoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, StoreDirName), nil
}
func Install(name, url string) error {
	storeDir, err := getStoreDir()
	if err != nil {
		return err
	}
	pkgId := name + "@" + extractVersionFromUrl(url)
	cachedPkgDir := filepath.Join(storeDir, pkgId)
	if _, err := os.Stat(cachedPkgDir); os.IsNotExist(err) {
		if err := downloadToStore(url, cachedPkgDir); err != nil {
			return err
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	targetDir := filepath.Join(cwd, "node_modules", name)
	os.RemoveAll(targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	if err := linkPackage(cachedPkgDir, targetDir); err != nil {
		return err
	}
	return nil
}
func extractVersionFromUrl(url string) string {
	parts := strings.Split(url, "-")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		return strings.TrimSuffix(last, ".tgz")
	}
	return "latest"
}
func downloadToStore(url, destDir string) error {
	tempDir := destDir + ".tmp"
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(header.Name, "package/")
		if rel == "" {
			continue
		}
		target := filepath.Join(tempDir, rel)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			dst, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(dst, tr); err != nil {
				dst.Close()
				return err
			}
			dst.Close()
		}
	}
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return err
	}
	return os.Rename(tempDir, destDir)
}
func linkPackage(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		err = os.Link(path, targetPath)
		if err == nil {
			return nil
		}
		return copyFile(path, targetPath)
	})
}
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	return err
}
