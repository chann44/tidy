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

func Install(name, url string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	nodeModules := filepath.Join(cwd, "node_modules")
	_ = os.MkdirAll(nodeModules, 0755)

	packageDir := filepath.Join(nodeModules, name)

	_ = os.MkdirAll(filepath.Dir(packageDir), 0755)
	_ = os.MkdirAll(packageDir, 0755)

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

		target := filepath.Join(packageDir, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0755)

		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
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

	return nil
}
