package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate a .gearrc configuration file",
	Long: `Generate a .gearrc configuration file in the current directory.

The .gearrc file allows you to customize GEAR validation behavior:
- Set exclude patterns for files and directories
- Configure rule severities (error, warning, info)
- Persist settings across validation runs

Example .gearrc content:
  exclude:
    - "vendor"
    - "*_test.go" 
    - "*.pb.go"
  
  rules:
    R01: "error"    # Interface contracts
    R02: "warning"  # Interface usage
    R03: "info"     # Constructor patterns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateStandaloneGearRC()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func generateStandaloneGearRC() error {
	// Check if .gearrc already exists
	if _, err := os.Stat(".gearrc"); err == nil {
		fmt.Print("⚠️  .gearrc already exists. Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("❌ Cancelled")
			return nil
		}
	}

	content := `exclude:
  - "vendor"
  - "*_test.go"
  - "*.pb.go"
  - "scripts"
  - "docs"

rules:
  R01: "warning"  # Interface contracts (exported interfaces, unexported structs)
  R02: "error"    # Interface usage (no pointer-to-interface anti-patterns)
  R03: "warning"  # Constructor patterns (returning interfaces)
  R04: "info"     # Domain boundaries (clean layer separation)
  R05: "error"    # Centralized configuration (internal/config package)
  R06: "error"    # Systematic error handling (internal/errors package)
`

	if err := writeFile(".gearrc", content); err != nil {
		return fmt.Errorf("failed to create .gearrc: %w", err)
	}

	fmt.Println("✅ .gearrc configuration file created successfully!")
	fmt.Println("\nYou can now:")
	fmt.Println("  - Customize exclude patterns")
	fmt.Println("  - Adjust rule severities (error/warning/info)")
	fmt.Println("  - Run 'gear validate' to use your settings")

	return nil
}
