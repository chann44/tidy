package ui
import (
	"fmt"
	"os"
	"sync"
	"github.com/chann44/tidy/internal"
	tea "github.com/charmbracelet/bubbletea"
)
type installCompleteMsg struct {
	message string
}
type installProgressMsg struct {
	progress    string
	pkg         string
	version     string
	pkgProgress int
	pkgDone     bool
	pkgFailed   bool
}
type errorMsg struct {
	err error
}
type scriptsLoadedMsg struct {
	scripts []string
}
type projectInfoMsg struct {
	info string
}
func installDependenciesCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return errorMsg{err: err}
		}
		if _, err := os.Stat("package.json"); os.IsNotExist(err) {
			return errorMsg{err: fmt.Errorf("no package.json found")}
		}
		jsn, err := internal.ReadJson(wd)
		if err != nil {
			return errorMsg{err: err}
		}
		resolved, err := internal.Resolve(jsn)
		if err != nil {
			return errorMsg{err: err}
		}
		var toInstall []string
		for name := range resolved {
			if !internal.IsInstalled(name) {
				toInstall = append(toInstall, name)
			}
		}
		total := len(toInstall)
		if total == 0 {
			return installCompleteMsg{
				message: fmt.Sprintf("✅ All %d packages are already installed!", len(resolved)),
			}
		}
		const maxConcurrency = 50
		semaphore := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		results := make(chan installProgressMsg, total)
		for _, name := range toInstall {
			deps := resolved[name]
			wg.Add(1)
			go func(name string, deps internal.Deps) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				err := internal.Install(name, deps.Tarball)
				if err != nil {
					results <- installProgressMsg{
						pkg:         name,
						version:     deps.Version,
						pkgProgress: 100,
						pkgDone:     false,
						pkgFailed:   true,
					}
					return
				}
				results <- installProgressMsg{
					pkg:         name,
					version:     deps.Version,
					pkgProgress: 100,
					pkgDone:     true,
					pkgFailed:   false,
				}
			}(name, deps)
		}
		go func() {
			wg.Wait()
			close(results)
		}()
		count := 0
		for range results {
			count++
		}
		return installCompleteMsg{
			message: fmt.Sprintf("✅ Successfully installed %d packages", count),
		}
	}
}
func installPackageCmd(name string, deps internal.Deps) tea.Cmd {
	return func() tea.Msg {
		err := internal.Install(name, deps.Tarball)
		if err != nil {
			return installProgressMsg{
				pkg:         name,
				version:     deps.Version,
				pkgProgress: 100,
				pkgDone:     false,
				pkgFailed:   true,
			}
		}
		return installProgressMsg{
			pkg:         name,
			version:     deps.Version,
			pkgProgress: 100,
			pkgDone:     true,
			pkgFailed:   false,
		}
	}
}
func scanAndInstallCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return errorMsg{err: err}
		}
		var jsn internal.PackageJson
		if _, err := os.Stat("package.json"); os.IsNotExist(err) {
			jsn = internal.PackageJson{
				Dependencies:    make(map[string]string),
				DevDependencies: make(map[string]string),
			}
		} else {
			jsn, _ = internal.ReadJson(wd)
		}
		internal.Grep(jsn)
		if _, err := os.Stat("package.json"); !os.IsNotExist(err) {
			jsn, _ = internal.ReadJson(wd)
		}
		resolved, err := internal.Resolve(jsn)
		if err != nil {
			return errorMsg{err: err}
		}
		var toInstall []string
		for name := range resolved {
			if !internal.IsInstalled(name) {
				toInstall = append(toInstall, name)
			}
		}
		total := len(toInstall)
		if total == 0 {
			return installCompleteMsg{
				message: fmt.Sprintf("✅ Scanned and installed %d / %d package(s)", 0, len(resolved)),
			}
		}
		const maxConcurrency = 50
		semaphore := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		results := make(chan installProgressMsg, total)
		for _, name := range toInstall {
			deps := resolved[name]
			wg.Add(1)
			go func(name string, deps internal.Deps) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				err := internal.Install(name, deps.Tarball)
				if err != nil {
					results <- installProgressMsg{
						pkg:         name,
						version:     deps.Version,
						pkgProgress: 100,
						pkgDone:     false,
						pkgFailed:   true,
					}
					return
				}
				results <- installProgressMsg{
					pkg:         name,
					version:     deps.Version,
					pkgProgress: 100,
					pkgDone:     true,
					pkgFailed:   false,
				}
			}(name, deps)
		}
		go func() {
			wg.Wait()
			close(results)
		}()
		count := 0
		for range results {
			count++
		}
		return installCompleteMsg{
			message: fmt.Sprintf("✅ Scanned and installed %d / %d package(s)", count, len(resolved)),
		}
	}
}
func loadScriptsCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return errorMsg{err: err}
		}
		if _, err := os.Stat("package.json"); os.IsNotExist(err) {
			return errorMsg{err: fmt.Errorf("no package.json found")}
		}
		jsn, err := internal.ReadJson(wd)
		if err != nil {
			return errorMsg{err: err}
		}
		runner := internal.NewScriptRunner(jsn)
		scripts := runner.ListScripts()
		return scriptsLoadedMsg{scripts: scripts}
	}
}
type runScriptMsg struct {
	scriptName string
}
func runScriptCmd(scriptName string) tea.Cmd {
	return func() tea.Msg {
		return runScriptMsg{scriptName: scriptName}
	}
}
func showProjectInfoCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return errorMsg{err: err}
		}
		if _, err := os.Stat("package.json"); os.IsNotExist(err) {
			return errorMsg{err: fmt.Errorf("no package.json found")}
		}
		jsn, err := internal.ReadJson(wd)
		if err != nil {
			return errorMsg{err: err}
		}
		info := fmt.Sprintf("Name: %s\nVersion: %s\nDependencies: %d\nDev Dependencies: %d\nScripts: %d",
			jsn.Name, jsn.Version, len(jsn.Dependencies), len(jsn.DevDependencies), len(jsn.Scripts))
		return projectInfoMsg{info: info}
	}
}