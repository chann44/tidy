package ui

import (
	"fmt"
	"os"
	"time"

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

		count := 0
		total := len(resolved)

		for name, deps := range resolved {
			if !internal.IsInstalled(name) {
				time.Sleep(10 * time.Millisecond)

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

				count++
				return installProgressMsg{
					pkg:         name,
					version:     deps.Version,
					pkgProgress: 100,
					pkgDone:     true,
					pkgFailed:   false,
				}
			}
		}

		return installCompleteMsg{
			message: fmt.Sprintf("✅ Successfully installed %d / %d package(s)", count, total),
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

		count := 0
		total := len(resolved)

		for name, deps := range resolved {
			if !internal.IsInstalled(name) {
				time.Sleep(10 * time.Millisecond)

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

				count++
				return installProgressMsg{
					pkg:         name,
					version:     deps.Version,
					pkgProgress: 100,
					pkgDone:     true,
					pkgFailed:   false,
				}
			}
		}

		return installCompleteMsg{
			message: fmt.Sprintf("✅ Scanned and installed %d / %d package(s)", count, total),
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
