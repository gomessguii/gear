package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var addDomainCmd = &cobra.Command{
	Use:   "add-domain [domain-name]",
	Short: "Add a new domain to the GEAR project",
	Long: `Add a new domain following GEAR architecture patterns.

Creates a complete domain structure with:
- Interface definitions
- Service implementation (unexported struct)
- Repository interface and implementation
- Model definitions with response objects
- Handler with route registration
- Optional test files`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domainName := args[0]
		return addDomain(domainName)
	},
}

func addDomain(domainName string) error {
	fmt.Printf("🏗️  Adding domain: %s\n", domainName)

	// Validate we're in a GEAR project
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Go project directory (go.mod not found)")
	}

	// Read module name from go.mod
	moduleName, err := getModuleName()
	if err != nil {
		return fmt.Errorf("failed to read module name: %w", err)
	}

	// Create domain directory structure
	domainPath := filepath.Join("pkg", domainName)
	dirs := []string{
		filepath.Join(domainPath, "handler"),
		filepath.Join(domainPath, "service"),
		filepath.Join(domainPath, "repository"),
		filepath.Join(domainPath, "model"),
	}

	if includeTests {
		dirs = append(dirs,
			filepath.Join(domainPath, "service", "test"),
			filepath.Join(domainPath, "repository", "test"),
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate domain files
	if err := generateModel(domainName, moduleName); err != nil {
		return err
	}

	if err := generateRepository(domainName, moduleName); err != nil {
		return err
	}

	if err := generateService(domainName, moduleName); err != nil {
		return err
	}

	if err := generateHandler(domainName, moduleName); err != nil {
		return err
	}

	fmt.Printf("✅ Domain %s added successfully!\n", domainName)
	fmt.Printf("\nGenerated files:\n")
	fmt.Printf("  pkg/%s/model/%s.go\n", domainName, domainName)
	fmt.Printf("  pkg/%s/repository/%s_repository.go\n", domainName, domainName)
	fmt.Printf("  pkg/%s/service/%s_service.go\n", domainName, domainName)
	fmt.Printf("  pkg/%s/handler/%s_handler.go\n", domainName, domainName)

	return nil
}

func generateModel(domainName, moduleName string) error {
	structName := capitalize(domainName)

	content := fmt.Sprintf(`package model

import (
	"time"

	"github.com/google/uuid"
)

// %s represents the domain model for a %s
type %s struct {
	ID        uuid.UUID `+"`gorm:\"type:uuid;primary_key;default:gen_random_uuid()\" json:\"-\"`"+`
	Name      string    `+"`gorm:\"size:255;not null\" json:\"-\"`"+`
	CreatedAt time.Time `+"`json:\"-\"`"+`
	UpdatedAt time.Time `+"`json:\"-\"`"+`
}

// %sResponse represents the API response for a %s
type %sResponse struct {
	ID        uuid.UUID `+"`json:\"id\"`"+`
	Name      string    `+"`json:\"name\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\"`"+`
	UpdatedAt time.Time `+"`json:\"updated_at\"`"+`
}

// ToResponse converts a %s domain model to a %sResponse
func (u *%s) ToResponse() *%sResponse {
	return &%sResponse{
		ID:        u.ID,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
`, structName, domainName, structName, structName, domainName, structName, structName, structName, structName, structName, structName)

	fileName := filepath.Join("pkg", domainName, "model", domainName+".go")
	return writeFile(fileName, content)
}

func generateRepository(domainName, moduleName string) error {
	structName := capitalize(domainName)

	content := fmt.Sprintf(`package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"%s/pkg/%s/model"
)

// %sRepository defines the interface for %s data operations
type %sRepository interface {
	Create(ctx context.Context, %s model.%s) (*model.%s, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.%s, error)
	Update(ctx context.Context, %s *model.%s) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]model.%s, error)
}

type %sRepository struct {
	db *gorm.DB
}

// New%sRepository creates a new %s repository instance
func New%sRepository(db *gorm.DB) %sRepository {
	return &%sRepository{
		db: db,
	}
}

func (r *%sRepository) Create(ctx context.Context, %s model.%s) (*model.%s, error) {
	if err := r.db.WithContext(ctx).Create(&%s).Error; err != nil {
		return nil, err
	}
	return &%s, nil
}

func (r *%sRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.%s, error) {
	var %s model.%s
	err := r.db.WithContext(ctx).First(&%s, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &%s, nil
}

func (r *%sRepository) Update(ctx context.Context, %s *model.%s) error {
	return r.db.WithContext(ctx).Save(%s).Error
}

func (r *%sRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.%s{}, "id = ?", id).Error
}

func (r *%sRepository) List(ctx context.Context) ([]model.%s, error) {
	var %ss []model.%s
	err := r.db.WithContext(ctx).Find(&%ss).Error
	if err != nil {
		return nil, err
	}
	return %ss, nil
}
`, moduleName, domainName, structName, domainName, structName, domainName, structName, structName, structName, domainName, structName, structName, domainName, structName, structName, domainName, structName, structName, domainName, structName, domainName, structName, structName, domainName, structName, structName, structName, domainName, structName, structName, domainName, structName, domainName, structName, domainName, structName, structName, structName, domainName, structName, domainName, domainName, structName, domainName, domainName)

	fileName := filepath.Join("pkg", domainName, "repository", domainName+"_repository.go")
	return writeFile(fileName, content)
}

func generateService(domainName, moduleName string) error {
	structName := capitalize(domainName)

	content := fmt.Sprintf(`package service

import (
	"context"

	"github.com/google/uuid"

	"%s/internal/errors"
	"%s/pkg/%s/model"
	"%s/pkg/%s/repository"
)

// %sService defines the interface for %s operations
type %sService interface {
	Get%s(ctx context.Context, id uuid.UUID) (*model.%s, error)
	Create%s(ctx context.Context, %s model.%s) (*model.%s, error)
	Update%s(ctx context.Context, %s *model.%s) (*model.%s, error)
	Delete%s(ctx context.Context, id uuid.UUID) error
	List%ss(ctx context.Context) ([]model.%s, error)
}

type %sService struct {
	repo repository.%sRepository
}

// New%sService creates a new %s service instance
func New%sService(repo repository.%sRepository) %sService {
	return &%sService{
		repo: repo,
	}
}

func (s *%sService) Get%s(ctx context.Context, id uuid.UUID) (*model.%s, error) {
	%s, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.ErrInternalInstance.WithError(err)
	}
	return %s, nil
}

func (s *%sService) Create%s(ctx context.Context, %s model.%s) (*model.%s, error) {
	created%s, err := s.repo.Create(ctx, %s)
	if err != nil {
		return nil, errors.ErrInternalInstance.WithError(err)
	}
	return created%s, nil
}

func (s *%sService) Update%s(ctx context.Context, %s *model.%s) (*model.%s, error) {
	if err := s.repo.Update(ctx, %s); err != nil {
		return nil, errors.ErrInternalInstance.WithError(err)
	}
	return %s, nil
}

func (s *%sService) Delete%s(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.ErrInternalInstance.WithError(err)
	}
	return nil
}

func (s *%sService) List%ss(ctx context.Context) ([]model.%s, error) {
	%ss, err := s.repo.List(ctx)
	if err != nil {
		return nil, errors.ErrInternalInstance.WithError(err)
	}
	return %ss, nil
}
`, moduleName, domainName, moduleName, domainName, structName, domainName, structName, structName, structName, structName, domainName, structName, structName, structName, domainName, structName, structName, structName, structName, domainName, structName, structName, structName, structName, domainName, structName, structName, domainName, structName, domainName, structName, structName, structName, domainName, domainName, domainName, structName, structName, domainName, structName, structName, domainName, structName, structName, structName, domainName, structName, structName, domainName, structName, structName, structName, structName, domainName, domainName, domainName)

	fileName := filepath.Join("pkg", domainName, "service", domainName+"_service.go")
	return writeFile(fileName, content)
}

func generateHandler(domainName, moduleName string) error {
	structName := capitalize(domainName)

	content := fmt.Sprintf(`package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"%s/internal/errors"
	"%s/pkg/%s/model"
	"%s/pkg/%s/service"
)

// %sHandler handles HTTP requests for %s operations
type %sHandler interface {
	Get%s(c *gin.Context)
	Create%s(c *gin.Context)
	Update%s(c *gin.Context)
	Delete%s(c *gin.Context)
	List%ss(c *gin.Context)
	RegisterRoutes(router gin.IRouter)
}

type %sHandler struct {
	%sService service.%sService
}

// New%sHandler creates a new %s handler instance
func New%sHandler(%sService service.%sService) %sHandler {
	return &%sHandler{
		%sService: %sService,
	}
}

// RegisterRoutes registers all %s routes
func (h *%sHandler) RegisterRoutes(router gin.IRouter) {
	%sGroup := router.Group("/%ss")
	{
		%sGroup.GET("/:id", h.Get%s)
		%sGroup.POST("", h.Create%s)
		%sGroup.PUT("/:id", h.Update%s)
		%sGroup.DELETE("/:id", h.Delete%s)
		%sGroup.GET("", h.List%ss)
	}
}

// Get%s handles GET /%ss/:id requests
func (h *%sHandler) Get%s(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidInstance.WithVariables(map[string]string{
			"field": "id",
		}).WithError(err))
		return
	}

	%s, err := h.%sService.Get%s(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, %s.ToResponse())
}

// Create%s handles POST /%ss requests
func (h *%sHandler) Create%s(c *gin.Context) {
	var %s model.%s
	if err := c.ShouldBindJSON(&%s); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidInstance.WithVariables(map[string]string{
			"field": "request body",
		}).WithError(err))
		return
	}

	created%s, err := h.%sService.Create%s(c.Request.Context(), %s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusCreated, created%s.ToResponse())
}

// Update%s handles PUT /%ss/:id requests
func (h *%sHandler) Update%s(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidInstance.WithVariables(map[string]string{
			"field": "id",
		}).WithError(err))
		return
	}

	var %s model.%s
	if err := c.ShouldBindJSON(&%s); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidInstance.WithVariables(map[string]string{
			"field": "request body",
		}).WithError(err))
		return
	}

	%s.ID = id
	updated%s, err := h.%sService.Update%s(c.Request.Context(), &%s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, updated%s.ToResponse())
}

// Delete%s handles DELETE /%ss/:id requests
func (h *%sHandler) Delete%s(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidInstance.WithVariables(map[string]string{
			"field": "id",
		}).WithError(err))
		return
	}

	err = h.%sService.Delete%s(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// List%ss handles GET /%ss requests
func (h *%sHandler) List%ss(c *gin.Context) {
	%ss, err := h.%sService.List%ss(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var responses []*model.%sResponse
	for _, %s := range %ss {
		responses = append(responses, %s.ToResponse())
	}
	
	c.JSON(http.StatusOK, responses)
}
`, moduleName, domainName, moduleName, domainName, structName, domainName, structName, structName, structName, structName, structName, structName, domainName, structName, domainName, structName, structName, domainName, structName, domainName, structName, structName, domainName, structName, domainName, domainName, domainName, domainName, domainName, structName, domainName, structName, domainName, structName, domainName, structName, structName, domainName, domainName, structName, structName, domainName, domainName, structName, structName, domainName, structName, structName, domainName, structName, domainName, structName, domainName, structName, structName, structName, structName, structName, domainName, structName, structName, domainName, structName, structName, structName, domainName, structName, structName, domainName, structName, structName, structName, domainName, structName, domainName, structName, structName, structName, structName, domainName, structName, domainName, domainName, structName, structName, domainName, domainName, domainName, structName, domainName, domainName, structName, domainName, domainName)

	fileName := filepath.Join("pkg", domainName, "handler", domainName+"_handler.go")
	return writeFile(fileName, content)
}

func getModuleName() (string, error) {
	// Simple implementation - read first line of go.mod
	// In a real implementation, you'd want to parse this properly
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "module ") {
		return strings.TrimSpace(strings.TrimPrefix(lines[0], "module ")), nil
	}

	return "", fmt.Errorf("could not parse module name from go.mod")
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
