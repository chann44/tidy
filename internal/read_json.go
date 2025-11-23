package internal
import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)
type PackageJson struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}
func ReadJson(wd string) (PackageJson, error) {
	path := filepath.Join(wd, "package.json")
	var jsn PackageJson
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &jsn)
	if err != nil {
		log.Fatal(err)
	}
	return jsn, nil
}