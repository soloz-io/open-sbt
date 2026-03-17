package interfaces

import (
	"context"

	"github.com/soloz-io/open-sbt/pkg/models"
)

// IApplicationPlaneUtils provides utility functions for Application Plane operations
type IApplicationPlaneUtils interface {
	// Resource Validation
	ValidateResourceConfig(ctx context.Context, spec models.ResourceSpec) error
	ValidateResourceDependencies(ctx context.Context, specs []models.ResourceSpec) error

	// Resource Naming
	GenerateResourceName(tenantID string, resourceType string) string

	// Resource Requirements
	CalculateResourceRequirements(ctx context.Context, tier string) ([]models.ResourceSpec, error)

	// Template Generation
	GenerateResourceTemplate(ctx context.Context, resourceType string, params map[string]interface{}) (string, error)

	// Cost Estimation
	EstimateResourceCost(ctx context.Context, specs []models.ResourceSpec) (float64, error)

	// Health Checks
	CheckResourceHealth(ctx context.Context, tenantID string, resourceID string) (string, error)

	// Batch Operations
	BatchProvisionResources(ctx context.Context, requests []models.ProvisionRequest) ([]models.ProvisionResult, error)
}
