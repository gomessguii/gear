package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	projectName  string
	moduleName   string
	webHandler   string
	orm          string
	includeTests bool
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new GEAR-compliant Go project",
	Long: `Initialize a new Go project following GEAR architecture rules.

Creates a complete project structure with:
- Clean architecture layers (handler/service/repository/model)
- Interface-first encapsulation
- Centralized configuration
- Systematic error handling
- Optional web framework and ORM integration`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName = args[0]

		if moduleName == "" {
			moduleName = projectName
		}

		return initializeProject()
	},
}

func init() {
	initCmd.Flags().StringVarP(&moduleName, "module", "m", "", "Go module name (defaults to project name)")
	initCmd.Flags().StringVar(&webHandler, "handler", "gin", "Web handler framework (gin|mux|fiber|echo)")
	initCmd.Flags().StringVar(&orm, "orm", "gorm", "ORM library (gorm|sqlx|ent)")
	initCmd.Flags().BoolVar(&includeTests, "tests", true, "Include test files and examples")
}

func initializeProject() error {
	fmt.Printf("🚀 Initializing GEAR project: %s\n", projectName)
	fmt.Printf("📦 Module: %s\n", moduleName)
	fmt.Printf("🌐 Handler: %s\n", webHandler)
	fmt.Printf("🗄️  ORM: %s\n", orm)

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create directory structure
	dirs := []string{
		"cmd",
		"internal/config",
		"internal/errors",
		"pkg",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectName, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	// Generate files
	if err := generateGoMod(); err != nil {
		return err
	}

	if err := generateMainFile(); err != nil {
		return err
	}

	if err := generateConfigPackage(); err != nil {
		return err
	}

	if err := generateErrorsPackage(); err != nil {
		return err
	}

	if err := generateMakefile(); err != nil {
		return err
	}

	fmt.Printf("✅ GEAR project %s created successfully!\n", projectName)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  gear add-domain user  # Add your first domain\n")
	fmt.Printf("  make run              # Start the application\n")

	return nil
}

func generateGoMod() error {
	content := fmt.Sprintf(`module %s

go 1.23.5

require (`, moduleName)

	if webHandler == "gin" {
		content += `
	github.com/gin-gonic/gin v1.9.1`
	}

	if orm == "gorm" {
		content += `
	gorm.io/gorm v1.25.7
	gorm.io/driver/postgres v1.5.6`
	}

	content += `
)
`

	return writeProjectFile("go.mod", content)
}

func generateMainFile() error {
	content := fmt.Sprintf(`package main

import (
	"log"

	"%s/internal/config"
)

func main() {
	cfg := config.NewConfig()

	log.Printf("Starting %%s on port %%s", cfg.AppName, cfg.Port)
	
	// TODO: Initialize your application here
	// server := server.New(cfg)
	// server.Start()
}
`, moduleName)

	return writeProjectFile("cmd/main.go", content)
}

func generateConfigPackage() error {
	content := fmt.Sprintf(`package config

import (
	"log"
	"os"
)

// Config holds all application configuration
type Config struct {
	// Private fields for sensitive data
	databaseURL string
	
	// Public fields for general configuration
	AppName     string
	Environment string
	Port        string
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		AppName:     getOrDefault("APP_NAME", "%s"),
		Environment: getOrDefault("ENVIRONMENT", "development"),
		Port:        getOrDefault("PORT", "8080"),
		databaseURL: getRequired("DATABASE_URL"),
	}
}

// GetDatabaseURL returns the database connection string
func (c *Config) GetDatabaseURL() string {
	return c.databaseURL
}

// getOrDefault gets environment variable with fallback to default value
func getOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getRequired gets environment variable and terminates program if empty
func getRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %%s is not set", key)
	}
	return value
}
`, projectName)

	return writeProjectFile("internal/config/config.go", content)
}

func generateErrorsPackage() error {
	content := `package errors

import "fmt"

// Error types are defined as constants
const (
	ErrInvalid      = "INVALID"
	ErrNotFound     = "NOT_FOUND"
	ErrUnauthorized = "UNAUTHORIZED"
	ErrForbidden    = "FORBIDDEN"
	ErrInternal     = "INTERNAL"
)

// Error represents a domain error with context
type Error struct {
	Code      string
	Message   string
	Variables map[string]string
	Err       error
}

// NewError creates a new error instance
func NewError(code string) *Error {
	return &Error{
		Code:      code,
		Variables: make(map[string]string),
	}
}

// WithVariables adds variables to the error context
func (e *Error) WithVariables(vars map[string]string) *Error {
	for k, v := range vars {
		e.Variables[k] = v
	}
	return e
}

// WithError wraps an underlying error
func (e *Error) WithError(err error) *Error {
	e.Err = err
	return e
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

// Predefined error instances
var (
	ErrInvalidInstance      = NewError(ErrInvalid)
	ErrNotFoundInstance     = NewError(ErrNotFound)
	ErrUnauthorizedInstance = NewError(ErrUnauthorized)
	ErrForbiddenInstance    = NewError(ErrForbidden)
	ErrInternalInstance     = NewError(ErrInternal)
)
`

	return writeProjectFile("internal/errors/errors.go", content)
}

func generateMakefile() error {
	content := `# GEAR Project Makefile

.PHONY: run build test clean deps lint

# Application
run:
	go run cmd/main.go

build:
	go build -o bin/app cmd/main.go

# Dependencies
deps:
	go mod download
	go mod tidy

# Testing
test:
	go test -v ./...

test-coverage:
	go test -v -cover ./...

# Linting
lint:
	golangci-lint run

# Cleanup
clean:
	rm -rf bin/
	go clean

# Development
dev: deps
	go run cmd/main.go

# Docker (optional)
docker-build:
	docker build -t ` + strings.ToLower(projectName) + ` .

docker-run:
	docker run -p 8080:8080 ` + strings.ToLower(projectName) + `
`

	return writeProjectFile("Makefile", content)
}

func writeProjectFile(fileName, content string) error {
	filePath := filepath.Join(projectName, fileName)
	return writeFile(filePath, content)
}
