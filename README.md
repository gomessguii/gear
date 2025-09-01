# GEAR - Go Engineering Architecture Rules

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **GEAR** is an opinionated architecture framework for building maintainable, scalable Go systems through enforceable engineering architecture rules.

Created by [@gomessguii](https://github.com/gomessguii)

## 🎯 What is GEAR?

GEAR (Go Engineering Architecture Rules) bridges the gap between traditional software architecture principles and Go's pragmatic characteristics. It provides a systematic, rule-based approach to Go development that ensures consistency from individual components to enterprise-wide systems.

### Core Principles

- **Interface-first encapsulation** - Exported interfaces with unexported implementations
- **Constructor patterns** - Constructors that return interfaces for compile-time contracts
- **Clean domain boundaries** - Systematic layer separation and dependency management
- **Centralized configuration** - Standardized environment variable handling
- **Systematic error handling** - Consistent, i18n-ready error management

## 🚀 Quick Start

### Installation

```bash
go install github.com/gomessguii/gear@latest
```

### Initialize a new GEAR project

```bash
gear init my-project
cd my-project
```

### Add a domain to your project

```bash
gear add-domain user
```

### Validate GEAR compliance

```bash
gear validate
```

## 📋 Available Rules

GEAR enforces 6 core architecture rules:

- **R01**: Interface contracts (exported interfaces + unexported structs)
- **R02**: Interface usage (no pointer-to-interface anti-patterns)  
- **R03**: Constructor patterns (constructors return interfaces)
- **R04**: Domain boundaries (clean layer separation)
- **R05**: Centralized configuration (internal/config package)
- **R06**: Systematic error handling (internal/errors package)

## ⚙️ Configuration

Create a `.gearrc` file in your project root to customize validation:

```yaml
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
  R06: "error"    # Systematic error handling
```

## 🛠️ Commands

### `gear init <project-name>`

Initialize a new GEAR-compliant Go project with:
- Go module setup
- Basic project structure
- Centralized configuration package
- Systematic error handling
- Sample Makefile

### `gear add-domain <domain-name>`

Add a new domain following GEAR patterns:
- Model (data structures)
- Repository (data access interface)
- Service (business logic interface)  
- Handler (HTTP interface)

### `gear validate`

Validate your project against all GEAR rules:
- Real-time feedback with line numbers
- Configurable rule severity
- Exclusion patterns support
- Comprehensive error reporting

**Options:**
- `--exclude strings` - Exclude directories/patterns from validation

## 📁 Project Structure

GEAR projects follow this structure:

```
my-project/
├── .gearrc                     # GEAR configuration
├── go.mod
├── main.go
├── Makefile
├── internal/
│   ├── config/                 # Centralized configuration
│   │   └── config.go
│   └── errors/                 # Systematic error handling
│       └── errors.go
└── pkg/
    └── user/                   # Domain example
        ├── model/
        │   └── user.go
        ├── repository/
        │   └── user_repository.go
        ├── service/
        │   └── user_service.go
        └── handler/
            └── user_handler.go
```

## 🎨 Code Examples

### Interface Contracts (R01)
```go
// ✅ GEAR compliant
type UserService interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
}

type userService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) UserService {
    return &userService{repo: repo}
}
```

### Constructor Patterns (R03)
```go
// ✅ Returns interface
func NewUserService(repo UserRepository) UserService {
    return &userService{repo: repo}
}

// ❌ Returns concrete type
func NewUserService(repo UserRepository) *userService {
    return &userService{repo: repo}
}
```

### Interface Usage (R02)
```go
// ✅ Interface without pointer
type Handler struct {
    userService UserService
}

// ❌ Pointer to interface (anti-pattern)
type Handler struct {
    userService *UserService
}
```

## 🧪 Testing

GEAR encourages testable code through interface-based design:

```go
func TestUserService_Create(t *testing.T) {
    // Mock repository implements UserRepository interface
    mockRepo := &MockUserRepository{}
    service := NewUserService(mockRepo)
    
    // Test business logic without database dependencies
    err := service.Create(context.Background(), &User{Name: "John"})
    assert.NoError(t, err)
}
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by Clean Architecture principles
- Built with Go's pragmatic design philosophy
- Influenced by Domain-Driven Design concepts

---

**Built with ❤️ by [@gomessguii](https://github.com/gomessguii)**
