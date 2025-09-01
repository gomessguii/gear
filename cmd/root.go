package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gear",
	Short: "GEAR CLI - Go Engineering Architecture Rules",
	Long: `
┌──────────────────────────────────────────────────────────┐
│                                                          │
│              ██████  ███████  █████  ██████              │
│             ██       ██      ██   ██ ██   ██             │
│             ██   ███ █████   ███████ ██████              │
│             ██    ██ ██      ██   ██ ██   ██             │
│              ██████  ███████ ██   ██ ██   ██             │
│                                                          │
│            Go Engineering Architecture Rules             │
│                                           by @gomessguii │
│                                                   v0.0.1 │
└──────────────────────────────────────────────────────────┘

GEAR CLI is a command-line tool for creating and managing Go projects
that follow the GEAR (Go Engineering Architecture Rules) opinionated architecture framework.

Available Rules:
- R01: Interface contracts (exported interfaces + unexported structs) [default: warning]
- R02: Interface usage (no pointer-to-interface anti-patterns) [default: error]
- R03: Constructor patterns (returning interfaces) [default: warning]
- R04: Domain boundaries (clean layer separation) [default: info]
- R05: Centralized configuration (internal/config package) [default: error]
- R06: Systematic error handling (internal/errors package) [default: error]`,
	Version: "0.0.1",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addDomainCmd)
	rootCmd.AddCommand(validateCmd)
}
