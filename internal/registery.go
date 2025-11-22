package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const REGISTRY_URL = "https://registry.npmjs.org"

type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		Tarball string `json:"tarball"`
		Shasum  string `json:"shasum"`
		Size    int    `json:"size"`
	} `json:"dist"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	ID           string            `json:"_id"`
}

func stripVersionPrefix(version string) string {
	version = strings.TrimSpace(version)

	if version == "*" || version == "x" || version == "X" {
		return "latest"
	}

	if strings.Contains(version, " - ") {
		return "latest"
	}

	prefixes := []string{"^", "~", ">=", "<=", ">", "<", "="}
	for _, prefix := range prefixes {
		version = strings.TrimPrefix(version, prefix)
	}

	version = strings.TrimSpace(version)

	if version == "" || version == "*" || version == "x" || version == "X" {
		return "latest"
	}

	if len(version) == 1 && (version == "1" || version == "2" || version == "3" || version == "4" || version == "5" || version == "6" || version == "7" || version == "8" || version == "9") {
		return "latest"
	}

	return version
}

func FetchManifest(pkg, version string) (Manifest, error) {
	exactVersion := stripVersionPrefix(version)
	url := REGISTRY_URL + "/" + pkg + "/" + exactVersion

	resp, err := http.Get(url)
	if err != nil {
		return Manifest{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Manifest{}, fmt.Errorf("failed to fetch: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("failed to unmarshal: %w, response: %s", err, string(body[:min(len(body), 1000)]))
	}

	return manifest, nil
}
