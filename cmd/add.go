package cmd
import (
	"fmt"
	"os"
	"github.com/chann44/tidy/internal"
	"github.com/spf13/cobra"
)
var (
	isDev   bool
	useGrep bool
)
var addCmd = &cobra.Command{
	Use:   "add [packages...]",
	Short: "Add packages to your project",
	Long: `Add packages to your project and install them.
Examples:
  tidy add react react-dom      # Add production dependencies
  tidy add -D typescript        # Add dev dependency
  tidy add -g                   # Scan codebase and add found packages`,
	Aliases: []string{"a"},
	Run: func(cmd *cobra.Command, args []string) {
		if useGrep {
			scanAndAddPackages()
		} else if len(args) > 0 {
			addSpecificPackages(args)
		} else {
			fmt.Println("Error: Please specify packages to add or use -g to scan codebase")
			cmd.Help()
			os.Exit(1)
		}
	},
}
func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&isDev, "dev", "D", false, "add as dev dependency")
	addCmd.Flags().BoolVarP(&useGrep, "grep", "g", false, "scan codebase and add found packages")
}
func scanAndAddPackages() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
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
	fmt.Println("🔍 Scanning codebase for packages...")
	internal.Grep(jsn)
	if _, err := os.Stat("package.json"); !os.IsNotExist(err) {
		jsn, _ = internal.ReadJson(wd)
	}
	fmt.Println("\n📦 Resolving dependencies...")
	resolved, err := internal.Resolve(jsn)
	if err != nil {
		fmt.Printf("Error resolving dependencies: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n📥 Installing packages...")
	installPackages(resolved)
}
func addSpecificPackages(packages []string) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
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
	depType := "production"
	if isDev {
		depType = "development"
		for _, pkg := range packages {
			jsn.DevDependencies[pkg] = "latest"
		}
	} else {
		for _, pkg := range packages {
			jsn.Dependencies[pkg] = "latest"
		}
	}
	fmt.Printf("Adding %d %s package(s)...\n", len(packages), depType)
	resolved, err := internal.Resolve(jsn)
	if err != nil {
		fmt.Printf("Error resolving dependencies: %v\n", err)
		os.Exit(1)
	}
	installPackages(resolved)
}