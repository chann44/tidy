package internal
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)
const (
	cacheDir = ".tidy-cache"
	cacheTTL = 24 * time.Hour  
)
type CachedManifest struct {
	Manifest Manifest  `json:"manifest"`
	CachedAt time.Time `json:"cached_at"`
}
func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, cacheDir, "manifests")
	return cacheDir, nil
}
func getCacheKey(pkg, version string) string {
	hash := sha256.Sum256([]byte(pkg + "@" + version))
	return hex.EncodeToString(hash[:])
}
func LoadFromDiskCache(pkg, version string) (*Manifest, bool) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, false
	}
	cacheKey := getCacheKey(pkg, version)
	cachePath := filepath.Join(cacheDir, cacheKey+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	var cached CachedManifest
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if time.Since(cached.CachedAt) > cacheTTL {
		os.Remove(cachePath)  
		return nil, false
	}
	return &cached.Manifest, true
}
func SaveToDiskCache(pkg, version string, manifest Manifest) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	cached := CachedManifest{
		Manifest: manifest,
		CachedAt: time.Now(),
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	cacheKey := getCacheKey(pkg, version)
	cachePath := filepath.Join(cacheDir, cacheKey+".json")
	return os.WriteFile(cachePath, data, 0644)
}
func ClearCache() error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(cacheDir)
}