package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ScriptRunner struct {
	packageJson PackageJson
}

func NewScriptRunner(packageJson PackageJson) *ScriptRunner {
	return &ScriptRunner{
		packageJson: packageJson,
	}
}
func (sr *ScriptRunner) Run(scriptName string) error {
	if sr.packageJson.Scripts == nil {
		return fmt.Errorf("no scripts defined in package.json")
	}
	script, exists := sr.packageJson.Scripts[scriptName]
	if !exists {
		return fmt.Errorf("script '%s' not found in package.json\n\nAvailable scripts:\n%s",
			scriptName, sr.listScripts())
	}
	return sr.executeScript(script)
}
func (sr *ScriptRunner) ListScripts() []string {
	if sr.packageJson.Scripts == nil {
		return []string{}
	}
	scripts := make([]string, 0, len(sr.packageJson.Scripts))
	for name := range sr.packageJson.Scripts {
		scripts = append(scripts, name)
	}
	return scripts
}
func (sr *ScriptRunner) listScripts() string {
	scripts := sr.ListScripts()
	if len(scripts) == 0 {
		return "  (none)"
	}
	var sb strings.Builder
	for _, name := range scripts {
		sb.WriteString(fmt.Sprintf("  • %s: %s\n", name, sr.packageJson.Scripts[name]))
	}
	return sb.String()
}
func (sr *ScriptRunner) executeScript(script string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(cwd, "node_modules")); err == nil {
		_ = LinkBinaries() // Silently fail if linking fails, PATH will still work if binaries exist
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", script)
	} else {
		cmd = exec.Command("sh", "-c", script)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	binPath := filepath.Join(cwd, "node_modules", ".bin")
	pathEnv := os.Getenv("PATH")
	newPath := fmt.Sprintf("%s%c%s", binPath, os.PathListSeparator, pathEnv)

	env := os.Environ()
	found := false
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + newPath
			found = true
			break
		}
	}
	if !found {
		env = append(env, "PATH="+newPath)
	}

	cmd.Env = env

	return cmd.Run()
}
func (sr *ScriptRunner) GetScript(name string) (string, bool) {
	if sr.packageJson.Scripts == nil {
		return "", false
	}
	script, exists := sr.packageJson.Scripts[name]
	return script, exists
}
