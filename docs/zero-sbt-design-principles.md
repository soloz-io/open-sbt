# Zero-SBT Design Principles

**Version:** 1.0  
**Date:** 2026-03-16  
**Based on:** AWS SaaS Builder Toolkit (SBT-AWS) Architecture

## Executive Summary

Zero-SBT is a Go-based SaaS builder toolkit for the zero-ops platform, inspired by AWS SBT-AWS. It provides reusable abstractions for building multi-tenant SaaS applications with a clear separation between Control Plane and Application Plane.

**Key Differences from SBT-AWS:**
- Language: Go (not TypeScript/CDK)
- Event Bus: NATS (not AWS EventBridge)
- Auth: Ory Stack (not AWS Cognito)
- Provisioning: Crossplane + ArgoCD + Argo Workflows (not CloudFormation + CodeBuild)
- Database: PostgreSQL with sqlc (not DynamoDB)
- API: Gin framework (not API Gateway + Lambda)

## Core Architecture Principles

### 1. Control Plane + Application Plane Separation

**Principle:** Maintain clear boundaries between tenant management (Control Plane) and tenant workloads (Application Plane).

**Control Plane Responsibilities:**
- Tenant lifecycle management (CRUD operations)
- Tenant registration and onboarding workflows
- User management per tenant
- Tenant configuration storage
- Authentication and authorization
- Billing and metering (optional)
- Event orchestration

**Application Plane Responsibilities:**
- Tenant resource provisioning
- Tenant workload execution
- Responding to Control Plane events
- Tenant-specific infrastructure management


**Communication Pattern:**
```
Control Plane → NATS Event → Application Plane
Application Plane → NATS Event → Control Plane (status updates)
```

### 2. Interface-Based Abstraction

**Principle:** Define stable Go interfaces for all pluggable components to allow implementation swapping without breaking user code.

**Core Interfaces:**

```go
// IAuth - Authentication and authorization provider
type IAuth interface {
    // User management
    CreateUser(ctx context.Context, user User) error
    GetUser(ctx context.Context, userID string) (*User, error)
    UpdateUser(ctx context.Context, userID string, updates UserUpdates) error
    DeleteUser(ctx context.Context, userID string) error
    DisableUser(ctx context.Context, userID string) error
    EnableUser(ctx context.Context, userID string) error
    ListUsers(ctx context.Context, filters UserFilters) ([]User, error)
    
    // Authentication
    AuthenticateUser(ctx context.Context, credentials Credentials) (*Token, error)
    ValidateToken(ctx context.Context, token string) (*Claims, error)
    RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
    
    // Admin user creation
    CreateAdminUser(ctx context.Context, props CreateAdminUserProps) error
    
    // Token configuration
    GetJWTIssuer() string
    GetJWTAudience() []string
    GetTokenEndpoint() string
    GetWellKnownEndpoint() string
}
```


```go
// IEventBus - Message bus for Control Plane ↔ Application Plane communication
type IEventBus interface {
    // Event publishing
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event) error
    
    // Event subscription
    Subscribe(ctx context.Context, eventType string, handler EventHandler) error
    SubscribeQueue(ctx context.Context, eventType string, queueGroup string, handler EventHandler) error
    
    // Event definitions
    GetControlPlaneEventSource() string
    GetApplicationPlaneEventSource() string
    CreateControlPlaneEvent(detailType string) EventDefinition
    CreateApplicationPlaneEvent(detailType string) EventDefinition
    CreateCustomEvent(detailType string, source string) EventDefinition
    
    // Standard events
    GetStandardEvents() map[string]EventDefinition
    
    // Permissions
    GrantPublishPermissions(grantee string) error
}

// IProvisioner - Tenant resource provisioning
type IProvisioner interface {
    // Provision tenant resources
    ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
    
    // Deprovision tenant resources
    DeprovisionTenant(ctx context.Context, req DeprovisionRequest) (*DeprovisionResult, error)
    
    // Get provisioning status
    GetProvisioningStatus(ctx context.Context, tenantID string) (*ProvisioningStatus, error)
    
    // Update tenant resources
    UpdateTenantResources(ctx context.Context, req UpdateRequest) (*UpdateResult, error)
}
```


```go
// IStorage - Data persistence layer
type IStorage interface {
    // Tenant management
    CreateTenant(ctx context.Context, tenant Tenant) error
    GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
    UpdateTenant(ctx context.Context, tenantID string, updates TenantUpdates) error
    DeleteTenant(ctx context.Context, tenantID string) error
    ListTenants(ctx context.Context, filters TenantFilters) ([]Tenant, error)
    
    // Tenant registration
    CreateTenantRegistration(ctx context.Context, reg TenantRegistration) error
    GetTenantRegistration(ctx context.Context, regID string) (*TenantRegistration, error)
    UpdateTenantRegistration(ctx context.Context, regID string, updates RegistrationUpdates) error
    DeleteTenantRegistration(ctx context.Context, regID string) error
    ListTenantRegistrations(ctx context.Context, filters RegistrationFilters) ([]TenantRegistration, error)
    
    // Tenant configuration
    SetTenantConfig(ctx context.Context, tenantID string, config map[string]interface{}) error
    GetTenantConfig(ctx context.Context, tenantID string) (map[string]interface{}, error)
    DeleteTenantConfig(ctx context.Context, tenantID string) error
}

// IBilling - Billing integration (optional)
type IBilling interface {
    // Customer management
    CreateCustomer(ctx context.Context, customer BillingCustomer) error
    DeleteCustomer(ctx context.Context, customerID string) error
    
    // Usage tracking
    RecordUsage(ctx context.Context, usage UsageRecord) error
    GetUsage(ctx context.Context, customerID string, period TimePeriod) (*UsageReport, error)
    
    // Webhook handling
    HandleWebhook(ctx context.Context, payload []byte) error
}
```


```go
// IMetering - Usage metering (optional)
type IMetering interface {
    // Meter management
    CreateMeter(ctx context.Context, meter Meter) error
    GetMeter(ctx context.Context, meterID string) (*Meter, error)
    UpdateMeter(ctx context.Context, meterID string, updates MeterUpdates) error
    DeleteMeter(ctx context.Context, meterID string) error
    ListMeters(ctx context.Context, filters MeterFilters) ([]Meter, error)
    
    // Usage ingestion
    IngestUsageEvent(ctx context.Context, event UsageEvent) error
    
    // Usage queries
    GetUsage(ctx context.Context, meterID string, period TimePeriod) (*UsageData, error)
    CancelUsageEvents(ctx context.Context, eventIDs []string) error
}
```

**Implementation Strategy:**
1. Define interfaces in `pkg/interfaces/` package
2. Provide default implementations in `pkg/providers/` package
3. Allow users to provide custom implementations
4. Use dependency injection for flexibility

**Example Usage:**
```go
// User can choose implementation
auth := ory.NewOryAuth(oryConfig)
// OR
auth := keycloak.NewKeycloakAuth(keycloakConfig)
// OR
auth := custom.NewCustomAuth(customConfig)

// Control plane works with any IAuth implementation
controlPlane := zerosbt.NewControlPlane(zerosbt.ControlPlaneConfig{
    Auth: auth,  // Interface, not concrete type
    EventBus: natsEventBus,
    Storage: postgresStorage,
})
```


### 3. Event-Driven Communication

**Principle:** Use NATS as the message bus for asynchronous communication between Control Plane and Application Plane.

**Standard Events:**

**Control Plane Events** (source: `zerosbt.control.plane`):
- `zerosbt_onboardingRequest` - Tenant onboarding initiated
- `zerosbt_offboardingRequest` - Tenant offboarding initiated
- `zerosbt_activateRequest` - Tenant activation requested
- `zerosbt_deactivateRequest` - Tenant deactivation requested
- `zerosbt_tenantUserCreated` - User created in tenant
- `zerosbt_tenantUserDeleted` - User deleted from tenant
- `zerosbt_billingSuccess` - Billing operation succeeded
- `zerosbt_billingFailure` - Billing operation failed

**Application Plane Events** (source: `zerosbt.application.plane`):
- `zerosbt_onboardingSuccess` - Tenant onboarded successfully
- `zerosbt_onboardingFailure` - Tenant onboarding failed
- `zerosbt_offboardingSuccess` - Tenant offboarded successfully
- `zerosbt_offboardingFailure` - Tenant offboarding failed
- `zerosbt_provisionSuccess` - Resources provisioned successfully
- `zerosbt_provisionFailure` - Resource provisioning failed
- `zerosbt_deprovisionSuccess` - Resources deprovisioned successfully
- `zerosbt_deprovisionFailure` - Resource deprovisioning failed
- `zerosbt_activateSuccess` - Tenant activated successfully
- `zerosbt_activateFailure` - Tenant activation failed
- `zerosbt_deactivateSuccess` - Tenant deactivated successfully
- `zerosbt_deactivateFailure` - Tenant deactivation failed
- `zerosbt_ingestUsage` - Usage data ingested


**Event Structure:**
```go
type Event struct {
    ID          string                 `json:"id"`
    Version     string                 `json:"version"`
    DetailType  string                 `json:"detailType"`
    Source      string                 `json:"source"`
    Time        time.Time              `json:"time"`
    Region      string                 `json:"region,omitempty"`
    Resources   []string               `json:"resources,omitempty"`
    Detail      map[string]interface{} `json:"detail"`
}

// Example: Onboarding Request Event
{
    "id": "6a7e8feb-b491-4cf7-a9f1-bf3703467718",
    "version": "1.0",
    "detailType": "zerosbt_onboardingRequest",
    "source": "zerosbt.control.plane",
    "time": "2026-03-16T18:43:48Z",
    "detail": {
        "tenantId": "e6878e03-ae2c-43ed-a863-08314487318b",
        "tier": "premium",
        "name": "acme-corp",
        "email": "admin@acme.com"
    }
}
```

**Event Flow Example:**
```
1. Control Plane: Tenant registration API called
2. Control Plane: Publishes zerosbt_onboardingRequest to NATS
3. Application Plane: Receives event, starts provisioning
4. Application Plane: Provisions resources (Crossplane + ArgoCD)
5. Application Plane: Publishes zerosbt_provisionSuccess to NATS
6. Control Plane: Receives event, updates tenant status
7. Control Plane: Returns success to API caller
```


### 4. Tenant Lifecycle Management

**Principle:** Provide standardized workflows for tenant onboarding, management, and offboarding.

**Tenant States:**
- `pending` - Registration created, awaiting provisioning
- `provisioning` - Resources being provisioned
- `active` - Tenant fully operational
- `suspended` - Tenant temporarily disabled
- `deprovisioning` - Resources being removed
- `deleted` - Tenant removed

**Tenant Registration Workflow:**
```
1. POST /tenant-registrations
   → Creates registration record (status: pending)
   → Publishes zerosbt_onboardingRequest event
   
2. Application Plane receives event
   → Provisions resources (Crossplane + Argo Workflows)
   → Publishes zerosbt_provisionSuccess event
   
3. Control Plane receives success event
   → Updates registration (status: active)
   → Creates tenant record
   → Returns tenant details to caller
```

**Tenant Offboarding Workflow:**
```
1. DELETE /tenants/{tenantId}
   → Updates tenant (status: deprovisioning)
   → Publishes zerosbt_offboardingRequest event
   
2. Application Plane receives event
   → Deprovisions resources
   → Publishes zerosbt_deprovisionSuccess event
   
3. Control Plane receives success event
   → Deletes tenant record
   → Deletes registration record
   → Returns success to caller
```


### 5. Provisioning Abstraction

**Principle:** Abstract tenant resource provisioning to support multiple provisioning engines.

**Provisioning Strategies:**

**Strategy 1: Crossplane Compositions**
```go
type CrossplaneProvisioner struct {
    client client.Client
    config CrossplaneConfig
}

func (p *CrossplaneProvisioner) ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    // Create Crossplane Composition
    composition := &compositionv1.Composition{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("tenant-%s", req.TenantID),
        },
        Spec: compositionv1.CompositionSpec{
            Resources: []compositionv1.ComposedTemplate{
                // Namespace
                {Resource: createNamespaceTemplate(req)},
                // PostgreSQL Database
                {Resource: createDatabaseTemplate(req)},
                // S3 Bucket
                {Resource: createS3BucketTemplate(req)},
            },
        },
    }
    
    return p.client.Create(ctx, composition)
}
```

**Strategy 2: Argo Workflows**
```go
type ArgoWorkflowProvisioner struct {
    client dynamic.Interface
    config ArgoConfig
}

func (p *ArgoWorkflowProvisioner) ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    // Create Argo Workflow
    workflow := &unstructured.Unstructured{
        Object: map[string]interface{}{
            "apiVersion": "argoproj.io/v1alpha1",
            "kind":       "Workflow",
            "metadata": map[string]interface{}{
                "name": fmt.Sprintf("provision-tenant-%s", req.TenantID),
            },
            "spec": map[string]interface{}{
                "entrypoint": "provision",
                "arguments": map[string]interface{}{
                    "parameters": []map[string]interface{}{
                        {"name": "tenantId", "value": req.TenantID},
                        {"name": "tier", "value": req.Tier},
                    },
                },
                "templates": []map[string]interface{}{
                    // Workflow steps
                },
            },
        },
    }
    
    return p.client.Create(ctx, workflow, metav1.CreateOptions{})
}
```


**Strategy 3: Hybrid (Crossplane + Argo Workflows)**
```go
type HybridProvisioner struct {
    crossplane *CrossplaneProvisioner
    argo       *ArgoWorkflowProvisioner
}

func (p *HybridProvisioner) ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    // Step 1: Create Crossplane Composition for infrastructure
    compResult, err := p.crossplane.ProvisionTenant(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Step 2: Create Argo Workflow for orchestration
    workflowReq := req
    workflowReq.InfrastructureID = compResult.CompositionID
    
    return p.argo.ProvisionTenant(ctx, workflowReq)
}
```

**Provisioning Request Structure:**
```go
type ProvisionRequest struct {
    TenantID    string                 `json:"tenantId"`
    Tier        string                 `json:"tier"`
    Name        string                 `json:"name"`
    Email       string                 `json:"email"`
    Config      map[string]interface{} `json:"config"`
    Resources   []ResourceSpec         `json:"resources"`
}

type ResourceSpec struct {
    Type       string                 `json:"type"`  // namespace, database, s3bucket, etc.
    Name       string                 `json:"name"`
    Parameters map[string]interface{} `json:"parameters"`
}
```


### 6. GitOps-First Approach

**Principle:** All infrastructure changes must go through Git, leveraging ArgoCD for continuous deployment.

**GitOps Workflow:**
```
1. Control Plane receives tenant creation request
2. Control Plane commits tenant configuration to Git repository
3. ArgoCD detects Git change
4. ArgoCD applies Crossplane Composition
5. Crossplane provisions infrastructure
6. ArgoCD reports sync status
7. Application Plane publishes success event
8. Control Plane updates tenant status
```

**Repository Structure:**
```
gitops-repo/
├── tenants/
│   ├── tenant-123/
│   │   ├── composition.yaml      # Crossplane Composition
│   │   ├── application.yaml      # ArgoCD Application
│   │   └── config.yaml           # Tenant configuration
│   └── tenant-456/
│       ├── composition.yaml
│       ├── application.yaml
│       └── config.yaml
├── platform/
│   ├── control-plane/
│   │   └── deployment.yaml
│   └── application-plane/
│       └── deployment.yaml
└── shared/
    ├── namespaces.yaml
    └── rbac.yaml
```

**ArgoCD Application Template:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: tenant-{{.TenantID}}
  namespace: argocd
spec:
  project: tenants
  source:
    repoURL: https://github.com/org/gitops-repo
    targetRevision: main
    path: tenants/tenant-{{.TenantID}}
  destination:
    server: https://kubernetes.default.svc
    namespace: tenant-{{.TenantID}}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```


### 7. MCP-First Integration

**Principle:** All Control Plane capabilities must be exposed via MCP tools for agent consumption.

**MCP Tool Structure:**
```go
// Control Plane MCP Tools
type ControlPlaneMCPServer struct {
    controlPlane *ControlPlane
}

func (s *ControlPlaneMCPServer) GetTools() []mcp.Tool {
    return []mcp.Tool{
        {
            Name:        "create_tenant",
            Description: "Create a new tenant in the platform",
            InputSchema: createTenantSchema,
            Handler:     s.handleCreateTenant,
        },
        {
            Name:        "get_tenant",
            Description: "Retrieve tenant details",
            InputSchema: getTenantSchema,
            Handler:     s.handleGetTenant,
        },
        {
            Name:        "update_tenant",
            Description: "Update tenant configuration",
            InputSchema: updateTenantSchema,
            Handler:     s.handleUpdateTenant,
        },
        {
            Name:        "delete_tenant",
            Description: "Remove a tenant",
            InputSchema: deleteTenantSchema,
            Handler:     s.handleDeleteTenant,
        },
        {
            Name:        "list_tenants",
            Description: "List all tenants",
            InputSchema: listTenantsSchema,
            Handler:     s.handleListTenants,
        },
    }
}
```

**MCP Tool Implementation Example:**
```go
func (s *ControlPlaneMCPServer) handleCreateTenant(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Extract tenant context from JWT (already in ctx)
    tenantID := ctx.Value("tenant_id").(string)
    
    // Parse parameters
    name := params["name"].(string)
    tier := params["tier"].(string)
    email := params["email"].(string)
    
    // Create tenant via Control Plane
    tenant, err := s.controlPlane.CreateTenant(ctx, CreateTenantRequest{
        Name:  name,
        Tier:  tier,
        Email: email,
    })
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "tenant_id":  tenant.ID,
        "name":       tenant.Name,
        "tier":       tenant.Tier,
        "status":     tenant.Status,
        "created_at": tenant.CreatedAt,
    }, nil
}
```


### 8. Multi-Tenant Security

**Principle:** Implement defense-in-depth security with tenant isolation at every layer.

**Security Layers:**

**1. API Layer (Gin + JWT)**
```go
func TenantContextMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract JWT claims
        claims := jwt.ExtractClaims(c)
        tenantID := claims["tenant_id"].(string)
        tenantTier := claims["tenant_tier"].(string)
        
        // Add to context
        ctx := context.WithValue(c.Request.Context(), "tenant_id", tenantID)
        ctx = context.WithValue(ctx, "tenant_tier", tenantTier)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}
```

**2. Authorization Layer (Ory Keto)**
```go
func CheckTenantPermission(ctx context.Context, keto *keto.Client, tenantID, permission string) error {
    userID := ctx.Value("user_id").(string)
    
    allowed, err := keto.Check(ctx, &ketoapi.RelationQuery{
        Namespace: "tenants",
        Object:    tenantID,
        Relation:  permission,
        Subject:   &ketoapi.Subject{ID: userID},
    })
    
    if err != nil || !allowed {
        return ErrUnauthorized
    }
    
    return nil
}
```

**3. Database Layer (PostgreSQL RLS)**
```sql
-- Enable RLS on all tenant tables
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_configs ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their tenant's data
CREATE POLICY tenant_isolation ON tenants
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY tenant_user_isolation ON tenant_users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Set tenant context before queries
SET app.tenant_id = 'tenant-uuid-here';
```


**4. Kubernetes Layer (Namespace Isolation)**
```yaml
# Namespace per tenant (for premium/enterprise tiers)
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-{{.TenantID}}
  labels:
    tenant-id: {{.TenantID}}
    tenant-tier: {{.Tier}}

---
# Network Policy: Deny cross-tenant traffic
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-cross-tenant
  namespace: tenant-{{.TenantID}}
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          tenant-id: {{.TenantID}}
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          tenant-id: {{.TenantID}}
```

**5. Agent Execution Layer (gVisor)**
```go
// Agent execution with gVisor sandboxing
type AgentExecutor struct {
    runtime string // "runsc" for gVisor
}

func (e *AgentExecutor) ExecuteAgent(ctx context.Context, req AgentExecutionRequest) error {
    // Create pod with gVisor runtime
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("agent-%s", req.AgentID),
            Namespace: fmt.Sprintf("tenant-%s", req.TenantID),
            Annotations: map[string]string{
                "io.kubernetes.cri.untrusted-workload": "true",
            },
        },
        Spec: corev1.PodSpec{
            RuntimeClassName: &e.runtime, // "gvisor"
            Containers: []corev1.Container{
                {
                    Name:  "agent",
                    Image: req.AgentImage,
                    // Resource limits, security context, etc.
                },
            },
        },
    }
    
    return e.k8sClient.Create(ctx, pod)
}
```


### 9. Observability and Monitoring

**Principle:** Provide tenant-aware observability with global and per-tenant views.

**Metrics Collection:**
```go
// Tenant-aware metrics
var (
    tenantRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "zerosbt_tenant_request_duration_seconds",
            Help: "Duration of tenant requests",
        },
        []string{"tenant_id", "tier", "method", "path", "status"},
    )
    
    tenantResourceUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "zerosbt_tenant_resource_usage",
            Help: "Tenant resource usage",
        },
        []string{"tenant_id", "tier", "resource_type"},
    )
)

// Record metrics
func RecordTenantRequest(tenantID, tier, method, path string, status int, duration time.Duration) {
    tenantRequestDuration.WithLabelValues(
        tenantID,
        tier,
        method,
        path,
        strconv.Itoa(status),
    ).Observe(duration.Seconds())
}
```

**Logging Pattern:**
```go
// Tenant-aware logging
log.WithFields(log.Fields{
    "tenant_id":   tenantID,
    "tenant_tier": tier,
    "user_id":     userID,
    "action":      "create_tenant",
    "resource_id": resourceID,
}).Info("Tenant created successfully")
```

**Monitoring Dashboards:**
- Global dashboard: Overall platform health
- Tenant dashboard: Per-tenant metrics and health
- Tier dashboard: Metrics aggregated by tier
- Cost dashboard: Per-tenant cost attribution


### 10. Testing Strategy

**Principle:** Use E2E tests with Testcontainers-Go to validate multi-tenant scenarios.

**Test Structure:**
```go
func TestTenantIsolation(t *testing.T) {
    // Start PostgreSQL with Testcontainers
    ctx := context.Background()
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
    )
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)
    
    // Start NATS with Testcontainers
    natsContainer, err := nats.RunContainer(ctx)
    require.NoError(t, err)
    defer natsContainer.Terminate(ctx)
    
    // Setup: Create Control Plane
    controlPlane := setupControlPlane(t, pgContainer, natsContainer)
    
    // Test: Create two tenants
    tenant1, err := controlPlane.CreateTenant(ctx, CreateTenantRequest{
        Name:  "tenant-1",
        Tier:  "basic",
        Email: "admin@tenant1.com",
    })
    require.NoError(t, err)
    
    tenant2, err := controlPlane.CreateTenant(ctx, CreateTenantRequest{
        Name:  "tenant-2",
        Tier:  "basic",
        Email: "admin@tenant2.com",
    })
    require.NoError(t, err)
    
    // Test: Tenant 1 creates a resource
    resource1, err := createResource(ctx, tenant1.ID, "resource-1")
    require.NoError(t, err)
    
    // Test: Tenant 2 cannot access tenant 1's resource
    _, err = getResource(ctx, tenant2.ID, resource1.ID)
    assert.Error(t, err, "Tenant 2 should not access tenant 1's resource")
    
    // Test: Tenant 1 can access their own resource
    resource, err := getResource(ctx, tenant1.ID, resource1.ID)
    assert.NoError(t, err)
    assert.Equal(t, resource1.ID, resource.ID)
}
```


## Package Structure

```
zero-sbt/
├── pkg/
│   ├── interfaces/           # Core interfaces (IAuth, IEventBus, etc.)
│   │   ├── auth.go
│   │   ├── eventbus.go
│   │   ├── provisioner.go
│   │   ├── storage.go
│   │   ├── billing.go
│   │   └── metering.go
│   │
│   ├── providers/            # Default implementations
│   │   ├── ory/             # Ory Stack auth implementation
│   │   ├── nats/            # NATS event bus implementation
│   │   ├── crossplane/      # Crossplane provisioner
│   │   ├── argoworkflows/   # Argo Workflows provisioner
│   │   └── postgres/        # PostgreSQL storage implementation
│   │
│   ├── controlplane/         # Control Plane components
│   │   ├── controlplane.go  # Main Control Plane struct
│   │   ├── api.go           # API server (Gin)
│   │   ├── tenant.go        # Tenant management service
│   │   ├── registration.go  # Tenant registration service
│   │   ├── user.go          # User management service
│   │   └── config.go        # Tenant configuration service
│   │
│   ├── applicationplane/     # Application Plane components
│   │   ├── applicationplane.go
│   │   ├── provisioner.go
│   │   └── workflows.go
│   │
│   ├── events/               # Event definitions and handlers
│   │   ├── definitions.go
│   │   ├── handlers.go
│   │   └── publisher.go
│   │
│   ├── mcp/                  # MCP server implementation
│   │   ├── server.go
│   │   ├── tools.go
│   │   └── handlers.go
│   │
│   └── models/               # Data models
│       ├── tenant.go
│       ├── user.go
│       ├── registration.go
│       └── events.go
│
├── examples/                 # Example implementations
│   ├── basic/               # Basic Control Plane + App Plane
│   ├── with-billing/        # With billing integration
│   └── with-metering/       # With metering integration
│
├── docs/                     # Documentation
│   ├── zero-sbt-design-principles.md
│   ├── getting-started.md
│   └── api-reference.md
│
└── tests/                    # E2E tests
    ├── integration/
    └── e2e/
```


## Usage Example

### Basic Control Plane Setup

```go
package main

import (
    "context"
    "log"
    
    zerosbt "github.com/zero-ops/zero-sbt/pkg"
    "github.com/zero-ops/zero-sbt/pkg/providers/ory"
    "github.com/zero-ops/zero-sbt/pkg/providers/nats"
    "github.com/zero-ops/zero-sbt/pkg/providers/postgres"
)

func main() {
    ctx := context.Background()
    
    // Initialize providers
    auth := ory.NewOryAuth(ory.Config{
        KratosURL: "http://kratos:4433",
        HydraURL:  "http://hydra:4444",
        KetoURL:   "http://keto:4466",
    })
    
    eventBus := nats.NewNATSEventBus(nats.Config{
        URL: "nats://nats:4222",
    })
    
    storage := postgres.NewPostgresStorage(postgres.Config{
        Host:     "postgres",
        Port:     5432,
        Database: "zerosbt",
        User:     "zerosbt",
        Password: "password",
    })
    
    // Create Control Plane
    controlPlane, err := zerosbt.NewControlPlane(ctx, zerosbt.ControlPlaneConfig{
        Auth:              auth,
        EventBus:          eventBus,
        Storage:           storage,
        SystemAdminEmail:  "admin@example.com",
        SystemAdminName:   "admin",
        APIPort:           8080,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Start Control Plane
    if err := controlPlane.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```


### Basic Application Plane Setup

```go
package main

import (
    "context"
    "log"
    
    zerosbt "github.com/zero-ops/zero-sbt/pkg"
    "github.com/zero-ops/zero-sbt/pkg/providers/nats"
    "github.com/zero-ops/zero-sbt/pkg/providers/crossplane"
    "github.com/zero-ops/zero-sbt/pkg/providers/argoworkflows"
)

func main() {
    ctx := context.Background()
    
    // Initialize providers
    eventBus := nats.NewNATSEventBus(nats.Config{
        URL: "nats://nats:4222",
    })
    
    provisioner := argoworkflows.NewArgoWorkflowProvisioner(argoworkflows.Config{
        KubeConfig: "/path/to/kubeconfig",
        Namespace:  "argo",
    })
    
    // Create Application Plane
    appPlane, err := zerosbt.NewApplicationPlane(ctx, zerosbt.ApplicationPlaneConfig{
        EventBus:    eventBus,
        Provisioner: provisioner,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Register event handlers
    appPlane.OnOnboardingRequest(func(ctx context.Context, event zerosbt.OnboardingRequestEvent) error {
        log.Printf("Provisioning tenant: %s", event.TenantID)
        
        // Provision tenant resources
        result, err := provisioner.ProvisionTenant(ctx, zerosbt.ProvisionRequest{
            TenantID: event.TenantID,
            Tier:     event.Tier,
            Name:     event.Name,
            Email:    event.Email,
        })
        if err != nil {
            // Publish failure event
            return appPlane.PublishProvisionFailure(ctx, event.TenantID, err)
        }
        
        // Publish success event
        return appPlane.PublishProvisionSuccess(ctx, event.TenantID, result)
    })
    
    // Start Application Plane
    if err := appPlane.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```


### Custom Provider Implementation

```go
// Example: Custom authentication provider
package custom

import (
    "context"
    zerosbt "github.com/zero-ops/zero-sbt/pkg/interfaces"
)

type CustomAuth struct {
    config CustomAuthConfig
    client *CustomAuthClient
}

func NewCustomAuth(config CustomAuthConfig) zerosbt.IAuth {
    return &CustomAuth{
        config: config,
        client: NewCustomAuthClient(config),
    }
}

func (a *CustomAuth) CreateUser(ctx context.Context, user zerosbt.User) error {
    // Custom implementation
    return a.client.CreateUser(ctx, user)
}

func (a *CustomAuth) AuthenticateUser(ctx context.Context, credentials zerosbt.Credentials) (*zerosbt.Token, error) {
    // Custom implementation
    return a.client.Authenticate(ctx, credentials)
}

// Implement all other IAuth methods...

// Use custom provider
func main() {
    customAuth := custom.NewCustomAuth(custom.CustomAuthConfig{
        // Custom configuration
    })
    
    controlPlane, err := zerosbt.NewControlPlane(ctx, zerosbt.ControlPlaneConfig{
        Auth: customAuth,  // Works seamlessly
        // ...
    })
}
```


## Implementation Roadmap

### Phase 1: Core Interfaces and Models (Week 1-2)
- [ ] Define all core interfaces (IAuth, IEventBus, IProvisioner, IStorage)
- [ ] Define data models (Tenant, User, Registration, Event)
- [ ] Create package structure
- [ ] Write interface documentation

### Phase 2: Default Providers (Week 3-4)
- [ ] Implement Ory Stack auth provider
- [ ] Implement NATS event bus provider
- [ ] Implement PostgreSQL storage provider (with sqlc)
- [ ] Implement Crossplane provisioner
- [ ] Implement Argo Workflows provisioner

### Phase 3: Control Plane (Week 5-6)
- [ ] Implement Control Plane core
- [ ] Implement Tenant Management Service
- [ ] Implement Tenant Registration Service
- [ ] Implement User Management Service
- [ ] Implement Tenant Configuration Service
- [ ] Implement API server (Gin)
- [ ] Add JWT middleware
- [ ] Add tenant context middleware

### Phase 4: Application Plane (Week 7-8)
- [ ] Implement Application Plane core
- [ ] Implement event handlers
- [ ] Implement provisioning workflows
- [ ] Integrate with Crossplane
- [ ] Integrate with Argo Workflows
- [ ] Integrate with ArgoCD

### Phase 5: MCP Integration (Week 9)
- [ ] Implement MCP server
- [ ] Create MCP tools for all Control Plane operations
- [ ] Add MCP tool documentation
- [ ] Test MCP integration

### Phase 6: Testing and Documentation (Week 10-11)
- [ ] Write E2E tests with Testcontainers-Go
- [ ] Write integration tests
- [ ] Create example implementations
- [ ] Write getting started guide
- [ ] Write API reference documentation

### Phase 7: Advanced Features (Week 12+)
- [ ] Implement billing integration (optional)
- [ ] Implement metering integration (optional)
- [ ] Add observability helpers
- [ ] Add CLI tool
- [ ] Create Helm charts for deployment


## Key Design Decisions

### 1. Why Go Instead of TypeScript/CDK?
- **Reason**: Zero-ops platform is Go-based, better performance, simpler deployment
- **Trade-off**: Lose CDK's CloudFormation synthesis, but gain Kubernetes-native tooling

### 2. Why NATS Instead of EventBridge?
- **Reason**: Cloud-agnostic, better performance, simpler operations, BYOC model
- **Trade-off**: Need to manage NATS infrastructure, but gain flexibility

### 3. Why Ory Stack Instead of Cognito?
- **Reason**: Open-source, self-hosted, cloud-agnostic, better customization
- **Trade-off**: More operational overhead, but gain control and cost savings

### 4. Why Crossplane + Argo Workflows Instead of CloudFormation?
- **Reason**: Kubernetes-native, GitOps-first, cloud-agnostic, better for BYOC
- **Trade-off**: Steeper learning curve, but gain flexibility and portability

### 5. Why PostgreSQL Instead of DynamoDB?
- **Reason**: Relational model fits tenant data, better query capabilities, RLS support
- **Trade-off**: Need to manage PostgreSQL, but gain SQL power and RLS

### 6. Why Interface-Based Abstraction?
- **Reason**: Allow provider swapping without breaking user code, future-proof
- **Trade-off**: More upfront design work, but gain long-term flexibility

### 7. Why GitOps-First?
- **Reason**: Audit trail, rollback capability, declarative, aligns with platform principles
- **Trade-off**: Slower provisioning, but gain safety and traceability

### 8. Why MCP-First?
- **Reason**: Agent-friendly interface, standardized protocol, discoverable capabilities
- **Trade-off**: Additional abstraction layer, but gain agent integration


## Comparison: SBT-AWS vs Zero-SBT

| Aspect | SBT-AWS | Zero-SBT |
|--------|---------|----------|
| **Language** | TypeScript | Go |
| **Infrastructure** | AWS CDK | Kubernetes-native |
| **Event Bus** | AWS EventBridge | NATS |
| **Authentication** | AWS Cognito | Ory Stack (Kratos/Hydra/Keto) |
| **Provisioning** | CloudFormation + CodeBuild | Crossplane + Argo Workflows |
| **Database** | DynamoDB | PostgreSQL + sqlc |
| **API** | API Gateway + Lambda | Gin (Go HTTP framework) |
| **Deployment** | CloudFormation | GitOps (ArgoCD) |
| **Isolation** | IAM + VPC | RLS + Ory Keto + K8s Namespaces |
| **Agent Integration** | N/A | MCP Protocol |
| **Cloud Model** | AWS-only | Cloud-agnostic (BYOC) |
| **Package Manager** | npm | Go modules |
| **Testing** | Jest | Testcontainers-Go |

## Similarities with SBT-AWS

1. **Control Plane + Application Plane Architecture**: Same conceptual separation
2. **Interface-Based Abstraction**: Both use interfaces for pluggable components
3. **Event-Driven Communication**: Both use events for plane-to-plane communication
4. **Tenant Lifecycle Management**: Same workflows (onboarding, offboarding, etc.)
5. **Pluggable Providers**: Both allow swapping implementations
6. **Standard Events**: Similar event naming and structure
7. **Multi-Tenant Security**: Both emphasize tenant isolation
8. **Observability**: Both provide tenant-aware monitoring


## Best Practices

### 1. Always Use Interfaces
```go
// ❌ DON'T: Depend on concrete types
func NewControlPlane(oryAuth *ory.OryAuth) *ControlPlane {
    // Tightly coupled to Ory
}

// ✅ DO: Depend on interfaces
func NewControlPlane(auth zerosbt.IAuth) *ControlPlane {
    // Works with any IAuth implementation
}
```

### 2. Always Include Tenant Context
```go
// ❌ DON'T: Query without tenant context
func GetUser(ctx context.Context, userID string) (*User, error) {
    return db.Query("SELECT * FROM users WHERE id = $1", userID)
}

// ✅ DO: Always filter by tenant
func GetUser(ctx context.Context, tenantID, userID string) (*User, error) {
    return db.Query("SELECT * FROM users WHERE tenant_id = $1 AND id = $2", tenantID, userID)
}
```

### 3. Always Use GitOps for Infrastructure
```go
// ❌ DON'T: Apply Kubernetes resources directly
func ProvisionTenant(ctx context.Context, req ProvisionRequest) error {
    return k8sClient.Create(ctx, namespace)
}

// ✅ DO: Commit to Git, let ArgoCD apply
func ProvisionTenant(ctx context.Context, req ProvisionRequest) error {
    return gitClient.CommitTenantConfig(ctx, req.TenantID, config)
}
```

### 4. Always Emit Events for Async Operations
```go
// ❌ DON'T: Block on long-running operations
func OnboardTenant(ctx context.Context, req OnboardRequest) (*Tenant, error) {
    tenant := createTenant(req)
    provisionResources(tenant)  // Blocks for minutes
    return tenant, nil
}

// ✅ DO: Emit event and return immediately
func OnboardTenant(ctx context.Context, req OnboardRequest) (*Tenant, error) {
    tenant := createTenant(req)
    eventBus.Publish(ctx, OnboardingRequestEvent{TenantID: tenant.ID})
    return tenant, nil
}
```


### 5. Always Log with Tenant Context
```go
// ❌ DON'T: Log without tenant information
log.Info("User created")

// ✅ DO: Include tenant context in all logs
log.WithFields(log.Fields{
    "tenant_id": tenantID,
    "user_id":   userID,
    "action":    "create_user",
}).Info("User created successfully")
```

### 6. Always Tag Metrics with Tenant ID
```go
// ❌ DON'T: Record metrics without tenant labels
requestDuration.Observe(duration.Seconds())

// ✅ DO: Tag with tenant_id
requestDuration.WithLabelValues(tenantID, tier, method, path).Observe(duration.Seconds())
```

### 7. Always Test Multi-Tenant Scenarios
```go
// ❌ DON'T: Test single-tenant only
func TestCreateUser(t *testing.T) {
    user, err := CreateUser(ctx, "user-1")
    assert.NoError(t, err)
}

// ✅ DO: Test tenant isolation
func TestTenantIsolation(t *testing.T) {
    tenant1 := createTenant(t, "tenant-1")
    tenant2 := createTenant(t, "tenant-2")
    
    user1 := createUser(t, tenant1.ID, "user-1")
    
    // Tenant 2 should not access tenant 1's user
    _, err := getUser(t, tenant2.ID, user1.ID)
    assert.Error(t, err)
}
```

### 8. Always Use Type-Safe Database Queries
```go
// ❌ DON'T: Use raw SQL strings
func GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
    var tenant Tenant
    err := db.QueryRow("SELECT * FROM tenants WHERE id = $1", tenantID).Scan(&tenant)
    return &tenant, err
}

// ✅ DO: Use sqlc generated queries
func GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
    return queries.GetTenant(ctx, tenantID)
}
```


## Conclusion

Zero-SBT provides a Go-based SaaS builder toolkit that adapts the proven architecture patterns from AWS SBT-AWS to the zero-ops platform's technology stack. By following these design principles, developers can build multi-tenant SaaS applications with:

- **Clear separation of concerns** (Control Plane vs Application Plane)
- **Pluggable components** (swap implementations without breaking code)
- **Event-driven architecture** (async communication via NATS)
- **Strong tenant isolation** (defense-in-depth security)
- **GitOps-first operations** (declarative, auditable, rollback-capable)
- **MCP-first integration** (agent-friendly interface)
- **Cloud-agnostic design** (BYOC model, portable across clouds)

The toolkit abstracts away the complexity of multi-tenant SaaS infrastructure while maintaining flexibility for customization and future evolution.

## References

- [AWS SaaS Builder Toolkit (SBT-AWS)](https://github.com/awslabs/sbt-aws)
- [AWS Well-Architected SaaS Lens](https://docs.aws.amazon.com/wellarchitected/latest/saas-lens/saas-lens.html)
- [Crossplane Documentation](https://docs.crossplane.io/)
- [Argo Workflows Documentation](https://argoproj.github.io/argo-workflows/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [NATS Documentation](https://docs.nats.io/)
- [Ory Documentation](https://www.ory.sh/docs/)

---

**Document Version:** 1.0  
**Last Updated:** 2026-03-16  
**Status:** Draft for Review

## Additional Design Principles

### 11. Dual Admin Console Architecture

**Principle:** Provide separate admin consoles for SaaS Provider (platform admin) and Tenant Admins with distinct capabilities.

**SaaS Provider Admin Console:**
- **Purpose**: Platform-wide management and operations
- **Technology**: TypeScript/React (Platform Console)
- **Capabilities**:
  - Tenant management (create, update, delete, suspend)
  - Tier management (define tiers, pricing, quotas)
  - Platform monitoring (all tenants, global metrics)
  - System configuration
  - Billing and metering oversight
  - User management (platform admins)
  - Audit logs (all tenant activities)
  - Resource allocation and quotas
  - Feature flag management

**Tenant Admin Console:**
- **Purpose**: Tenant-specific management
- **Technology**: TypeScript/React (embedded or standalone)
- **Capabilities**:
  - Tenant user management (CRUD operations)
  - Tenant configuration
  - Tenant-specific monitoring and metrics
  - Billing and usage reports (own tenant)
  - Audit logs (own tenant only)
  - API key management
  - Webhook configuration
  - Integration settings


**Console Architecture:**
```
┌─────────────────────────────────────────────────────────────┐
│                    SaaS Provider Admin Console               │
│  (Platform Console - TypeScript/React)                       │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Tenants    │  │    Tiers     │  │  Monitoring  │      │
│  │  Management  │  │  Management  │  │   (Global)   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Billing    │  │  Audit Logs  │  │   System     │      │
│  │  (All Tenants)│  │  (Platform)  │  │   Config     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Control Plane API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Control Plane (Go)                      │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Tenant     │  │     Tier     │  │     User     │      │
│  │  Management  │  │  Management  │  │  Management  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Tenant API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Tenant Admin Console                      │
│  (Tenant-specific - TypeScript/React)                        │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │    Users     │  │    Config    │  │  Monitoring  │      │
│  │  (Own Tenant)│  │  (Own Tenant)│  │  (Own Tenant)│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Billing    │  │  Audit Logs  │  │   API Keys   │      │
│  │  (Own Tenant)│  │  (Own Tenant)│  │              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

