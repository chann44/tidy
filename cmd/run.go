package cmd
import (
	"fmt"
	"os"
	"github.com/chann44/tidy/internal"
	"github.com/spf13/cobra"
)
var runCmd = &cobra.Command{
	Use:   "run <script>",
	Short: "Run a script from package.json",
	Long: `Run a script defined in package.json.
Examples:
  tidy run dev              # Run the dev script
  tidy run build            # Run the build script
  tidy run test             # Run the test script`,
	Aliases: []string{"r"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptName := args[0]
		runScript(scriptName)
	},
}
func init() {
	rootCmd.AddCommand(runCmd)
}
func runScript(scriptName string) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		fmt.Println("❌ No package.json found")
		os.Exit(1)
	}
	jsn, err := internal.ReadJson(wd)
	if err != nil {
		fmt.Printf("Error reading package.json: %v\n", err)
		os.Exit(1)
	}
	runner := internal.NewScriptRunner(jsn)
	if !IsQuiet() {
		fmt.Printf("▶️  Running script: %s\n\n", scriptName)
	}
	if err := runner.Run(scriptName); err != nil {
		fmt.Printf("\n❌ Script failed: %v\n", err)
		os.Exit(1)
	}
	if !IsQuiet() {
		fmt.Printf("\n✅ Script completed successfully\n")
	}
}