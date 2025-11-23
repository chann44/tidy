package internal
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)
const REGISTRY_URL = "https://registry.npmjs.org"
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
	manifestCache  = make(map[string]Manifest)
	cacheMu        sync.RWMutex
)
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
	if version == "" || version == "*" || version == "x" || version == "X" || version == "latest" {
		return "latest"
	}
	if strings.HasPrefix(version, "npm:") {
		parts := strings.Split(version, "@")
		if len(parts) >= 2 {
			version = parts[len(parts)-1]
		}
	}
	if strings.Contains(version, "||") {
		parts := strings.Split(version, "||")
		version = strings.TrimSpace(parts[0])
	}
	if strings.Contains(version, " - ") {
		parts := strings.Split(version, " - ")
		version = strings.TrimSpace(parts[0])
	}
	prefixes := []string{"^", "~", ">=", "<=", ">", "<", "=", "v"}
	for _, prefix := range prefixes {
		version = strings.TrimPrefix(version, prefix)
	}
	version = strings.TrimSpace(version)
	if version == "" || version == "*" || version == "x" || version == "X" {
		return "latest"
	}
	parts := strings.Split(version, ".")
	if len(parts) == 1 && isDigit(parts[0]) {
		return parts[0] + ".x"
	}
	if len(parts) == 2 && isDigit(parts[0]) && (parts[1] == "x" || parts[1] == "X" || parts[1] == "*") {
		return parts[0] + ".x"
	}
	if strings.Contains(version, "x") || strings.Contains(version, "X") || strings.Contains(version, "*") {
		version = strings.ReplaceAll(version, "x", "0")
		version = strings.ReplaceAll(version, "X", "0")
		version = strings.ReplaceAll(version, "*", "0")
	}
	return version
}
func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,  
			Transport: &http.Transport{
				MaxIdleConns:        100,  
				MaxIdleConnsPerHost: 20,   
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,  
				ForceAttemptHTTP2:   true,   
				MaxConnsPerHost:     0,      
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	})
	return httpClient
}
func FetchManifest(pkg, version string) (Manifest, error) {
	exactVersion := stripVersionPrefix(version)
	cacheKey := pkg + "@" + exactVersion
	if diskCached, ok := LoadFromDiskCache(pkg, exactVersion); ok {
		cacheMu.Lock()
		manifestCache[cacheKey] = *diskCached
		cacheMu.Unlock()
		return *diskCached, nil
	}
	cacheMu.RLock()
	if cached, ok := manifestCache[cacheKey]; ok {
		cacheMu.RUnlock()
		return cached, nil
	}
	cacheMu.RUnlock()
	url := REGISTRY_URL + "/" + pkg + "/" + exactVersion
	client := getHTTPClient()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Manifest{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tidy/1.0")
	resp, err := client.Do(req)
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
	cacheMu.Lock()
	manifestCache[cacheKey] = manifest
	cacheMu.Unlock()
	go SaveToDiskCache(pkg, exactVersion, manifest)
	return manifest, nil
}