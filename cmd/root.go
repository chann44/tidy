package cmd

import (
	"fmt"
	"os"

	"github.com/chann44/tidy/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const Version = "0.1.0"

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "tidy",
	Short: "🧹 Tidy - A modern package manager for Node.js",
	Long: `Tidy is a fast, intelligent package manager that can:
  • Install dependencies from package.json
  • Scan your codebase and auto-detect packages
  • Run scripts from package.json
  • Support multiple package managers (Bun, pnpm, npm)
  
When run without arguments, Tidy launches an interactive UI.`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output")
}

func IsVerbose() bool {
	return verbose
}

func IsQuiet() bool {
	return quiet
}
