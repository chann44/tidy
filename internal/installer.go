package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PackageManager interface {
	Install(packages []string, isDev bool) error
	Run(script string) error
	Add(packages []string, isDev bool) error
}

func GetPackageManager(pmType string) PackageManager {
	switch strings.ToLower(pmType) {
	case "bun":
		return &BunInstaller{}
	case "pnpm":
		return &PnpmInstaller{}
	case "npm":
		return &NpmInstaller{}
	default:
		return &TidyInstaller{}
	}
}

type BunInstaller struct{}

func (b *BunInstaller) Install(packages []string, isDev bool) error {
	args := []string{"add"}
	if isDev {
		args = append(args, "-d")
	}
	args = append(args, packages...)

	cmd := exec.Command("bun", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *BunInstaller) Run(script string) error {
	cmd := exec.Command("bun", "run", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *BunInstaller) Add(packages []string, isDev bool) error {
	return b.Install(packages, isDev)
}

type PnpmInstaller struct{}

func (p *PnpmInstaller) Install(packages []string, isDev bool) error {
	args := []string{"add"}
	if isDev {
		args = append(args, "-D")
	}
	args = append(args, packages...)

	cmd := exec.Command("pnpm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *PnpmInstaller) Run(script string) error {
	cmd := exec.Command("pnpm", "run", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *PnpmInstaller) Add(packages []string, isDev bool) error {
	return p.Install(packages, isDev)
}

type NpmInstaller struct{}

func (n *NpmInstaller) Install(packages []string, isDev bool) error {
	args := []string{"install"}
	if isDev {
		args = append(args, "--save-dev")
	}
	args = append(args, packages...)

	cmd := exec.Command("npm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (n *NpmInstaller) Run(script string) error {
	cmd := exec.Command("npm", "run", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (n *NpmInstaller) Add(packages []string, isDev bool) error {
	return n.Install(packages, isDev)
}

type TidyInstaller struct{}

func (t *TidyInstaller) Install(packages []string, isDev bool) error {
	return fmt.Errorf("tidy installer not yet implemented for specific packages")
}

func (t *TidyInstaller) Run(script string) error {
	return fmt.Errorf("use ScriptRunner for running scripts")
}

func (t *TidyInstaller) Add(packages []string, isDev bool) error {
	return t.Install(packages, isDev)
}
