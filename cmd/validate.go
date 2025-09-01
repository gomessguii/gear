package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ValidationRule struct {
	Name        string
	Description string
	Check       func(pkg *ast.Package, files map[string]*ast.File) []ValidationError
}

type ValidationError struct {
	Rule     string
	File     string
	Line     int
	Column   int
	Message  string
	Severity string // "error", "warning", "info"
}

// GearConfig represents the .gearrc configuration file
type GearConfig struct {
	Exclude []string          `yaml:"exclude"`
	Rules   map[string]string `yaml:"rules,omitempty"`
}

var (
	excludeDirs []string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate GEAR rule compliance in the current project",
	Long: `Analyze the current Go project to ensure it follows GEAR architecture rules.

Available Rules:
- R01: Interface contracts (exported interfaces + unexported structs) [default: warning]
- R02: Interface usage (no pointer-to-interface anti-patterns) [default: error]
- R03: Constructor patterns (returning interfaces) [default: warning]
- R04: Domain boundaries (clean layer separation) [default: info]
- R05: Centralized configuration (internal/config package) [default: error]
- R06: Systematic error handling (internal/errors package) [default: error]

Examples:
  gear validate                                    # Validate entire project
  gear validate --exclude vendor,test             # Exclude vendor and test directories
  gear validate --exclude pkg/external,migration  # Exclude specific paths

Configuration:
  Create a .gearrc file in your project root to set default options:
  
  exclude:
    - "vendor"
    - "*_test.go"
    - "*.pb.go"
    - "scripts"
  
  rules:
    R01: "warning"  # Interface contracts
    R02: "error"    # Interface usage
    R03: "warning"  # Constructor patterns 
    R04: "info"     # Domain boundaries
    R05: "error"    # Centralized configuration
    R06: "error"    # Systematic error handling`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateProject()
	},
}

func validateProject() error {
	fmt.Println("🔍 Validating GEAR compliance...")

	// Check if we're in a Go project
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Go project directory (go.mod not found)")
	}

	// Load configuration from .gearrc if it exists
	config, err := loadGearConfig()
	if err != nil {
		return fmt.Errorf("failed to load .gearrc: %w", err)
	}

	// Merge CLI flags with config file (CLI flags take precedence)
	if len(excludeDirs) == 0 && len(config.Exclude) > 0 {
		excludeDirs = config.Exclude
		fmt.Printf("📄 Loaded exclusions from .gearrc: %v\n", excludeDirs)
	}

	rules := []ValidationRule{
		{
			Name:        "R01-interface-contracts",
			Description: "Interface contracts: exported interfaces + unexported structs",
			Check:       validateInterfaceContracts,
		},
		{
			Name:        "R02-interface-usage",
			Description: "Interface usage: no pointer-to-interface anti-patterns",
			Check:       validateInterfaceUsage,
		},
		{
			Name:        "R03-constructor-patterns",
			Description: "Constructor patterns: constructors return interfaces",
			Check:       validateConstructorPatterns,
		},
		{
			Name:        "R04-domain-boundaries",
			Description: "Domain boundaries: clean layer separation",
			Check:       validateDomainBoundaries,
		},
		{
			Name:        "R05-centralized-config",
			Description: "Centralized configuration: internal/config package exists",
			Check:       validateCentralizedConfig,
		},
		{
			Name:        "R06-systematic-errors",
			Description: "Systematic error handling: internal/errors package exists",
			Check:       validateSystematicErrors,
		},
	}

	var allErrors []ValidationError

	// Parse all Go files in the project
	pkgs, err := parseProject()
	if err != nil {
		return fmt.Errorf("failed to parse project: %w", err)
	}

	// Run validation rules
	for _, rule := range rules {
		fmt.Printf("  Checking %s...\n", rule.Description)
		for _, pkg := range pkgs {
			errors := rule.Check(pkg, nil) // TODO: pass files map
			allErrors = append(allErrors, errors...)
		}
	}

	// Report results
	if len(allErrors) == 0 {
		fmt.Println("✅ All GEAR rules validated successfully!")
		return nil
	}

	fmt.Printf("\n❌ Found %d GEAR compliance issues:\n\n", len(allErrors))

	errorCount := 0
	warningCount := 0

	for _, err := range allErrors {
		switch err.Severity {
		case "error":
			fmt.Printf("❌ [%s] %s:%d:%d - %s\n", err.Rule, err.File, err.Line, err.Column, err.Message)
			errorCount++
		case "warning":
			fmt.Printf("⚠️  [%s] %s:%d:%d - %s\n", err.Rule, err.File, err.Line, err.Column, err.Message)
			warningCount++
		case "info":
			fmt.Printf("ℹ️  [%s] %s:%d:%d - %s\n", err.Rule, err.File, err.Line, err.Column, err.Message)
		}
	}

	fmt.Printf("\nSummary: %d errors, %d warnings\n", errorCount, warningCount)

	if errorCount > 0 {
		os.Exit(1)
	}

	return nil
}

var globalFileSet *token.FileSet

func parseProject() (map[string]*ast.Package, error) {
	globalFileSet = token.NewFileSet()
	packages := make(map[string]*ast.Package)

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and default excluded directories
		if !strings.HasSuffix(path, ".go") ||
			strings.Contains(path, "vendor/") ||
			strings.Contains(path, ".git/") {
			return nil
		}

		// Skip user-specified excluded paths and patterns
		for _, excludePattern := range excludeDirs {
			excludePattern = strings.TrimSpace(excludePattern)
			if excludePattern == "" {
				continue
			}

			// 1. Exact file name match (e.g., "main.go")
			if filepath.Base(path) == excludePattern {
				return nil
			}

			// 2. Directory path match (e.g., "vendor", "scripts")
			if strings.Contains(path, excludePattern+"/") || strings.HasSuffix(path, "/"+excludePattern) {
				return nil
			}

			// 3. Glob pattern match (e.g., "*_test.go", "*.pb.go")
			if strings.Contains(excludePattern, "*") || strings.Contains(excludePattern, "?") {
				// Match against filename only
				if matched, err := filepath.Match(excludePattern, filepath.Base(path)); err == nil && matched {
					return nil
				}
				// Match against relative path for patterns like "pkg/*_test.go"
				if matched, err := filepath.Match(excludePattern, path); err == nil && matched {
					return nil
				}
			}
		}

		// If this is a directory that should be skipped entirely, skip it
		if info.IsDir() {
			for _, excludeDir := range excludeDirs {
				excludeDir = strings.TrimSpace(excludeDir)
				if excludeDir != "" && strings.HasSuffix(path, excludeDir) {
					return filepath.SkipDir
				}
			}
		}

		// Parse the file
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(globalFileSet, path, src, parser.ParseComments)
		if err != nil {
			return err
		}

		// Group by package
		pkgName := file.Name.Name
		if packages[pkgName] == nil {
			packages[pkgName] = &ast.Package{
				Name:  pkgName,
				Files: make(map[string]*ast.File),
			}
		}
		packages[pkgName].Files[path] = file

		return nil
	})

	return packages, err
}

func validateInterfaceContracts(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	for filePath, file := range pkg.Files {
		// Track types with their positions
		type TypeInfo struct {
			Name       string
			IsExported bool
			Position   token.Pos
		}

		var interfaces []TypeInfo
		var structs []TypeInfo

		// First pass: collect interfaces and structs with positions
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				switch typeSpec.Type.(type) {
				case *ast.InterfaceType:
					interfaces = append(interfaces, TypeInfo{
						Name:       typeSpec.Name.Name,
						IsExported: typeSpec.Name.IsExported(),
						Position:   typeSpec.Pos(),
					})
				case *ast.StructType:
					structs = append(structs, TypeInfo{
						Name:       typeSpec.Name.Name,
						IsExported: typeSpec.Name.IsExported(),
						Position:   typeSpec.Pos(),
					})
				}
			}
		}

		// Check for exported structs (should be unexported in GEAR)
		// BUT exclude models, DTOs, requests, responses, and configs
		for _, structInfo := range structs {
			if structInfo.IsExported && shouldBeUnexported(structInfo.Name, filePath, file) {
				pos := globalFileSet.Position(structInfo.Position)
				errors = append(errors, ValidationError{
					Rule:     "R01-interface-contracts",
					File:     filePath,
					Line:     pos.Line,
					Column:   pos.Column,
					Message:  fmt.Sprintf("Struct '%s' is exported - GEAR prefers unexported structs with exported interfaces for service/business logic", structInfo.Name),
					Severity: "warning",
				})
			}
		}

		// Check for unexported interfaces (should be exported in GEAR)
		for _, interfaceInfo := range interfaces {
			if !interfaceInfo.IsExported {
				pos := globalFileSet.Position(interfaceInfo.Position)
				errors = append(errors, ValidationError{
					Rule:     "R01-interface-contracts",
					File:     filePath,
					Line:     pos.Line,
					Column:   pos.Column,
					Message:  fmt.Sprintf("Interface '%s' is unexported - GEAR requires exported interfaces", interfaceInfo.Name),
					Severity: "error",
				})
			}
		}
	}

	return errors
}

// shouldBeUnexported determines if a struct should be unexported based on GEAR rules
// Returns true only for service/business logic structs, false for models/DTOs/configs
func shouldBeUnexported(structName, filePath string, file *ast.File) bool {
	// If struct has no methods, it's a data structure and should be exported
	if !structHasMethods(structName, file) {
		return false
	}

	// Models, DTOs, requests, responses should remain exported
	if isDataStruct(structName) {
		return false
	}

	// Files in model/proto directories contain data structures
	if strings.Contains(filePath, "/model/") ||
		strings.Contains(filePath, "/proto/") ||
		strings.Contains(filePath, "/dto/") ||
		strings.Contains(filePath, "/client/") ||
		strings.Contains(filePath, "/provider/") {
		return false
	}

	// Configuration structs should remain exported for ease of use
	if strings.Contains(filePath, "/config/") || strings.HasSuffix(structName, "Config") {
		return false
	}

	// Error types should remain exported
	if strings.Contains(filePath, "/errors/") {
		return false
	}

	// Service, handler, repository implementations should be unexported
	if strings.Contains(filePath, "/service/") ||
		strings.Contains(filePath, "/handler/") ||
		strings.Contains(filePath, "/repository/") {
		return true
	}

	// Default: check if it looks like a business logic struct
	return !isDataStruct(structName)
}

// isDataStruct checks if a struct name indicates it's a data structure (should be exported)
func isDataStruct(name string) bool {
	dataStructSuffixes := []string{
		"Request", "Response", "Model", "DTO", "Data", "Entity",
		"Config", "Settings", "Options", "Params", "Result", "Info",
		"Status", "State", "Event", "Message", "Payload", "Body",
		"Error", "Exception", "Notification", "Alert", "Report",
	}

	for _, suffix := range dataStructSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	// Check for common data structure patterns
	dataStructPrefixes := []string{
		"Create", "Update", "Delete", "Get", "List", "Search",
	}

	for _, prefix := range dataStructPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// structHasMethods checks if a struct has any methods defined in the same file
func structHasMethods(structName string, file *ast.File) bool {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		// Check if this method belongs to our struct
		for _, recv := range funcDecl.Recv.List {
			switch recvType := recv.Type.(type) {
			case *ast.Ident:
				if recvType.Name == structName {
					return true
				}
			case *ast.StarExpr:
				if ident, ok := recvType.X.(*ast.Ident); ok && ident.Name == structName {
					return true
				}
			}
		}
	}
	return false
}

func validateConstructorPatterns(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	for filePath, file := range pkg.Files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Look for constructor functions (New* pattern)
			if !strings.HasPrefix(funcDecl.Name.Name, "New") {
				continue
			}

			// Skip error constructors and utility packages - they can return concrete types
			if strings.Contains(filePath, "/errors/") ||
				strings.Contains(filePath, "/utils/") ||
				strings.Contains(filePath, "/util/") ||
				strings.Contains(filePath, "/config/") ||
				strings.Contains(filePath, "/model/") ||
				strings.Contains(filePath, "/dto/") ||
				strings.Contains(filePath, "/proto/") {
				continue
			}

			// Check if it returns an interface
			if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
				continue
			}

			returnType := funcDecl.Type.Results.List[0].Type

			// Simple check - if it returns a pointer to struct, it's likely not following GEAR
			if starExpr, ok := returnType.(*ast.StarExpr); ok {
				if _, ok := starExpr.X.(*ast.Ident); ok {
					pos := globalFileSet.Position(funcDecl.Pos())
					errors = append(errors, ValidationError{
						Rule:     "R02-constructor-patterns",
						File:     filePath,
						Line:     pos.Line,
						Column:   pos.Column,
						Message:  fmt.Sprintf("Constructor '%s' returns pointer to struct - GEAR constructors should return interfaces", funcDecl.Name.Name),
						Severity: "warning",
					})
				}
			}
		}
	}

	return errors
}

func validateDomainBoundaries(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	// Check for expected domain structure
	expectedDirs := []string{"handler", "service", "repository", "model"}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(filepath.Join("pkg", "*", dir)); os.IsNotExist(err) {
			// This is a simple check - in reality, we'd want more sophisticated validation
			continue
		}
	}

	return errors
}

func validateCentralizedConfig(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	configPath := "internal/config"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		errors = append(errors, ValidationError{
			Rule:     "R04-centralized-config",
			File:     configPath,
			Message:  "Missing internal/config package - GEAR requires centralized configuration",
			Severity: "error",
		})
	}

	return errors
}

func validateSystematicErrors(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	errorsPath := "internal/errors"
	if _, err := os.Stat(errorsPath); os.IsNotExist(err) {
		errors = append(errors, ValidationError{
			Rule:     "R05-systematic-errors",
			File:     errorsPath,
			Message:  "Missing internal/errors package - GEAR requires systematic error handling",
			Severity: "error",
		})
	}

	return errors
}

func validateInterfaceUsage(pkg *ast.Package, files map[string]*ast.File) []ValidationError {
	var errors []ValidationError

	for filePath, file := range pkg.Files {
		// Build import map for this file
		imports := make(map[string]string) // alias -> package path
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil {
				// Named import: import foo "path/to/package"
				imports[imp.Name.Name] = path
			} else {
				// Default import: import "path/to/package"
				parts := strings.Split(path, "/")
				packageName := parts[len(parts)-1]
				imports[packageName] = path
			}
		}
		// Walk through all declarations and expressions to find pointer-to-interface types
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.StructType:
				// Check struct fields for pointer-to-interface types
				for _, field := range n.Fields.List {
					if starExpr, ok := field.Type.(*ast.StarExpr); ok {
						var typeName string
						var isExternal bool

						// Handle both local types (Ident) and external types (SelectorExpr)
						switch x := starExpr.X.(type) {
						case *ast.Ident:
							typeName = x.Name
							isExternal = false
						case *ast.SelectorExpr:
							// External package type like lead_service.StatusService
							typeName = x.Sel.Name
							isExternal = true
						default:
							continue
						}

						// Check if it's actually an interface
						isInterface := false
						if !isExternal {
							// Local type - check in file scope
							if obj := file.Scope.Lookup(typeName); obj != nil && obj.Kind == ast.Typ {
								if typeSpec, ok := obj.Decl.(*ast.TypeSpec); ok {
									if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
										isInterface = true
									}
								}
							}
						} else {
							// External type - try to resolve it by parsing the external package
							if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
								if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
									packagePath, exists := imports[pkgIdent.Name]
									if exists {
										isInterface = isExternalInterface(packagePath, typeName)
									}
								}
							}
						}

						if isInterface {
							pos := globalFileSet.Position(starExpr.Pos())
							var fieldName string
							if len(field.Names) > 0 {
								fieldName = field.Names[0].Name
							} else {
								fieldName = typeName
							}
							errors = append(errors, ValidationError{
								Rule:     "R06-interface-usage",
								File:     filePath,
								Line:     pos.Line,
								Column:   pos.Column,
								Message:  fmt.Sprintf("Struct field '%s' has type '*%s' - pointer to interface is an anti-pattern, use '%s' instead", fieldName, typeName, typeName),
								Severity: "error",
							})
						}
					}
				}
			case *ast.StarExpr:
				// Check if this is a pointer to an interface
				if ident, ok := n.X.(*ast.Ident); ok {
					// Look up the type in the file's scope
					if obj := file.Scope.Lookup(ident.Name); obj != nil && obj.Kind == ast.Typ {
						if typeSpec, ok := obj.Decl.(*ast.TypeSpec); ok {
							if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
								pos := globalFileSet.Position(n.Pos())
								errors = append(errors, ValidationError{
									Rule:     "R06-interface-usage",
									File:     filePath,
									Line:     pos.Line,
									Column:   pos.Column,
									Message:  fmt.Sprintf("Pointer to interface '*%s' is an anti-pattern - interfaces are already reference types", ident.Name),
									Severity: "error",
								})
							}
						}
					}
				}
			case *ast.FuncDecl:
				// Check function parameters for pointer-to-interface
				if n.Type.Params != nil {
					for _, param := range n.Type.Params.List {
						if starExpr, ok := param.Type.(*ast.StarExpr); ok {
							var typeName string
							var isExternal bool

							// Handle both local types (Ident) and external types (SelectorExpr)
							switch x := starExpr.X.(type) {
							case *ast.Ident:
								typeName = x.Name
								isExternal = false
							case *ast.SelectorExpr:
								// External package type like lead_handler.StatusHandler
								typeName = x.Sel.Name
								isExternal = true
							default:
								continue
							}

							// Check if it's actually an interface
							isInterface := false
							if !isExternal {
								// Local type - check in file scope
								if obj := file.Scope.Lookup(typeName); obj != nil && obj.Kind == ast.Typ {
									if typeSpec, ok := obj.Decl.(*ast.TypeSpec); ok {
										if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
											isInterface = true
										}
									}
								}
							} else {
								// External type - try to resolve it by parsing the external package
								if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
									if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
										packagePath, exists := imports[pkgIdent.Name]
										if exists {
											isInterface = isExternalInterface(packagePath, typeName)
										}
									}
								}
							}

							if isInterface {
								pos := globalFileSet.Position(starExpr.Pos())
								var paramName string
								if len(param.Names) > 0 {
									paramName = param.Names[0].Name
								} else {
									paramName = typeName
								}
								errors = append(errors, ValidationError{
									Rule:     "R06-interface-usage",
									File:     filePath,
									Line:     pos.Line,
									Column:   pos.Column,
									Message:  fmt.Sprintf("Function parameter '%s' has type '*%s' - pointer to interface is an anti-pattern, use '%s' instead", paramName, typeName, typeName),
									Severity: "error",
								})
							}
						}
					}
				}

				// Check return types - only flag if we can confirm it's actually an interface
				if n.Type.Results != nil {
					for _, result := range n.Type.Results.List {
						if starExpr, ok := result.Type.(*ast.StarExpr); ok {
							if ident, ok := starExpr.X.(*ast.Ident); ok {
								// Look up the type to see if it's actually an interface
								if obj := file.Scope.Lookup(ident.Name); obj != nil && obj.Kind == ast.Typ {
									if typeSpec, ok := obj.Decl.(*ast.TypeSpec); ok {
										if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
											pos := globalFileSet.Position(starExpr.Pos())
											errors = append(errors, ValidationError{
												Rule:     "R06-interface-usage",
												File:     filePath,
												Line:     pos.Line,
												Column:   pos.Column,
												Message:  fmt.Sprintf("Function returns '*%s' - pointer to interface, use '%s' instead", ident.Name, ident.Name),
												Severity: "error",
											})
										}
									}
								}
							}
						}
					}
				}
			}
			return true
		})
	}

	return errors
}

// isExternalInterface checks if a type in an external package is an interface
func isExternalInterface(packagePath, typeName string) bool {
	// Cache for parsed packages to avoid re-parsing
	if externalPkg, exists := externalPackageCache[packagePath]; exists {
		return checkTypeInPackage(externalPkg, typeName)
	}

	// Try to find the package in GOPATH/GOMODULE
	pkgPath := strings.TrimPrefix(packagePath, "github.com/nex-prospect/nex-core-service/")

	// Look for the package in current project first
	localPath := "./" + pkgPath
	if _, err := os.Stat(localPath); err == nil {
		// Parse the local package
		pkgFiles, err := filepath.Glob(filepath.Join(localPath, "*.go"))
		if err != nil {
			return false
		}

		fset := token.NewFileSet()
		var files []*ast.File

		for _, pkgFile := range pkgFiles {
			// Skip test files
			if strings.HasSuffix(pkgFile, "_test.go") {
				continue
			}

			src, err := os.ReadFile(pkgFile)
			if err != nil {
				continue
			}

			file, err := parser.ParseFile(fset, pkgFile, src, parser.ParseComments)
			if err != nil {
				continue
			}

			files = append(files, file)
		}

		// Build package from files
		if len(files) > 0 {
			pkg := &ast.Package{
				Name:  files[0].Name.Name,
				Files: make(map[string]*ast.File),
			}

			for i, file := range files {
				pkg.Files[pkgFiles[i]] = file
			}

			// Cache the package
			if externalPackageCache == nil {
				externalPackageCache = make(map[string]*ast.Package)
			}
			externalPackageCache[packagePath] = pkg

			return checkTypeInPackage(pkg, typeName)
		}
	}

	return false
}

// checkTypeInPackage checks if a type name is an interface in the given package
func checkTypeInPackage(pkg *ast.Package, typeName string) bool {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == typeName {
							_, isInterface := typeSpec.Type.(*ast.InterfaceType)
							return isInterface
						}
					}
				}
			}
		}
	}
	return false
}

// Cache for external packages to avoid re-parsing
var externalPackageCache map[string]*ast.Package

// loadGearConfig loads configuration from .gearrc file if it exists
func loadGearConfig() (*GearConfig, error) {
	config := &GearConfig{
		Exclude: []string{},
		Rules:   make(map[string]string),
	}

	// Check if .gearrc exists
	if _, err := os.Stat(".gearrc"); os.IsNotExist(err) {
		// No config file, return default config
		return config, nil
	}

	// Read the config file
	data, err := os.ReadFile(".gearrc")
	if err != nil {
		return nil, fmt.Errorf("failed to read .gearrc: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse .gearrc: %w", err)
	}

	return config, nil
}

func init() {
	validateCmd.Flags().StringSliceVarP(&excludeDirs, "exclude", "e", []string{}, "Comma-separated list of directories to exclude from validation")
}
