# open-sbt Design Document

## Overview

open-sbt is a Go-based SaaS builder toolkit that provides reusable abstractions for building multi-tenant SaaS applications. The toolkit follows a Control Plane + Application Plane architecture pattern with event-driven communication, enabling SaaS builders to create production-grade backend infrastructure without vendor lock-in.

The design emphasizes interface-based abstraction, allowing developers to swap implementations without breaking application code. This approach provides flexibility while maintaining consistency across different technology stacks and deployment environments.

### Key Design Principles

1. **Interface-First Architecture**: All core components are defined as Go interfaces, enabling pluggable implementations
2. **Event-Driven Communication**: Asynchronous messaging between Control Plane and Application Plane via event bus
3. **Multi-Tenant Security**: Defense-in-depth isolation with tenant context enforcement at every layer
4. **GitOps-First**: All infrastructure changes managed through Git commits and automated reconciliation
5. **Provider Agnostic**: No prescribed technology implementations, supporting multiple providers per interface
6. **Comprehensive Testing**: E2E testing patterns with Testcontainers-Go for reliability across implementations

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Control Plane                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │     Tenant      │  │      User       │  │   Configuration │ │
│  │   Management    │  │   Management    │  │   Management    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                │                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │      Auth       │  │     Storage     │  │     Billing     │ │
│  │   (IAuth)       │  │   (IStorage)    │  │   (IBilling)    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────┬───────────────────────────────┘
                                  │ Event Bus (IEventBus)
                                  │ Async Communication
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Application Plane                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Provisioner   │  │     Metering    │  │   Event Bus     │ │
│  │ (IProvisioner)  │  │  (IMetering)    │  │  (IEventBus)    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                │                                │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │     GitOps      │  │  Infrastructure │  │   Monitoring    │ │
│  │   Workflows     │  │   Management    │  │   & Logging     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Component Relationships

The architecture follows a strict separation between Control Plane and Application Plane:

- **Control Plane**: Handles business logic, tenant management, authentication, and user-facing APIs
- **Application Plane**: Manages infrastructure provisioning, resource allocation, and responds to Control Plane events
- **Event Bus**: Provides asynchronous communication channel between planes with guaranteed delivery
- **Interfaces**: Abstract core functionality allowing pluggable provider implementations

## Components and Interfaces

### Core Interface Definitions

#### IAuth Interface

The authentication and authorization provider interface handles user management and tenant-scoped authentication.

```go
// IAuth provides authentication and authorization capabilities
type IAuth interface {
    // User Management
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
    
    // Admin Operations
    CreateAdminUser(ctx context.Context, props CreateAdminUserProps) error
    
    // Token Configuration
    GetJWTIssuer() string
    GetJWTAudience() []string
    GetTokenEndpoint() string
    GetWellKnownEndpoint() string
}

// Supporting Types
type User struct {
    ID       string                 `json:"id"`
    Email    string                 `json:"email"`
    Name     string                 `json:"name"`
    TenantID string                 `json:"tenant_id"`
    Roles    []string               `json:"roles"`
    Metadata map[string]interface{} `json:"metadata"`
    Active   bool                   `json:"active"`
    CreatedAt time.Time             `json:"created_at"`
    UpdatedAt time.Time             `json:"updated_at"`
}

type Token struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    TokenType    string    `json:"token_type"`
    ExpiresIn    int       `json:"expires_in"`
    ExpiresAt    time.Time `json:"expires_at"`
}

type Claims struct {
    UserID     string   `json:"sub"`
    TenantID   string   `json:"tenant_id"`
    TenantTier string   `json:"tenant_tier"`
    Roles      []string `json:"roles"`
    Permissions []string `json:"permissions"`
    ExpiresAt  int64    `json:"exp"`
    IssuedAt   int64    `json:"iat"`
}
```

#### IEventBus Interface

The event bus interface provides asynchronous communication between Control Plane and Application Plane.

```go
// IEventBus provides message bus capabilities for inter-plane communication
type IEventBus interface {
    // Event Publishing
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event) error
    
    // Event Subscription
    Subscribe(ctx context.Context, eventType string, handler EventHandler) error
    SubscribeQueue(ctx context.Context, eventType string, queueGroup string, handler EventHandler) error
    
    // Event Definitions
    GetControlPlaneEventSource() string
    GetApplicationPlaneEventSource() string
    CreateControlPlaneEvent(detailType string) EventDefinition
    CreateApplicationPlaneEvent(detailType string) EventDefinition
    CreateCustomEvent(detailType string, source string) EventDefinition
    
    // Standard Events
    GetStandardEvents() map[string]EventDefinition
    
    // Permissions
    GrantPublishPermissions(grantee string) error
}

// Event Structure
type Event struct {
    ID         string                 `json:"id"`
    Version    string                 `json:"version"`
    DetailType string                 `json:"detailType"`
    Source     string                 `json:"source"`
    Time       time.Time              `json:"time"`
    Region     string                 `json:"region,omitempty"`
    Resources  []string               `json:"resources,omitempty"`
    Detail     map[string]interface{} `json:"detail"`
}

type EventHandler func(ctx context.Context, event Event) error

type EventDefinition struct {
    DetailType  string `json:"detailType"`
    Source      string `json:"source"`
    Description string `json:"description"`
    Schema      string `json:"schema"`
}
```

#### IProvisioner Interface

The provisioner interface handles tenant resource provisioning and infrastructure management.

```go
// IProvisioner handles tenant resource provisioning and management
type IProvisioner interface {
    // Tenant Resource Management
    ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
    DeprovisionTenant(ctx context.Context, req DeprovisionRequest) (*DeprovisionResult, error)
    UpdateTenantResources(ctx context.Context, req UpdateRequest) (*UpdateResult, error)
    
    // Status and Monitoring
    GetProvisioningStatus(ctx context.Context, tenantID string) (*ProvisioningStatus, error)
    ListTenantResources(ctx context.Context, tenantID string) ([]Resource, error)
    
    // Warm Pool Management (Gap 2 Fix)
    ClaimWarmSlot(ctx context.Context, tenantID string, tier string) (*WarmSlotResult, error)
    RefillWarmPool(ctx context.Context, tier string, targetCount int) error
    GetWarmPoolStatus(ctx context.Context, tier string) (*WarmPoolStatus, error)
    
    // GitOps Integration
    CommitTenantConfig(ctx context.Context, tenantID string, config TenantConfig) error
    RollbackTenantConfig(ctx context.Context, tenantID string, commitHash string) error
    GetGitRepository(ctx context.Context, tenantID string) (*GitRepository, error)
    
    // Sync Triggers (Gap 3 Fix)
    TriggerSync(ctx context.Context, tenantID string) error
    TriggerWebhookSync(ctx context.Context, tenantID string, webhookURL string) error
}

// Provisioning Types
type ProvisionRequest struct {
    TenantID    string                 `json:"tenant_id"`
    Tier        string                 `json:"tier"`
    Name        string                 `json:"name"`
    Email       string                 `json:"email"`
    Config      map[string]interface{} `json:"config"`
    Resources   []ResourceSpec         `json:"resources"`
}

type ProvisionResult struct {
    TenantID      string            `json:"tenant_id"`
    Status        string            `json:"status"`
    Resources     []Resource        `json:"resources"`
    GitCommitHash string            `json:"git_commit_hash"`
    Metadata      map[string]string `json:"metadata"`
    CreatedAt     time.Time         `json:"created_at"`
}

type TenantConfig struct {
    TenantID     string                 `json:"tenant_id"`
    Tier         string                 `json:"tier"`
    HelmValues   map[string]interface{} `json:"helm_values"`
    GitCommit    string                 `json:"git_commit,omitempty"`
    SyncStatus   string                 `json:"sync_status,omitempty"`
    LastSyncTime time.Time              `json:"last_sync_time,omitempty"`
}

type GitRepository struct {
    URL       string `json:"url"`
    Branch    string `json:"branch"`
    Path      string `json:"path"`
    CommitSHA string `json:"commit_sha"`
}

type ResourceSpec struct {
    Type       string                 `json:"type"`
    Name       string                 `json:"name"`
    Parameters map[string]interface{} `json:"parameters"`
}

type Resource struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`
    Name       string                 `json:"name"`
    Status     string                 `json:"status"`
    Properties map[string]interface{} `json:"properties"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}

// Warm Pool Types (Gap 2 Fix)
type WarmSlotResult struct {
    SlotID        string            `json:"slot_id"`
    TenantID      string            `json:"tenant_id"`
    Tier          string            `json:"tier"`
    Resources     []Resource        `json:"resources"`
    ClaimedAt     time.Time         `json:"claimed_at"`
    Metadata      map[string]string `json:"metadata"`
}

type WarmPoolStatus struct {
    Tier           string    `json:"tier"`
    AvailableSlots int       `json:"available_slots"`
    TotalSlots     int       `json:"total_slots"`
    TargetSlots    int       `json:"target_slots"`
    LastRefill     time.Time `json:"last_refill"`
}

// Provisioning Status Types (Gap 6 Fix - Reconciliation)
type ProvisioningStatus struct {
    TenantID      string            `json:"tenant_id"`
    Status        string            `json:"status"`        // "synced", "healthy", "degraded", "failed", "progressing", "syncing", "not_found"
    Resources     []Resource        `json:"resources"`
    GitCommitHash string            `json:"git_commit_hash"`
    ErrorMessage  string            `json:"error_message,omitempty"`
    LastSyncTime  time.Time         `json:"last_sync_time"`
    Metadata      map[string]string `json:"metadata"`
}
```

#### IStorage Interface

The storage interface provides tenant-aware data persistence with strong isolation guarantees.

```go
// IStorage provides tenant-aware data persistence capabilities
type IStorage interface {
    // Tenant Management
    CreateTenant(ctx context.Context, tenant Tenant) error
    GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
    UpdateTenant(ctx context.Context, tenantID string, updates TenantUpdates) error
    DeleteTenant(ctx context.Context, tenantID string) error
    ListTenants(ctx context.Context, filters TenantFilters) ([]Tenant, error)
    
    // Tenant Registration
    CreateTenantRegistration(ctx context.Context, reg TenantRegistration) error
    GetTenantRegistration(ctx context.Context, regID string) (*TenantRegistration, error)
    UpdateTenantRegistration(ctx context.Context, regID string, updates RegistrationUpdates) error
    DeleteTenantRegistration(ctx context.Context, regID string) error
    ListTenantRegistrations(ctx context.Context, filters RegistrationFilters) ([]TenantRegistration, error)
    
    // Tenant Configuration
    SetTenantConfig(ctx context.Context, tenantID string, config map[string]interface{}) error
    GetTenantConfig(ctx context.Context, tenantID string) (map[string]interface{}, error)
    DeleteTenantConfig(ctx context.Context, tenantID string) error
    
    // Event Idempotency (Gap 5 Fix - Inbox Pattern)
    RecordProcessedEvent(ctx context.Context, eventID string) error
    IsEventProcessed(ctx context.Context, eventID string) (bool, error)
    
    // Webhook-Driven State Management (Event-Driven State Machine)
    UpdateTenantStatus(ctx context.Context, tenantID string, status string) error
    UpdateTenantArgoStatus(ctx context.Context, tenantID string, syncStatus, healthStatus string) error
    TouchTenantObservation(ctx context.Context, tenantID string) error // Updates LastObservedAt
    
    // Orphaned Infrastructure Detection (Gap 6 Fix - Zero Polling)
    ListStuckTenants(ctx context.Context, stuckStates []string, olderThan time.Duration) ([]Tenant, error)
    ListUnobservedTenants(ctx context.Context, olderThan time.Duration) ([]Tenant, error) // No ArgoCD webhooks
    
    // Transaction Support
    BeginTransaction(ctx context.Context) (Transaction, error)
}

// Storage Types - Event-Driven State Machine Model
type Tenant struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Tier        string                 `json:"tier"`
    
    // Business Truth State Machine (PostgreSQL = Source of Truth)
    Status      string                 `json:"status"` // CREATING, GIT_COMMITTED, SYNCING, READY, FAILED
    
    // Infrastructure Observability (Pushed from ArgoCD Webhooks)
    ArgoSyncStatus   string            `json:"argo_sync_status,omitempty"`   // Synced, OutOfSync
    ArgoHealthStatus string            `json:"argo_health_status,omitempty"` // Healthy, Progressing, Degraded
    
    OwnerEmail       string                 `json:"owner_email"`
    Config           map[string]interface{} `json:"config"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
    LastObservedAt   time.Time              `json:"last_observed_at"` // Updated every ArgoCD webhook
}

type TenantRegistration struct {
    ID         string                 `json:"id"`
    TenantID   string                 `json:"tenant_id"`
    Status     string                 `json:"status"`
    Name       string                 `json:"name"`
    Email      string                 `json:"email"`
    Tier       string                 `json:"tier"`
    Config     map[string]interface{} `json:"config"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}

type Transaction interface {
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

#### IBilling Interface

The billing interface provides integration with external billing systems and customer management.

```go
// IBilling provides billing system integration capabilities
type IBilling interface {
    // Customer Management
    CreateCustomer(ctx context.Context, customer BillingCustomer) error
    GetCustomer(ctx context.Context, customerID string) (*BillingCustomer, error)
    UpdateCustomer(ctx context.Context, customerID string, updates CustomerUpdates) error
    DeleteCustomer(ctx context.Context, customerID string) error
    
    // Subscription Management
    CreateSubscription(ctx context.Context, subscription Subscription) error
    GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
    UpdateSubscription(ctx context.Context, subscriptionID string, updates SubscriptionUpdates) error
    CancelSubscription(ctx context.Context, subscriptionID string) error
    
    // Usage and Billing
    RecordUsage(ctx context.Context, usage UsageRecord) error
    GetUsage(ctx context.Context, customerID string, period TimePeriod) (*UsageReport, error)
    GenerateInvoice(ctx context.Context, customerID string, period TimePeriod) (*Invoice, error)
    
    // Webhook Handling
    HandleWebhook(ctx context.Context, payload []byte) error
}

// Billing Types
type BillingCustomer struct {
    ID         string                 `json:"id"`
    TenantID   string                 `json:"tenant_id"`
    Email      string                 `json:"email"`
    Name       string                 `json:"name"`
    Metadata   map[string]interface{} `json:"metadata"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}

type Subscription struct {
    ID         string                 `json:"id"`
    CustomerID string                 `json:"customer_id"`
    PlanID     string                 `json:"plan_id"`
    Status     string                 `json:"status"`
    Metadata   map[string]interface{} `json:"metadata"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}

type UsageRecord struct {
    ID         string                 `json:"id"`
    CustomerID string                 `json:"customer_id"`
    MeterName  string                 `json:"meter_name"`
    Value      float64                `json:"value"`
    Timestamp  time.Time              `json:"timestamp"`
    Metadata   map[string]interface{} `json:"metadata"`
}
```

#### IMetering Interface

The metering interface provides usage tracking and aggregation capabilities for billing and monitoring.

```go
// IMetering provides usage metering and tracking capabilities
type IMetering interface {
    // Meter Management
    CreateMeter(ctx context.Context, meter Meter) error
    GetMeter(ctx context.Context, meterID string) (*Meter, error)
    UpdateMeter(ctx context.Context, meterID string, updates MeterUpdates) error
    DeleteMeter(ctx context.Context, meterID string) error
    ListMeters(ctx context.Context, filters MeterFilters) ([]Meter, error)
    
    // Usage Ingestion
    IngestUsageEvent(ctx context.Context, event UsageEvent) error
    IngestUsageEventBatch(ctx context.Context, events []UsageEvent) error
    
    // Usage Queries
    GetUsage(ctx context.Context, meterID string, period TimePeriod) (*UsageData, error)
    GetTenantUsage(ctx context.Context, tenantID string, period TimePeriod) (*TenantUsageData, error)
    AggregateUsage(ctx context.Context, req AggregationRequest) (*AggregationResult, error)
    
    // Usage Management
    CancelUsageEvents(ctx context.Context, eventIDs []string) error
}

// Metering Types
type Meter struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Unit        string                 `json:"unit"`
    Type        string                 `json:"type"`
    Config      map[string]interface{} `json:"config"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type UsageEvent struct {
    ID         string                 `json:"id"`
    TenantID   string                 `json:"tenant_id"`
    MeterID    string                 `json:"meter_id"`
    Value      float64                `json:"value"`
    Timestamp  time.Time              `json:"timestamp"`
    Properties map[string]interface{} `json:"properties"`
}

type UsageData struct {
    MeterID     string                 `json:"meter_id"`
    TenantID    string                 `json:"tenant_id"`
    Period      TimePeriod             `json:"period"`
    TotalUsage  float64                `json:"total_usage"`
    EventCount  int64                  `json:"event_count"`
    Breakdown   map[string]interface{} `json:"breakdown"`
}
```

#### ITierManager Interface

The tier manager interface provides tier configuration and quota management capabilities (Gap 12 Fix).

```go
// ITierManager provides tier configuration and quota management
type ITierManager interface {
    // Tier Management
    CreateTier(ctx context.Context, tier TierConfig) error
    GetTier(ctx context.Context, tierName string) (*TierConfig, error)
    UpdateTier(ctx context.Context, tierName string, updates TierUpdates) error
    DeleteTier(ctx context.Context, tierName string) error
    ListTiers(ctx context.Context) ([]TierConfig, error)
    
    // Quota Management
    ValidateTierQuota(ctx context.Context, tierName string, usage ResourceUsage) error
    GetTierQuotas(ctx context.Context, tierName string) (*TierQuotas, error)
    UpdateTierQuotas(ctx context.Context, tierName string, quotas TierQuotas) error
    
    // Tier Features
    GetTierFeatures(ctx context.Context, tierName string) ([]string, error)
    IsTierFeatureEnabled(ctx context.Context, tierName string, feature string) (bool, error)
}

// Tier Management Types
type TierConfig struct {
    Name         string                 `json:"name"`
    DisplayName  string                 `json:"display_name"`
    Description  string                 `json:"description"`
    Quotas       TierQuotas             `json:"quotas"`
    Features     []string               `json:"features"`
    Pricing      map[string]interface{} `json:"pricing"`
    Metadata     map[string]interface{} `json:"metadata"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}

type TierQuotas struct {
    Users       int     `json:"users"`        // Max users per tenant
    StorageGB   int     `json:"storage_gb"`   // Max storage in GB
    APIRequests int     `json:"api_requests"` // Max API requests per month
    CPU         string  `json:"cpu"`          // Max CPU allocation
    Memory      string  `json:"memory"`       // Max memory allocation
    Custom      map[string]interface{} `json:"custom"` // Custom quotas
}

type ResourceUsage struct {
    Users       int                    `json:"users"`
    StorageGB   float64                `json:"storage_gb"`
    APIRequests int                    `json:"api_requests"`
    CPU         string                 `json:"cpu"`
    Memory      string                 `json:"memory"`
    Custom      map[string]interface{} `json:"custom"`
}

type TierUpdates struct {
    DisplayName *string                 `json:"display_name,omitempty"`
    Description *string                 `json:"description,omitempty"`
    Quotas      *TierQuotas             `json:"quotas,omitempty"`
    Features    *[]string               `json:"features,omitempty"`
    Pricing     *map[string]interface{} `json:"pricing,omitempty"`
    Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}
```

#### ISecretManager Interface

The secret manager interface provides secure secret management for GitOps workflows (Gap 8 Fix).

```go
// ISecretManager provides secure secret management for GitOps workflows
type ISecretManager interface {
    // Secret Management
    CreateSecret(ctx context.Context, secret SecretData) error
    GetSecret(ctx context.Context, secretID string) (*SecretData, error)
    UpdateSecret(ctx context.Context, secretID string, updates SecretUpdates) error
    DeleteSecret(ctx context.Context, secretID string) error
    ListSecrets(ctx context.Context, filters SecretFilters) ([]SecretMetadata, error)
    
    // Tenant-Scoped Secrets
    CreateTenantSecret(ctx context.Context, tenantID string, secret TenantSecretData) error
    GetTenantSecret(ctx context.Context, tenantID string, secretName string) (*TenantSecretData, error)
    ListTenantSecrets(ctx context.Context, tenantID string) ([]TenantSecretMetadata, error)
    
    // GitOps Integration
    EncryptForGit(ctx context.Context, data []byte) ([]byte, error)
    DecryptFromGit(ctx context.Context, encryptedData []byte) ([]byte, error)
    RotateEncryptionKey(ctx context.Context) error
}

// Secret Management Types
type SecretData struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Type        string                 `json:"type"` // "generic", "tls", "docker", etc.
    Data        map[string][]byte      `json:"data"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type TenantSecretData struct {
    Name      string            `json:"name"`
    TenantID  string            `json:"tenant_id"`
    Type      string            `json:"type"`
    Data      map[string][]byte `json:"data"`
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

type SecretMetadata struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Type      string    `json:"type"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type TenantSecretMetadata struct {
    Name      string    `json:"name"`
    TenantID  string    `json:"tenant_id"`
    Type      string    `json:"type"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type SecretUpdates struct {
    Data     *map[string][]byte      `json:"data,omitempty"`
    Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

type SecretFilters struct {
    Type     []string `json:"type,omitempty"`
    Name     string   `json:"name,omitempty"`
    Limit    int      `json:"limit,omitempty"`
    Offset   int      `json:"offset,omitempty"`
}
```

## Data Models

### Core Data Structures

The toolkit defines several core data structures that are used across interfaces:

```go
// Common Types
type TimePeriod struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}

type Filters struct {
    Limit  int                    `json:"limit,omitempty"`
    Offset int                    `json:"offset,omitempty"`
    Sort   string                 `json:"sort,omitempty"`
    Where  map[string]interface{} `json:"where,omitempty"`
}

type TenantFilters struct {
    Filters
    Status []string `json:"status,omitempty"`
    Tier   []string `json:"tier,omitempty"`
}

type UserFilters struct {
    Filters
    TenantID []string `json:"tenant_id,omitempty"`
    Active   *bool    `json:"active,omitempty"`
}

// Update Types
type TenantUpdates struct {
    Name       *string                 `json:"name,omitempty"`
    Tier       *string                 `json:"tier,omitempty"`
    Status     *string                 `json:"status,omitempty"`
    OwnerEmail *string                 `json:"owner_email,omitempty"`
    Config     *map[string]interface{} `json:"config,omitempty"`
}

type UserUpdates struct {
    Email    *string                 `json:"email,omitempty"`
    Name     *string                 `json:"name,omitempty"`
    Roles    *[]string               `json:"roles,omitempty"`
    Metadata *map[string]interface{} `json:"metadata,omitempty"`
    Active   *bool                   `json:"active,omitempty"`
}
```

### Event Schema

Standard events follow a consistent schema for inter-plane communication:

```go
// Standard Control Plane Events
const (
    EventOnboardingRequest   = "opensbt_onboardingRequest"
    EventOffboardingRequest  = "opensbt_offboardingRequest"
    EventActivateRequest     = "opensbt_activateRequest"
    EventDeactivateRequest   = "opensbt_deactivateRequest"
    EventTenantUserCreated   = "opensbt_tenantUserCreated"
    EventTenantUserDeleted   = "opensbt_tenantUserDeleted"
    EventBillingSuccess      = "opensbt_billingSuccess"
    EventBillingFailure      = "opensbt_billingFailure"
)

// Standard Application Plane Events
const (
    EventOnboardingSuccess   = "opensbt_onboardingSuccess"
    EventOnboardingFailure   = "opensbt_onboardingFailure"
    EventOffboardingSuccess  = "opensbt_offboardingSuccess"
    EventOffboardingFailure  = "opensbt_offboardingFailure"
    EventProvisionSuccess    = "opensbt_provisionSuccess"
    EventProvisionFailure    = "opensbt_provisionFailure"
    EventDeprovisionSuccess  = "opensbt_deprovisionSuccess"
    EventDeprovisionFailure  = "opensbt_deprovisionFailure"
    EventActivateSuccess     = "opensbt_activateSuccess"
    EventActivateFailure     = "opensbt_activateFailure"
    EventDeactivateSuccess   = "opensbt_deactivateSuccess"
    EventDeactivateFailure   = "opensbt_deactivateFailure"
    EventIngestUsage         = "opensbt_ingestUsage"
    
    // Event-Driven State Machine Events
    EventGitCommitted        = "opensbt_gitCommitted"        // App Plane committed to Git
    EventArgoSyncStarted     = "opensbt_argoSyncStarted"     // ArgoCD webhook: sync started
    EventArgoSyncCompleted   = "opensbt_argoSyncCompleted"   // ArgoCD webhook: sync completed
    EventArgoHealthChanged   = "opensbt_argoHealthChanged"   // ArgoCD webhook: health status changed
)

// Tenant Status Constants - Event-Driven State Machine
// Key Insight: Git = desired state, ArgoCD = executor, PostgreSQL = business truth
const (
    TenantStatusCreating      = "CREATING"       // Control Plane saved tenant, published NATS event
    TenantStatusGitCommitted  = "GIT_COMMITTED"  // App Plane committed values.yaml to Git
    TenantStatusSyncing       = "SYNCING"        // ArgoCD is applying Kubernetes resources
    TenantStatusReady         = "READY"          // ArgoCD reports Healthy, tenant is active
    TenantStatusFailed        = "FAILED"         // ArgoCD reports Degraded, provisioning failed
    TenantStatusSuspended     = "SUSPENDED"      // Tenant temporarily disabled
    TenantStatusDeprovisioning = "DEPROVISIONING" // Tenant deletion in progress
    TenantStatusDeleted       = "DELETED"        // Tenant fully removed
)

// Event Detail Schemas
type OnboardingRequestDetail struct {
    TenantID string                 `json:"tenantId"`
    Tier     string                 `json:"tier"`
    Name     string                 `json:"name"`
    Email    string                 `json:"email"`
    Config   map[string]interface{} `json:"config,omitempty"`
}

type ProvisionSuccessDetail struct {
    TenantID      string            `json:"tenantId"`
    Resources     []Resource        `json:"resources"`
    GitCommitHash string            `json:"gitCommitHash"`
    Metadata      map[string]string `json:"metadata"`
}

// Event-Driven State Machine Event Details
type GitCommittedDetail struct {
    TenantID      string    `json:"tenantId"`
    GitCommitHash string    `json:"gitCommitHash"`
    GitRepository string    `json:"gitRepository"`
    HelmValuesPath string   `json:"helmValuesPath"`
    Timestamp     time.Time `json:"timestamp"`
}

type ArgoSyncStatusDetail struct {
    TenantID         string    `json:"tenantId"`
    ApplicationName  string    `json:"applicationName"`
    SyncStatus       string    `json:"syncStatus"`       // Synced, OutOfSync, Unknown
    HealthStatus     string    `json:"healthStatus"`     // Healthy, Progressing, Degraded, Suspended, Missing, Unknown
    SyncRevision     string    `json:"syncRevision"`
    Message          string    `json:"message,omitempty"`
    Timestamp        time.Time `json:"timestamp"`
}
```

## Security and Isolation Mechanisms

### Multi-Tenant Security Architecture

The toolkit implements defense-in-depth security with tenant isolation at multiple layers:

#### 1. Authentication Layer
- JWT-based authentication with tenant context binding
- Token validation with tenant scope enforcement
- Role-based access control (RBAC) within tenant boundaries
- Automatic token refresh with tenant context preservation

#### 2. Authorization Layer
- Tenant-scoped permissions and roles
- Context-aware authorization checks
- Cross-tenant access prevention
- Administrative privilege separation

#### 3. Data Layer
- Tenant context enforcement in all storage operations
- Automatic tenant ID injection in database queries
- Row-level security (RLS) support for compatible databases
- Transaction isolation with tenant boundaries

#### 4. Infrastructure Layer
- Tenant-specific resource provisioning
- Namespace isolation for containerized workloads
- Network segmentation and policies
- Resource quotas and limits per tenant tier

### Security Implementation Patterns

```go
// Tenant Context Middleware
type TenantContext struct {
    TenantID   string
    TenantTier string
    UserID     string
    Roles      []string
}

// Context key for tenant information
type contextKey string
const TenantContextKey contextKey = "tenant_context"

// Middleware for extracting tenant context from JWT
func TenantContextMiddleware(auth IAuth) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractTokenFromRequest(r)
            if token == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            claims, err := auth.ValidateToken(r.Context(), token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            tenantCtx := &TenantContext{
                TenantID:   claims.TenantID,
                TenantTier: claims.TenantTier,
                UserID:     claims.UserID,
                Roles:      claims.Roles,
            }
            
            ctx := context.WithValue(r.Context(), TenantContextKey, tenantCtx)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Helper function to get tenant context from request context
func GetTenantContext(ctx context.Context) (*TenantContext, error) {
    tenantCtx, ok := ctx.Value(TenantContextKey).(*TenantContext)
    if !ok {
        return nil, errors.New("tenant context not found")
    }
    return tenantCtx, nil
}
```

### GitOps-Helm Provisioner Implementation

The GitOps-Helm provisioner implements the IProvisioner interface using a pure GitOps approach with Helm templating. This approach eliminates the need for custom operators while providing full auditability and rollback capabilities.

#### Architecture

```
Tenant Request → IProvisioner → Git Commit (Helm values.yaml) → ArgoCD Sync → Kubernetes Resources
```

#### Implementation Pattern

```go
// GitOps-Helm Provisioner Implementation
type GitOpsHelmProvisioner struct {
    gitClient      GitClient
    helmClient     HelmClient
    argoClient     ArgoClient
    secretManager  ISecretManager  // Gap 8 Fix
    tierManager    ITierManager    // Gap 12 Fix
    config         GitOpsConfig
}

type GitOpsConfig struct {
    RepoURL        string `json:"repo_url"`
    Branch         string `json:"branch"`
    BasePath       string `json:"base_path"`
    ChartPath      string `json:"chart_path"`
    ArgoAppPrefix  string `json:"argo_app_prefix"`
    WebhookURL     string `json:"webhook_url"`     // Gap 3 Fix
    WarmPoolConfig WarmPoolConfig `json:"warm_pool"` // Gap 2 Fix
}

type WarmPoolConfig struct {
    BasicTierSlots    int `json:"basic_tier_slots"`
    StandardTierSlots int `json:"standard_tier_slots"`
    RefillThreshold   int `json:"refill_threshold"`
}

func (p *GitOpsHelmProvisioner) ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    // Check if this is a warm pool tier (basic/standard)
    if req.Tier == "basic" || req.Tier == "standard" {
        return p.provisionFromWarmPool(ctx, req)
    }
    
    // For premium/enterprise tiers, use standard GitOps provisioning
    return p.provisionDedicated(ctx, req)
}

// Event-Driven State Machine Implementation
func (p *GitOpsHelmProvisioner) provisionDedicated(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    // 1. Generate Helm values based on tier configuration
    helmValues, err := p.generateTierBasedValues(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 2. Commit tenant configuration to Git
    tenantConfig := TenantConfig{
        TenantID:   req.TenantID,
        Tier:       req.Tier,
        HelmValues: helmValues,
    }
    
    commitSHA, err := p.CommitTenantConfig(ctx, req.TenantID, tenantConfig)
    if err != nil {
        return nil, err
    }
    
    // 3. Publish GitCommitted event (triggers CREATING → GIT_COMMITTED transition)
    gitCommittedEvent := Event{
        ID:         generateEventID(),
        DetailType: EventGitCommitted,
        Source:     p.eventBus.GetApplicationPlaneEventSource(),
        Time:       time.Now(),
        Detail: GitCommittedDetail{
            TenantID:       req.TenantID,
            GitCommitHash:  commitSHA,
            GitRepository:  p.config.RepoURL,
            HelmValuesPath: fmt.Sprintf("%s/tenants/%s/values.yaml", p.config.BasePath, req.TenantID),
            Timestamp:      time.Now(),
        },
    }
    
    err = p.eventBus.Publish(ctx, gitCommittedEvent)
    if err != nil {
        return nil, err
    }
    
    // 4. Create ArgoCD Application for GitOps sync
    err = p.createArgoApplication(ctx, req.TenantID, req.Tier)
    if err != nil {
        return nil, err
    }
    
    // 5. Trigger immediate ArgoCD sync via webhook (no polling!)
    err = p.TriggerWebhookSync(ctx, req.TenantID, p.config.WebhookURL)
    if err != nil {
        return nil, err
    }
    
    return &ProvisionResult{
        TenantID:      req.TenantID,
        Status:        TenantStatusGitCommitted, // State machine: GIT_COMMITTED
        GitCommitHash: commitSHA,
        Metadata: map[string]string{
            "provisioning_method": "gitops-helm-dedicated",
            "git_repository":      p.config.RepoURL,
            "argo_application":    fmt.Sprintf("%s-%s", p.config.ArgoAppPrefix, req.TenantID),
        },
        CreatedAt: time.Now(),
    }, nil
}

// Gap 2 Fix: Warm Pool Implementation
func (p *GitOpsHelmProvisioner) ClaimWarmSlot(ctx context.Context, tenantID string, tier string) (*WarmSlotResult, error) {
    // 1. Find available warm slot
    availableSlot, err := p.findAvailableWarmSlot(ctx, tier)
    if err != nil {
        return nil, err
    }
    
    // 2. Update slot configuration to assign to tenant
    slotConfig := TenantConfig{
        TenantID:   tenantID,
        Tier:       tier,
        HelmValues: p.generateWarmSlotValues(tenantID, tier, availableSlot.ID),
    }
    
    // 3. Commit updated configuration to Git
    commitSHA, err := p.CommitTenantConfig(ctx, availableSlot.ID, slotConfig)
    if err != nil {
        return nil, err
    }
    
    // 4. Trigger immediate sync via webhook
    err = p.TriggerWebhookSync(ctx, availableSlot.ID, p.config.WebhookURL)
    if err != nil {
        return nil, err
    }
    
    // 5. Mark slot as claimed in database
    err = p.markSlotAsClaimed(ctx, availableSlot.ID, tenantID)
    if err != nil {
        return nil, err
    }
    
    return &WarmSlotResult{
        SlotID:    availableSlot.ID,
        TenantID:  tenantID,
        Tier:      tier,
        Resources: availableSlot.Resources,
        ClaimedAt: time.Now(),
        Metadata: map[string]string{
            "git_commit": commitSHA,
            "slot_type":  "warm_pool",
        },
    }, nil
}

func (p *GitOpsHelmProvisioner) RefillWarmPool(ctx context.Context, tier string, targetCount int) error {
    // 1. Get current warm pool status
    status, err := p.GetWarmPoolStatus(ctx, tier)
    if err != nil {
        return err
    }
    
    // 2. Calculate how many slots to create
    slotsToCreate := targetCount - status.TotalSlots
    if slotsToCreate <= 0 {
        return nil // Pool is already at target
    }
    
    // 3. Create new warm slots
    for i := 0; i < slotsToCreate; i++ {
        slotID := fmt.Sprintf("warm-pool-%s-%d", tier, status.TotalSlots+i+1)
        
        // Generate default Helm values for warm slot
        helmValues := p.generateWarmSlotValues("", tier, slotID)
        helmValues["assigned"] = false
        
        // Commit warm slot configuration
        config := TenantConfig{
            TenantID:   "", // Unassigned
            Tier:       tier,
            HelmValues: helmValues,
        }
        
        _, err := p.CommitTenantConfig(ctx, slotID, config)
        if err != nil {
            return err
        }
    }
    
    return nil
}

func (p *GitOpsHelmProvisioner) GetWarmPoolStatus(ctx context.Context, tier string) (*WarmPoolStatus, error) {
    // Query Git repository for warm pool slots
    slots, err := p.listWarmPoolSlots(ctx, tier)
    if err != nil {
        return nil, err
    }
    
    availableCount := 0
    for _, slot := range slots {
        if !slot.Assigned {
            availableCount++
        }
    }
    
    return &WarmPoolStatus{
        Tier:           tier,
        AvailableSlots: availableCount,
        TotalSlots:     len(slots),
        TargetSlots:    p.getTargetSlotCount(tier),
        LastRefill:     p.getLastRefillTime(ctx, tier),
    }, nil
}

// Gap 3 Fix: Webhook Sync Implementation
func (p *GitOpsHelmProvisioner) TriggerSync(ctx context.Context, tenantID string) error {
    argoAppName := fmt.Sprintf("%s-%s", p.config.ArgoAppPrefix, tenantID)
    return p.argoClient.SyncApplication(ctx, argoAppName)
}

func (p *GitOpsHelmProvisioner) TriggerWebhookSync(ctx context.Context, tenantID string, webhookURL string) error {
    // 1. Trigger ArgoCD sync via API
    err := p.TriggerSync(ctx, tenantID)
    if err != nil {
        return err
    }
    
    // 2. Send webhook notification for immediate reconciliation
    webhookPayload := map[string]interface{}{
        "tenant_id":   tenantID,
        "action":      "sync",
        "timestamp":   time.Now(),
        "source":      "gitops-helm-provisioner",
    }
    
    return p.sendWebhook(ctx, webhookURL, webhookPayload)
}

func (p *GitOpsHelmProvisioner) generateHelmValues(req ProvisionRequest) map[string]interface{} {
    // Get tier configuration
    tierConfig, err := p.tierManager.GetTier(context.Background(), req.Tier)
    if err != nil {
        // Fallback to default values
        return p.generateDefaultHelmValues(req)
    }
    
    values := map[string]interface{}{
        "tenant": map[string]interface{}{
            "id":   req.TenantID,
            "name": req.Name,
            "tier": req.Tier,
        },
        "resources": map[string]interface{}{
            "cpu":     tierConfig.Quotas.CPU,
            "memory":  tierConfig.Quotas.Memory,
            "storage": fmt.Sprintf("%dGi", tierConfig.Quotas.StorageGB),
        },
        "quotas": map[string]interface{}{
            "users":        tierConfig.Quotas.Users,
            "api_requests": tierConfig.Quotas.APIRequests,
        },
        "features": tierConfig.Features,
    }
    
    // Add custom configuration from request
    if req.Config != nil {
        values["custom"] = req.Config
    }
    
    return values
}

func (p *GitOpsHelmProvisioner) UpdateTenantResources(ctx context.Context, req UpdateRequest) (*UpdateResult, error) {
    // 1. Read current Helm values
    tenantPath := fmt.Sprintf("%s/tenants/%s/values.yaml", p.config.BasePath, req.TenantID)
    currentValues, err := p.gitClient.ReadFile(ctx, p.config.RepoURL, p.config.Branch, tenantPath)
    if err != nil {
        return nil, err
    }
    
    // 2. Merge updates with current values
    updatedValues := p.mergeHelmValues(currentValues, req.Updates)
    
    // 3. Commit updated values to Git
    commitSHA, err := p.gitClient.CommitFile(ctx, GitCommitRequest{
        RepoURL:  p.config.RepoURL,
        Branch:   p.config.Branch,
        FilePath: tenantPath,
        Content:  updatedValues,
        Message:  fmt.Sprintf("Update tenant %s resources", req.TenantID),
        Author:   "open-sbt-provisioner",
    })
    if err != nil {
        return nil, err
    }
    
    // 4. Trigger ArgoCD sync
    argoAppName := fmt.Sprintf("%s-%s", p.config.ArgoAppPrefix, req.TenantID)
    err = p.argoClient.SyncApplication(ctx, argoAppName)
    if err != nil {
        return nil, err
    }
    
    return &UpdateResult{
        TenantID:      req.TenantID,
        GitCommitHash: commitSHA,
        Status:        "updated",
        UpdatedAt:     time.Now(),
    }, nil
}

func (p *GitOpsHelmProvisioner) RollbackTenantConfig(ctx context.Context, tenantID string, commitHash string) error {
    // 1. Revert Git commit
    err := p.gitClient.RevertCommit(ctx, p.config.RepoURL, p.config.Branch, commitHash)
    if err != nil {
        return err
    }
    
    // 2. Trigger ArgoCD sync to apply rollback
    argoAppName := fmt.Sprintf("%s-%s", p.config.ArgoAppPrefix, tenantID)
    return p.argoClient.SyncApplication(ctx, argoAppName)
}
```

#### Universal Tenant Helm Chart

The GitOps-Helm provisioner uses a universal Helm chart that can provision any tenant tier through values-based configuration:

```yaml
# base-charts/tenant-factory/Chart.yaml
apiVersion: v2
name: tenant-factory
description: Universal Helm chart for tenant provisioning
version: 1.0.0

# base-charts/tenant-factory/values.yaml
tenant:
  id: ""
  name: ""
  tier: "basic"

resources:
  cpu: "1"
  memory: "2Gi"
  storage: "10Gi"

replicas: 1
dedicated: false

database:
  enabled: true
  instances: 1
  storage: "10Gi"

monitoring:
  enabled: true
  
networking:
  enabled: true
  ingress: true

custom: {}
```

```yaml
# base-charts/tenant-factory/templates/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-{{ .Values.tenant.id }}
  labels:
    tenant-id: {{ .Values.tenant.id }}
    tenant-tier: {{ .Values.tenant.tier }}
    managed-by: open-sbt

---
# base-charts/tenant-factory/templates/resourcequota.yaml
{{- if .Values.resources }}
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-{{ .Values.tenant.id }}-quota
  namespace: tenant-{{ .Values.tenant.id }}
spec:
  hard:
    requests.cpu: {{ .Values.resources.cpu }}
    requests.memory: {{ .Values.resources.memory }}
    persistentvolumeclaims: "10"
    {{- if .Values.resources.storage }}
    requests.storage: {{ .Values.resources.storage }}
    {{- end }}
{{- end }}

---
# base-charts/tenant-factory/templates/database.yaml
{{- if .Values.database.enabled }}
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: {{ .Values.tenant.id }}-db
  namespace: tenant-{{ .Values.tenant.id }}
spec:
  instances: {{ .Values.database.instances | default 1 }}
  
  postgresql:
    parameters:
      shared_preload_libraries: "pg_stat_statements"
      
  bootstrap:
    initdb:
      database: {{ .Values.tenant.id | replace "-" "_" }}
      owner: {{ .Values.tenant.id | replace "-" "_" }}
      
  storage:
    size: {{ .Values.database.storage | default "10Gi" }}
    storageClass: fast-ssd
    
  monitoring:
    enabled: {{ .Values.monitoring.enabled }}
{{- end }}
```

This approach provides:
- **Pure GitOps**: All changes tracked in Git with full audit trail
- **No Custom Operators**: Uses standard Helm + ArgoCD without custom controllers
- **Tier Flexibility**: Single chart supports all tenant tiers through values
- **Rollback Capability**: Git-based rollback with automatic ArgoCD sync
- **Audit Trail**: Every change is a Git commit with author and timestamp
- **Warm Pool Optimization**: Sub-second onboarding for basic/standard tiers
- **Webhook-Driven Sync**: Instant reconciliation without polling delays
- **Secure Secret Management**: HashiCorp Vault integration for sensitive data

#### Secret Management Integration (Gap 8 Fix)

The GitOps-Helm provisioner integrates with HashiCorp Vault for secure secret management:

```go
// Secret management in GitOps workflow
func (p *GitOpsHelmProvisioner) handleTenantSecrets(ctx context.Context, tenantID string, secrets map[string]interface{}) error {
    for secretName, secretData := range secrets {
        // Store sensitive data in Vault
        tenantSecret := TenantSecretData{
            Name:     secretName,
            TenantID: tenantID,
            Type:     "generic",
            Data:     convertToByteMap(secretData),
        }
        
        err := p.secretManager.CreateTenantSecret(ctx, tenantID, tenantSecret)
        if err != nil {
            return err
        }
        
        // Create Vault reference in Helm values
        vaultRef := map[string]interface{}{
            "vault": map[string]interface{}{
                "enabled": true,
                "path":    fmt.Sprintf("tenant/%s/%s", tenantID, secretName),
                "role":    fmt.Sprintf("tenant-%s", tenantID),
            },
        }
        
        // Add vault reference to Helm values instead of raw secret data
        p.addVaultReferenceToValues(tenantID, secretName, vaultRef)
    }
    
    return nil
}

// Vault integration for secret encryption in Git
func (p *GitOpsHelmProvisioner) encryptSecretsForGit(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
    encryptedData := make(map[string]interface{})
    
    for key, value := range data {
        if p.isSensitiveField(key) {
            // Encrypt sensitive data using Vault
            valueBytes, _ := json.Marshal(value)
            encrypted, err := p.secretManager.EncryptForGit(ctx, valueBytes)
            if err != nil {
                return nil, err
            }
            encryptedData[key] = base64.StdEncoding.EncodeToString(encrypted)
        } else {
            encryptedData[key] = value
        }
    }
    
    return encryptedData, nil
}
```

#### Tier-Based Configuration (Gap 12 Fix)

The provisioner uses the ITierManager interface for tier-based resource allocation:

```go
// Tier-based provisioning with formal tier definitions
func (p *GitOpsHelmProvisioner) generateTierBasedValues(ctx context.Context, req ProvisionRequest) (map[string]interface{}, error) {
    // Get tier configuration from tier manager
    tierConfig, err := p.tierManager.GetTier(ctx, req.Tier)
    if err != nil {
        return nil, fmt.Errorf("failed to get tier configuration: %w", err)
    }
    
    // Validate tenant usage against tier quotas
    currentUsage := p.getCurrentTenantUsage(ctx, req.TenantID)
    err = p.tierManager.ValidateTierQuota(ctx, req.Tier, currentUsage)
    if err != nil {
        return nil, fmt.Errorf("tier quota validation failed: %w", err)
    }
    
    // Generate Helm values based on tier configuration
    values := map[string]interface{}{
        "tenant": map[string]interface{}{
            "id":   req.TenantID,
            "name": req.Name,
            "tier": req.Tier,
        },
        "resources": map[string]interface{}{
            "cpu":     tierConfig.Quotas.CPU,
            "memory":  tierConfig.Quotas.Memory,
            "storage": fmt.Sprintf("%dGi", tierConfig.Quotas.StorageGB),
        },
        "quotas": map[string]interface{}{
            "users":        tierConfig.Quotas.Users,
            "api_requests": tierConfig.Quotas.APIRequests,
        },
        "features": tierConfig.Features,
        "pricing":  tierConfig.Pricing,
    }
    
    // Add tier-specific custom quotas
    if len(tierConfig.Quotas.Custom) > 0 {
        values["custom_quotas"] = tierConfig.Quotas.Custom
    }
    
    return values, nil
}
```

```go
// Storage operations with automatic tenant context
func (s *StorageImpl) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
    tenantCtx, err := GetTenantContext(ctx)
    if err != nil {
        return nil, err
    }
    
    // Enforce tenant isolation - users can only access their own tenant
    if tenantCtx.TenantID != tenantID {
        return nil, errors.New("access denied: cross-tenant access not allowed")
    }
    
    return s.db.GetTenant(ctx, tenantID)
}

// Provisioner operations with GitOps workflow
func (p *GitOpsHelmProvisioner) ProvisionTenant(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
    tenantCtx, err := GetTenantContext(ctx)
    if err != nil {
        return nil, err
    }
    
    // Generate tier-based Helm values
    helmValues := p.generateHelmValues(req)
    
    // Commit Helm values to Git repository
    commitSHA, err := p.CommitTenantConfig(ctx, req.TenantID, TenantConfig{
        TenantID:   req.TenantID,
        Tier:       req.Tier,
        HelmValues: helmValues,
    })
    if err != nil {
        return nil, err
    }
    
    // Create ArgoCD Application for GitOps sync
    err = p.createArgoApplication(ctx, req.TenantID, req.Tier)
    if err != nil {
        return nil, err
    }
    
    return &ProvisionResult{
        TenantID:      req.TenantID,
        Status:        "provisioning",
        GitCommitHash: commitSHA,
        Metadata: map[string]string{
            "provisioning_method": "gitops-helm",
            "chart_version":       "1.0.0",
        },
        CreatedAt: time.Now(),
    }, nil
}
```

## Error Handling

### Error Types and Patterns

The toolkit defines standard error types for consistent error handling across implementations:

```go
// Standard Error Types
type ErrorType string

const (
    ErrorTypeValidation     ErrorType = "VALIDATION_ERROR"
    ErrorTypeNotFound       ErrorType = "NOT_FOUND"
    ErrorTypeUnauthorized   ErrorType = "UNAUTHORIZED"
    ErrorTypeForbidden      ErrorType = "FORBIDDEN"
    ErrorTypeConflict       ErrorType = "CONFLICT"
    ErrorTypeInternal       ErrorType = "INTERNAL_ERROR"
    ErrorTypeTimeout        ErrorType = "TIMEOUT"
    ErrorTypeRateLimit      ErrorType = "RATE_LIMIT"
)

// Standard Error Structure
type Error struct {
    Type       ErrorType              `json:"type"`
    Code       string                 `json:"code"`
    Message    string                 `json:"message"`
    Details    map[string]interface{} `json:"details,omitempty"`
    Cause      error                  `json:"-"`
    TenantID   string                 `json:"tenant_id,omitempty"`
    RequestID  string                 `json:"request_id,omitempty"`
    Timestamp  time.Time              `json:"timestamp"`
}

func (e *Error) Error() string {
    return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// Error Constructors
func NewValidationError(code, message string, details map[string]interface{}) *Error {
    return &Error{
        Type:      ErrorTypeValidation,
        Code:      code,
        Message:   message,
        Details:   details,
        Timestamp: time.Now(),
    }
}

func NewNotFoundError(resource, id string) *Error {
    return &Error{
        Type:    ErrorTypeNotFound,
        Code:    "RESOURCE_NOT_FOUND",
        Message: fmt.Sprintf("%s with ID %s not found", resource, id),
        Details: map[string]interface{}{
            "resource": resource,
            "id":       id,
        },
        Timestamp: time.Now(),
    }
}

func NewUnauthorizedError(message string) *Error {
    return &Error{
        Type:      ErrorTypeUnauthorized,
        Code:      "UNAUTHORIZED",
        Message:   message,
        Timestamp: time.Now(),
    }
}
```

### Error Handling Patterns

```go
// Interface implementations should return structured errors
func (s *StorageImpl) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
    if tenantID == "" {
        return nil, NewValidationError("INVALID_TENANT_ID", "Tenant ID cannot be empty", nil)
    }
    
    tenant, err := s.db.QueryTenant(ctx, tenantID)
    if err != nil {
        if isNotFoundError(err) {
            return nil, NewNotFoundError("tenant", tenantID)
        }
        return nil, &Error{
            Type:      ErrorTypeInternal,
            Code:      "DATABASE_ERROR",
            Message:   "Failed to retrieve tenant",
            Cause:     err,
            Timestamp: time.Now(),
        }
    }
    
    return tenant, nil
}

// Event handlers with idempotency (Gap 5 Fix - Inbox Pattern)
func (app *ApplicationPlane) handleOnboardingRequest(ctx context.Context, event Event) error {
    // Gap 5 Fix: Check if event was already processed (Inbox Pattern)
    processed, err := app.storage.IsEventProcessed(ctx, event.ID)
    if err != nil {
        return NewInternalError("EVENT_IDEMPOTENCY_CHECK_FAILED", "Failed to check event processing status", err)
    }
    
    if processed {
        // Event already processed, acknowledge and return
        log.WithFields(log.Fields{
            "event_id": event.ID,
            "event_type": event.DetailType,
        }).Info("Event already processed, skipping duplicate")
        return nil
    }
    
    // Record event as being processed (prevents duplicate processing)
    err = app.storage.RecordProcessedEvent(ctx, event.ID)
    if err != nil {
        // If this fails due to unique constraint violation, another instance processed it
        if isDuplicateKeyError(err) {
            log.WithFields(log.Fields{
                "event_id": event.ID,
                "event_type": event.DetailType,
            }).Info("Event processed by another instance, skipping")
            return nil
        }
        return NewInternalError("EVENT_RECORD_FAILED", "Failed to record event processing", err)
    }
    
    var detail OnboardingRequestDetail
    if err := json.Unmarshal(event.Detail, &detail); err != nil {
        return NewValidationError("INVALID_EVENT_DETAIL", "Failed to parse onboarding request", map[string]interface{}{
            "event_id": event.ID,
            "error":    err.Error(),
        })
    }
    
    result, err := app.provisioner.ProvisionTenant(ctx, ProvisionRequest{
        TenantID: detail.TenantID,
        Tier:     detail.Tier,
        Name:     detail.Name,
        Email:    detail.Email,
    })
    
    if err != nil {
        // Publish failure event
        failureEvent := Event{
            ID:         generateEventID(),
            DetailType: EventProvisionFailure,
            Source:     app.eventBus.GetApplicationPlaneEventSource(),
            Time:       time.Now(),
            Detail: map[string]interface{}{
                "tenantId": detail.TenantID,
                "error":    err.Error(),
            },
        }
        
        app.eventBus.Publish(ctx, failureEvent)
        return err
    }
    
    // Publish success event
    successEvent := Event{
        ID:         generateEventID(),
        DetailType: EventProvisionSuccess,
        Source:     app.eventBus.GetApplicationPlaneEventSource(),
        Time:       time.Now(),
        Detail: ProvisionSuccessDetail{
            TenantID:      result.TenantID,
            Resources:     result.Resources,
            GitCommitHash: result.GitCommitHash,
            Metadata:      result.Metadata,
        },
    }
    
    return app.eventBus.Publish(ctx, successEvent)
}
```

### Orphaned Infrastructure Handling (Gap 6 Fix - Active Reconciliation Loop)

The Control Plane implements an active reconciliation loop to detect and resolve orphaned infrastructure when the Application Plane fails to publish status events.

```go
// Control Plane Reconciliation Loop (Gap 6 Fix)
type ControlPlaneReconciler struct {
    storage     IStorage
    provisioner IProvisioner
    eventBus    IEventBus
    config      ReconcilerConfig
}

type ReconcilerConfig struct {
    ReconcileInterval time.Duration `json:"reconcile_interval"` // Default: 5 minutes
    StuckThreshold    time.Duration `json:"stuck_threshold"`    // Default: 5 minutes
    MaxRetries        int           `json:"max_retries"`        // Default: 3
}

func (r *ControlPlaneReconciler) Start(ctx context.Context) error {
    ticker := time.NewTicker(r.config.ReconcileInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            err := r.reconcileStuckTenants(ctx)
            if err != nil {
                log.WithError(err).Error("Failed to reconcile stuck tenants")
            }
        }
    }
}

func (r *ControlPlaneReconciler) reconcileStuckTenants(ctx context.Context) error {
    // Find tenants stuck in transitional states
    stuckStates := []string{
        TenantStatusProvisioning,
        TenantStatusDeprovisioning,
    }
    
    stuckTenants, err := r.storage.ListStuckTenants(ctx, stuckStates, r.config.StuckThreshold)
    if err != nil {
        return err
    }
    
    for _, tenant := range stuckTenants {
        err := r.reconcileTenant(ctx, tenant)
        if err != nil {
            log.WithFields(log.Fields{
                "tenant_id": tenant.ID,
                "status":    tenant.Status,
                "error":     err.Error(),
            }).Error("Failed to reconcile stuck tenant")
        }
    }
    
    return nil
}

func (r *ControlPlaneReconciler) reconcileTenant(ctx context.Context, tenant Tenant) error {
    log.WithFields(log.Fields{
        "tenant_id": tenant.ID,
        "status":    tenant.Status,
        "stuck_since": time.Since(tenant.UpdatedAt),
    }).Info("Reconciling stuck tenant")
    
    // Check actual provisioning status via IProvisioner
    provisioningStatus, err := r.provisioner.GetProvisioningStatus(ctx, tenant.ID)
    if err != nil {
        return err
    }
    
    switch tenant.Status {
    case TenantStatusProvisioning:
        return r.reconcileProvisioningTenant(ctx, tenant, provisioningStatus)
    case TenantStatusDeprovisioning:
        return r.reconcileDeprovisioningTenant(ctx, tenant, provisioningStatus)
    default:
        return nil
    }
}

func (r *ControlPlaneReconciler) reconcileProvisioningTenant(ctx context.Context, tenant Tenant, status *ProvisioningStatus) error {
    switch status.Status {
    case "synced", "healthy":
        // Infrastructure is ready, update tenant to active
        err := r.storage.UpdateTenant(ctx, tenant.ID, TenantUpdates{
            Status: &TenantStatusActive,
        })
        if err != nil {
            return err
        }
        
        // Publish synthetic success event
        successEvent := Event{
            ID:         generateEventID(),
            DetailType: EventProvisionSuccess,
            Source:     "reconciler",
            Time:       time.Now(),
            Detail: ProvisionSuccessDetail{
                TenantID:      tenant.ID,
                Resources:     status.Resources,
                GitCommitHash: status.GitCommitHash,
                Metadata: map[string]string{
                    "reconciled": "true",
                    "reason":     "stuck_tenant_recovery",
                },
            },
        }
        
        return r.eventBus.Publish(ctx, successEvent)
        
    case "degraded", "failed", "error":
        // Infrastructure failed, update tenant to failed state
        err := r.storage.UpdateTenant(ctx, tenant.ID, TenantUpdates{
            Status: &TenantStatusFailed,
        })
        if err != nil {
            return err
        }
        
        // Publish synthetic failure event
        failureEvent := Event{
            ID:         generateEventID(),
            DetailType: EventProvisionFailure,
            Source:     "reconciler",
            Time:       time.Now(),
            Detail: map[string]interface{}{
                "tenantId": tenant.ID,
                "error":    status.ErrorMessage,
                "reconciled": true,
                "reason":   "stuck_tenant_recovery",
            },
        }
        
        return r.eventBus.Publish(ctx, failureEvent)
        
    case "progressing", "syncing":
        // Still in progress, extend the timeout
        log.WithFields(log.Fields{
            "tenant_id": tenant.ID,
            "status":    status.Status,
        }).Info("Tenant still provisioning, extending timeout")
        return nil
        
    default:
        // Unknown status, mark as failed
        return r.storage.UpdateTenant(ctx, tenant.ID, TenantUpdates{
            Status: &TenantStatusFailed,
        })
    }
}

func (r *ControlPlaneReconciler) reconcileDeprovisioningTenant(ctx context.Context, tenant Tenant, status *ProvisioningStatus) error {
    if status == nil || status.Status == "not_found" {
        // Infrastructure is gone, mark tenant as deleted
        err := r.storage.UpdateTenant(ctx, tenant.ID, TenantUpdates{
            Status: &TenantStatusDeleted,
        })
        if err != nil {
            return err
        }
        
        // Publish synthetic success event
        successEvent := Event{
            ID:         generateEventID(),
            DetailType: EventDeprovisionSuccess,
            Source:     "reconciler",
            Time:       time.Now(),
            Detail: map[string]interface{}{
                "tenantId": tenant.ID,
                "reconciled": true,
                "reason":   "stuck_tenant_recovery",
            },
        }
        
        return r.eventBus.Publish(ctx, successEvent)
    }
    
    // Infrastructure still exists, continue waiting or force cleanup
    log.WithFields(log.Fields{
        "tenant_id": tenant.ID,
        "status":    status.Status,
    }).Info("Tenant still deprovisioning")
    
    return nil
}

// Helper function to check for duplicate key errors
func isDuplicateKeyError(err error) bool {
    // Implementation depends on database provider
    // For PostgreSQL: check for error code 23505
    // For other databases: check appropriate error codes
    return strings.Contains(err.Error(), "duplicate key") ||
           strings.Contains(err.Error(), "UNIQUE constraint") ||
           strings.Contains(err.Error(), "23505")
}
```

## Testing Strategy

### Testing Architecture

The toolkit provides comprehensive testing patterns using Testcontainers-Go for integration testing and property-based testing for correctness validation.

#### E2E Testing with Testcontainers-Go

```go
// Base test setup with real dependencies
type TestSuite struct {
    postgres    testcontainers.Container
    nats        testcontainers.Container
    storage     IStorage
    eventBus    IEventBus
    auth        IAuth
    controlPlane *ControlPlane
    appPlane    *ApplicationPlane
}

func (ts *TestSuite) SetupSuite() error {
    ctx := context.Background()
    
    // Start PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        testcontainers.WithEnv(map[string]string{
            "POSTGRES_DB":       "opensbt_test",
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
        }),
    )
    if err != nil {
        return err
    }
    ts.postgres = pgContainer
    
    // Start NATS container
    natsContainer, err := nats.RunContainer(ctx,
        testcontainers.WithImage("nats:2.9-alpine"),
    )
    if err != nil {
        return err
    }
    ts.nats = natsContainer
    
    // Initialize providers
    pgConnStr, _ := pgContainer.ConnectionString(ctx)
    natsURL, _ := natsContainer.Endpoint(ctx, "4222")
    
    ts.storage = postgres.NewPostgresStorage(postgres.Config{
        ConnectionString: pgConnStr,
    })
    
    ts.eventBus = nats.NewNATSEventBus(nats.Config{
        URL: fmt.Sprintf("nats://%s", natsURL),
    })
    
    ts.auth = mock.NewMockAuth()
    
    // Initialize planes
    ts.controlPlane = NewControlPlane(ControlPlaneConfig{
        Storage:  ts.storage,
        EventBus: ts.eventBus,
        Auth:     ts.auth,
    })
    
    ts.appPlane = NewApplicationPlane(ApplicationPlaneConfig{
        EventBus:    ts.eventBus,
        Provisioner: mock.NewMockProvisioner(),
    })
    
    return nil
}

func (ts *TestSuite) TearDownSuite() error {
    ctx := context.Background()
    if ts.postgres != nil {
        ts.postgres.Terminate(ctx)
    }
    if ts.nats != nil {
        ts.nats.Terminate(ctx)
    }
    return nil
}
```

#### Tenant Isolation Testing

```go
func TestTenantIsolation(t *testing.T) {
    suite := &TestSuite{}
    require.NoError(t, suite.SetupSuite())
    defer suite.TearDownSuite()
    
    ctx := context.Background()
    
    // Create two tenants
    tenant1, err := suite.controlPlane.CreateTenant(ctx, CreateTenantRequest{
        Name:  "tenant-1",
        Tier:  "basic",
        Email: "admin@tenant1.com",
    })
    require.NoError(t, err)
    
    tenant2, err := suite.controlPlane.CreateTenant(ctx, CreateTenantRequest{
        Name:  "tenant-2", 
        Tier:  "basic",
        Email: "admin@tenant2.com",
    })
    require.NoError(t, err)
    
    // Create user in tenant 1
    user1, err := suite.auth.CreateUser(ctx, User{
        Email:    "user@tenant1.com",
        Name:     "User 1",
        TenantID: tenant1.ID,
    })
    require.NoError(t, err)
    
    // Test: Tenant 1 user cannot access tenant 2 data
    tenant1Ctx := context.WithValue(ctx, TenantContextKey, &TenantContext{
        TenantID: tenant1.ID,
        UserID:   user1.ID,
    })
    
    _, err = suite.storage.GetTenant(tenant1Ctx, tenant2.ID)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cross-tenant access not allowed")
    
    // Test: Tenant 1 user can access their own tenant
    retrievedTenant, err := suite.storage.GetTenant(tenant1Ctx, tenant1.ID)
    assert.NoError(t, err)
    assert.Equal(t, tenant1.ID, retrievedTenant.ID)
}
```

#### Property-Based Testing

```go
func TestTenantCreationProperties(t *testing.T) {
    suite := &TestSuite{}
    require.NoError(t, suite.SetupSuite())
    defer suite.TearDownSuite()
    
    // Property: All created tenants should be retrievable
    property := func(name, tier, email string) bool {
        if name == "" || tier == "" || email == "" {
            return true // Skip invalid inputs
        }
        
        ctx := context.Background()
        
        tenant, err := suite.controlPlane.CreateTenant(ctx, CreateTenantRequest{
            Name:  name,
            Tier:  tier,
            Email: email,
        })
        if err != nil {
            return false
        }
        
        retrieved, err := suite.storage.GetTenant(ctx, tenant.ID)
        if err != nil {
            return false
        }
        
        return retrieved.Name == name && 
               retrieved.Tier == tier && 
               retrieved.OwnerEmail == email
    }
    
    quick.Check(property, &quick.Config{MaxCount: 100})
}

func TestEventDeliveryProperties(t *testing.T) {
    suite := &TestSuite{}
    require.NoError(t, suite.SetupSuite())
    defer suite.TearDownSuite()
    
    // Property: All published events should be delivered to subscribers
    property := func(eventType, tenantID string) bool {
        if eventType == "" || tenantID == "" {
            return true
        }
        
        ctx := context.Background()
        received := make(chan Event, 1)
        
        // Subscribe to events
        err := suite.eventBus.Subscribe(ctx, eventType, func(ctx context.Context, event Event) error {
            received <- event
            return nil
        })
        if err != nil {
            return false
        }
        
        // Publish event
        testEvent := Event{
            ID:         generateEventID(),
            DetailType: eventType,
            Source:     "test",
            Time:       time.Now(),
            Detail: map[string]interface{}{
                "tenantId": tenantID,
            },
        }
        
        err = suite.eventBus.Publish(ctx, testEvent)
        if err != nil {
            return false
        }
        
        // Verify delivery
        select {
        case receivedEvent := <-received:
            return receivedEvent.ID == testEvent.ID
        case <-time.After(5 * time.Second):
            return false
        }
    }
    
    quick.Check(property, &quick.Config{MaxCount: 50})
}
```

#### Interface Compliance Testing

```go
// Generic interface compliance test
func TestInterfaceCompliance(t *testing.T) {
    testCases := []struct {
        name      string
        provider  interface{}
        interface interface{}
    }{
        {"PostgresStorage", &postgres.PostgresStorage{}, (*IStorage)(nil)},
        {"NATSEventBus", &nats.NATSEventBus{}, (*IEventBus)(nil)},
        {"MockAuth", &mock.MockAuth{}, (*IAuth)(nil)},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            providerType := reflect.TypeOf(tc.provider)
            interfaceType := reflect.TypeOf(tc.interface).Elem()
            
            assert.True(t, providerType.Implements(interfaceType),
                "Provider %s does not implement interface %s", 
                providerType, interfaceType)
        })
    }
}
```

### Testing Best Practices

1. **Use Real Dependencies**: Testcontainers-Go provides real database and message queue instances
2. **Test Tenant Isolation**: Verify cross-tenant access is prevented at all layers
3. **Property-Based Testing**: Use randomized inputs to test invariants and edge cases
4. **Interface Compliance**: Ensure all providers correctly implement interfaces
5. **Event Testing**: Verify event delivery and ordering guarantees
6. **Error Scenarios**: Test error handling and recovery patterns
7. **Performance Testing**: Validate performance characteristics under load

## Package Structure

The toolkit follows a clean architecture pattern with clear separation of concerns:

```
open-sbt/
├── pkg/
│   ├── interfaces/              # Core interface definitions
│   │   ├── auth.go             # IAuth interface
│   │   ├── eventbus.go         # IEventBus interface  
│   │   ├── provisioner.go      # IProvisioner interface
│   │   ├── storage.go          # IStorage interface
│   │   ├── billing.go          # IBilling interface
│   │   ├── metering.go         # IMetering interface
│   │   ├── tiermanager.go      # ITierManager interface (Gap 12 Fix)
│   │   ├── secretmanager.go    # ISecretManager interface (Gap 8 Fix)
│   │   └── common.go           # Common types and constants
│   │
│   ├── providers/              # Default provider implementations
│   │   ├── auth/
│   │   │   ├── mock/           # Mock auth provider for testing
│   │   │   └── ory/            # Ory Stack implementation
│   │   ├── eventbus/
│   │   │   ├── mock/           # Mock event bus for testing
│   │   │   └── nats/           # NATS implementation
│   │   ├── provisioner/
│   │   │   ├── mock/           # Mock provisioner for testing
│   │   │   └── gitops-helm/    # GitOps + Helm implementation
│   │   ├── storage/
│   │   │   ├── mock/           # Mock storage for testing
│   │   │   ├── postgres/       # PostgreSQL implementation
│   │   │   └── memory/         # In-memory implementation for testing
│   │   ├── billing/
│   │   │   ├── mock/           # Mock billing for testing
│   │   │   └── stripe/         # Stripe implementation
│   │   ├── metering/
│   │   │   ├── mock/           # Mock metering for testing
│   │   │   └── prometheus/     # Prometheus-based implementation
│   │   ├── tiermanager/
│   │   │   ├── mock/           # Mock tier manager for testing
│   │   │   └── database/       # Database-based implementation
│   │   └── secretmanager/
│   │       ├── mock/           # Mock secret manager for testing
│   │       └── vault/          # HashiCorp Vault implementation
│   │
│   ├── controlplane/           # Control Plane implementation
│   │   ├── controlplane.go     # Main Control Plane struct
│   │   ├── tenant.go           # Tenant management service
│   │   ├── user.go             # User management service
│   │   ├── registration.go     # Tenant registration service
│   │   ├── config.go           # Configuration management
│   │   └── api/                # HTTP API handlers
│   │       ├── handlers.go     # HTTP handlers
│   │       ├── middleware.go   # HTTP middleware
│   │       └── routes.go       # Route definitions
│   │
│   ├── applicationplane/       # Application Plane implementation
│   │   ├── applicationplane.go # Main Application Plane struct
│   │   ├── provisioner.go      # Provisioning workflows
│   │   ├── events.go           # Event handlers
│   │   └── workflows/          # Complex provisioning workflows
│   │       ├── onboarding.go   # Tenant onboarding workflow
│   │       └── offboarding.go  # Tenant offboarding workflow
│   │
│   ├── events/                 # Event definitions and utilities
│   │   ├── definitions.go      # Standard event definitions
│   │   ├── publisher.go        # Event publishing utilities
│   │   └── subscriber.go       # Event subscription utilities
│   │
│   ├── security/               # Security utilities and middleware
│   │   ├── context.go          # Tenant context management
│   │   ├── middleware.go       # Security middleware
│   │   └── isolation.go        # Tenant isolation helpers
│   │
│   ├── testing/                # Testing utilities and helpers
│   │   ├── suite.go            # Base test suite
│   │   ├── containers.go       # Testcontainers helpers
│   │   ├── fixtures.go         # Test data fixtures
│   │   └── assertions.go       # Custom test assertions
│   │
│   └── utils/                  # Common utilities
│       ├── errors.go           # Error types and constructors
│       ├── logging.go          # Logging utilities
│       ├── metrics.go          # Metrics utilities
│       └── validation.go       # Input validation helpers
│
├── examples/                   # Example implementations
│   ├── basic/                  # Basic Control + Application Plane
│   │   ├── main.go
│   │   ├── config.yaml
│   │   └── README.md
│   ├── with-billing/           # Example with billing integration
│   │   ├── main.go
│   │   ├── config.yaml
│   │   └── README.md
│   └── gitops/                 # GitOps-enabled example
│       ├── main.go
│       ├── config.yaml
│       └── README.md
│
├── tests/                      # Integration and E2E tests
│   ├── integration/            # Integration tests
│   │   ├── tenant_test.go
│   │   ├── auth_test.go
│   │   └── events_test.go
│   ├── e2e/                    # End-to-end tests
│   │   ├── onboarding_test.go
│   │   ├── isolation_test.go
│   │   └── workflows_test.go
│   └── properties/             # Property-based tests
│       ├── tenant_properties_test.go
│       └── event_properties_test.go
│
├── docs/                       # Documentation
│   ├── architecture.md         # Architecture overview
│   ├── interfaces.md           # Interface documentation
│   ├── providers.md            # Provider implementation guide
│   ├── security.md             # Security patterns and best practices
│   ├── testing.md              # Testing guide
│   └── examples/               # Usage examples and tutorials
│       ├── getting-started.md
│       ├── custom-providers.md
│       └── deployment.md
│
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── Makefile                    # Build and test automation
└── README.md                   # Project overview and quick start
```

### Package Dependencies

The package structure follows these dependency rules:

1. **interfaces/**: No dependencies on other packages (pure interfaces)
2. **providers/**: Depend only on interfaces/ and external libraries
3. **controlplane/**: Depends on interfaces/ and security/
4. **applicationplane/**: Depends on interfaces/ and events/
5. **events/**: Depends only on interfaces/
6. **security/**: Depends only on interfaces/
7. **testing/**: Can depend on any package for testing purposes
8. **utils/**: Minimal dependencies, mostly standard library

This structure ensures clean separation of concerns and prevents circular dependencies while maintaining flexibility for different deployment scenarios.
## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

The following correctness properties define the expected behavior of the open-sbt toolkit across all provider implementations and deployment scenarios. These properties serve as the foundation for property-based testing and validation of the toolkit's core guarantees.

### Property 1: Interface Polymorphism

*For any* interface in the toolkit (IAuth, IEventBus, IProvisioner, IStorage, IBilling, IMetering), swapping provider implementations should not break existing functionality or require code changes in user applications.

**Validates: Requirements 1.2, 1.4**

### Property 2: Architectural Separation

*For any* Control Plane operation requiring infrastructure changes, the operation should communicate through the Event Bus only and never make direct API calls to Application Plane components.

**Validates: Requirements 2.1, 2.3, 2.5**

### Property 3: Event-Driven Communication

*For any* event published to the Event Bus, all registered subscribers should receive the event with guaranteed delivery and proper ordering for tenant-specific operations.

**Validates: Requirements 3.1, 3.2, 3.4, 3.5**

### Property 4: Tenant Lifecycle Consistency

*For any* tenant lifecycle operation (create, update, delete, activate, deactivate), the Control Plane should publish appropriate events and the Application Plane should respond with corresponding success or failure events.

**Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5**

### Property 5: Tenant Data Isolation

*For any* tenant attempting to access data, the Storage Interface should only return data belonging to that tenant and never return data from other tenants.

**Validates: Requirements 5.1, 5.2, 5.5**

### Property 6: Authentication Token Binding

*For any* successful authentication, the Auth Interface should generate tokens that bind user identity to tenant context, ensuring users can only access their assigned tenant's resources.

**Validates: Requirements 5.3**

### Property 7: GitOps Infrastructure Management

*For any* infrastructure change requested through the Provisioner, the change should be committed to Git as Helm values before being applied, providing audit trails and rollback capabilities through ArgoCD sync.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

### Property 8: Tier-Based Resource Allocation

*For any* tenant created with a specific tier, the Provisioner should allocate resources according to that tier's configuration, and tier upgrades should adjust resources without downtime.

**Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5**

### Property 9: Storage Interface Consistency

*For any* database provider implementation of IStorage, the interface should provide consistent behavior for CRUD operations, transaction management, and tenant isolation guarantees.

**Validates: Requirements 9.2, 9.3, 9.4, 9.5**

### Property 10: Metering and Billing Integration

*For any* usage event or billing operation, the Metering and Billing interfaces should record events with proper tenant context and support integration with external systems.

**Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5**

### Property 11: Tenant-Aware Observability

*For any* logging or metrics operation, the observability helpers should automatically include tenant context and work with popular monitoring systems.

**Validates: Requirements 11.1, 11.2, 11.3, 11.4, 11.5**

### Property 12: Configuration Management Hierarchy

*For any* configuration value, the Configuration System should support tenant-specific overrides of global defaults, validate changes, and provide versioning with rollback capabilities.

**Validates: Requirements 12.1, 12.2, 12.3, 12.4, 12.5**

### Property 13: Interface Compliance Validation

*For any* provider implementation, the Test Framework should validate that the implementation correctly satisfies its interface contract and maintains expected behavior.

**Validates: Requirements 8.4**

### Property 14: Cross-Tenant Access Prevention

*For any* attempt to access resources across tenant boundaries, the toolkit should log the attempt and deny access, maintaining strict tenant isolation.

**Validates: Requirements 5.5**

### Property 15: Resource Provisioning Isolation

*For any* tenant provisioning operation, the Provisioner should create isolated resources based on the tenant's tier without affecting other tenants' resources.

**Validates: Requirements 5.4**

### Property 16: Warm Pool Efficiency (Gap 2 Fix)

*For any* basic or standard tier tenant onboarding request, the Provisioner should claim an available warm slot and complete onboarding in under 2 seconds, while automatically refilling the warm pool to maintain target capacity.

**Validates: Requirements 4.1, 7.1**

### Property 17: Webhook Sync Responsiveness (Gap 3 Fix)

*For any* Git commit made by the Provisioner, a webhook should trigger immediate ArgoCD synchronization without waiting for polling intervals, ensuring infrastructure changes are applied within seconds.

**Validates: Requirements 6.5**

### Property 18: Tier-Based Resource Allocation (Gap 12 Fix)

*For any* tenant created with a specific tier, the Provisioner should allocate resources according to the formal tier configuration from ITierManager, and tier quota validation should prevent resource allocation that exceeds tier limits.

**Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5**

### Property 19: Secure Secret Management (Gap 8 Fix)

*For any* sensitive data in tenant configurations, the SecretManager should store secrets in HashiCorp Vault and only include vault references in Git, ensuring sensitive data never appears in plain text in the GitOps repository.

**Validates: Requirements 6.1, 6.4**

### Property 20: Event Idempotency (Gap 5 Fix)

*For any* event published to the Event Bus with at-least-once delivery guarantees, duplicate events should be detected and ignored using the Inbox Pattern, ensuring each event is processed exactly once even during network failures or pod restarts.

**Validates: Requirements 3.1, 3.4**

### Property 21: Orphaned Infrastructure Recovery (Gap 6 Fix)

*For any* tenant stuck in transitional states (provisioning, deprovisioning) for longer than the configured threshold, the Control Plane reconciliation loop should detect the stuck state, verify actual infrastructure status, and automatically transition the tenant to the correct final state (active, failed, deleted).

**Validates: Requirements 4.2, 4.3, 4.4**

### Property 22: Failed State Management (Gap 4 Fix)

*For any* tenant provisioning operation that fails definitively, the tenant should be transitioned to a failed state that allows platform administrators to inspect logs, fix issues, and manually trigger retries without losing the failure context.

**Validates: Requirements 4.2, 4.4**

## Error Handling

### Error Classification and Response Patterns

The toolkit implements a comprehensive error handling strategy that ensures consistent error responses across all interfaces and providers:

#### Error Categories

1. **Validation Errors**: Invalid input parameters or malformed requests
2. **Authorization Errors**: Insufficient permissions or invalid authentication
3. **Resource Errors**: Resource not found, conflicts, or capacity issues
4. **System Errors**: Internal failures, timeouts, or external service issues
5. **Tenant Isolation Errors**: Cross-tenant access attempts or isolation violations

#### Error Propagation Strategy

```go
// Error handling in Control Plane operations
func (cp *ControlPlane) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
    // Validate input
    if err := validateCreateTenantRequest(req); err != nil {
        return nil, NewValidationError("INVALID_TENANT_REQUEST", err.Error(), map[string]interface{}{
            "request": req,
        })
    }
    
    // Check authorization
    if err := cp.checkCreateTenantPermission(ctx); err != nil {
        return nil, err // Already wrapped error
    }
    
    // Create tenant
    tenant, err := cp.storage.CreateTenant(ctx, Tenant{
        Name:       req.Name,
        Tier:       req.Tier,
        OwnerEmail: req.Email,
        Status:     "pending",
    })
    if err != nil {
        return nil, &Error{
            Type:      ErrorTypeInternal,
            Code:      "TENANT_CREATION_FAILED",
            Message:   "Failed to create tenant",
            Cause:     err,
            TenantID:  "", // No tenant ID yet
            Timestamp: time.Now(),
        }
    }
    
    // Publish onboarding event
    event := Event{
        ID:         generateEventID(),
        DetailType: EventOnboardingRequest,
        Source:     cp.eventBus.GetControlPlaneEventSource(),
        Time:       time.Now(),
        Detail: OnboardingRequestDetail{
            TenantID: tenant.ID,
            Tier:     tenant.Tier,
            Name:     tenant.Name,
            Email:    tenant.OwnerEmail,
        },
    }
    
    if err := cp.eventBus.Publish(ctx, event); err != nil {
        // Rollback tenant creation
        cp.storage.DeleteTenant(ctx, tenant.ID)
        
        return nil, &Error{
            Type:      ErrorTypeInternal,
            Code:      "EVENT_PUBLISH_FAILED",
            Message:   "Failed to publish onboarding event",
            Cause:     err,
            TenantID:  tenant.ID,
            Timestamp: time.Now(),
        }
    }
    
    return tenant, nil
}
```

#### Error Recovery Patterns

```go
// Retry pattern for transient failures
func (eb *EventBusImpl) PublishWithRetry(ctx context.Context, event Event, maxRetries int) error {
    var lastErr error
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := eb.Publish(ctx, event)
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        // Check if error is retryable
        if !isRetryableError(err) {
            break
        }
        
        // Exponential backoff
        backoff := time.Duration(attempt) * time.Second
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            continue
        }
    }
    
    return &Error{
        Type:      ErrorTypeTimeout,
        Code:      "PUBLISH_RETRY_EXHAUSTED",
        Message:   fmt.Sprintf("Failed to publish event after %d attempts", maxRetries),
        Cause:     lastErr,
        Timestamp: time.Now(),
    }
}

// Circuit breaker pattern for external service calls
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    failures    int
    lastFailure time.Time
    state       string // "closed", "open", "half-open"
    mutex       sync.RWMutex
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    cb.mutex.RLock()
    state := cb.state
    failures := cb.failures
    lastFailure := cb.lastFailure
    cb.mutex.RUnlock()
    
    // Check circuit state
    if state == "open" {
        if time.Since(lastFailure) < cb.timeout {
            return &Error{
                Type:    ErrorTypeTimeout,
                Code:    "CIRCUIT_BREAKER_OPEN",
                Message: "Circuit breaker is open",
            }
        }
        // Try to transition to half-open
        cb.mutex.Lock()
        cb.state = "half-open"
        cb.mutex.Unlock()
    }
    
    // Execute function
    err := fn()
    
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        
        return err
    }
    
    // Success - reset circuit
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

## Testing Strategy

### Comprehensive Testing Approach

The open-sbt toolkit employs a multi-layered testing strategy that ensures reliability, correctness, and performance across all provider implementations and deployment scenarios.

#### Testing Pyramid

1. **Unit Tests**: Fast, isolated tests for individual components
2. **Integration Tests**: Tests with real dependencies using Testcontainers-Go
3. **Property-Based Tests**: Randomized testing of correctness properties
4. **End-to-End Tests**: Full workflow testing across Control and Application Planes
5. **Performance Tests**: Load testing and benchmarking
6. **Chaos Tests**: Failure injection and resilience testing

#### Property-Based Testing Implementation

```go
// Property test for tenant isolation
func TestTenantIsolationProperty(t *testing.T) {
    suite := setupTestSuite(t)
    defer suite.tearDown()
    
    property := func(tenant1Data, tenant2Data TenantTestData) bool {
        if !tenant1Data.Valid() || !tenant2Data.Valid() || tenant1Data.ID == tenant2Data.ID {
            return true // Skip invalid or identical tenants
        }
        
        ctx := context.Background()
        
        // Create two tenants
        t1, err := suite.controlPlane.CreateTenant(ctx, tenant1Data.CreateRequest())
        if err != nil {
            return false
        }
        
        t2, err := suite.controlPlane.CreateTenant(ctx, tenant2Data.CreateRequest())
        if err != nil {
            return false
        }
        
        // Create data for tenant 1
        data1, err := suite.createTenantData(ctx, t1.ID, tenant1Data.SampleData())
        if err != nil {
            return false
        }
        
        // Try to access tenant 1's data from tenant 2's context
        t2Ctx := context.WithValue(ctx, TenantContextKey, &TenantContext{
            TenantID: t2.ID,
            UserID:   "user-" + t2.ID,
        })
        
        _, err = suite.storage.GetTenantData(t2Ctx, data1.ID)
        
        // Should always fail - tenant 2 cannot access tenant 1's data
        return err != nil && strings.Contains(err.Error(), "cross-tenant access")
    }
    
    quick.Check(property, &quick.Config{
        MaxCount: 100,
        Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
    })
}

// Property test for event delivery guarantees
func TestEventDeliveryProperty(t *testing.T) {
    suite := setupTestSuite(t)
    defer suite.tearDown()
    
    property := func(eventCount int, subscriberCount int) bool {
        if eventCount <= 0 || eventCount > 50 || subscriberCount <= 0 || subscriberCount > 10 {
            return true // Skip invalid ranges
        }
        
        ctx := context.Background()
        received := make([][]Event, subscriberCount)
        
        // Create subscribers
        for i := 0; i < subscriberCount; i++ {
            idx := i
            received[idx] = make([]Event, 0)
            
            err := suite.eventBus.Subscribe(ctx, "test_event", func(ctx context.Context, event Event) error {
                received[idx] = append(received[idx], event)
                return nil
            })
            if err != nil {
                return false
            }
        }
        
        // Publish events
        publishedEvents := make([]Event, eventCount)
        for i := 0; i < eventCount; i++ {
            event := Event{
                ID:         fmt.Sprintf("event-%d", i),
                DetailType: "test_event",
                Source:     "test",
                Time:       time.Now(),
                Detail: map[string]interface{}{
                    "sequence": i,
                },
            }
            
            err := suite.eventBus.Publish(ctx, event)
            if err != nil {
                return false
            }
            
            publishedEvents[i] = event
        }
        
        // Wait for delivery
        time.Sleep(1 * time.Second)
        
        // Verify all subscribers received all events
        for i := 0; i < subscriberCount; i++ {
            if len(received[i]) != eventCount {
                return false
            }
            
            // Verify event order and content
            for j := 0; j < eventCount; j++ {
                if received[i][j].ID != publishedEvents[j].ID {
                    return false
                }
            }
        }
        
        return true
    }
    
    quick.Check(property, &quick.Config{
        MaxCount: 20,
        Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
    })
}
```

#### Integration Testing with Testcontainers-Go

```go
// Comprehensive integration test setup
type IntegrationTestSuite struct {
    containers struct {
        postgres testcontainers.Container
        nats     testcontainers.Container
        redis    testcontainers.Container
    }
    
    providers struct {
        storage  IStorage
        eventBus IEventBus
        auth     IAuth
        billing  IBilling
        metering IMetering
    }
    
    planes struct {
        control     *ControlPlane
        application *ApplicationPlane
    }
}

func (suite *IntegrationTestSuite) SetupSuite() error {
    ctx := context.Background()
    
    // Start PostgreSQL for storage
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        testcontainers.WithEnv(map[string]string{
            "POSTGRES_DB":       "opensbt_integration",
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
        }),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        return err
    }
    suite.containers.postgres = pgContainer
    
    // Start NATS for event bus
    natsContainer, err := nats.RunContainer(ctx,
        testcontainers.WithImage("nats:2.9-alpine"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("Server is ready").
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        return err
    }
    suite.containers.nats = natsContainer
    
    // Start Redis for caching (optional)
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("Ready to accept connections").
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        return err
    }
    suite.containers.redis = redisContainer
    
    // Initialize providers with real implementations
    pgConnStr, _ := pgContainer.ConnectionString(ctx)
    natsURL, _ := natsContainer.Endpoint(ctx, "4222")
    redisURL, _ := redisContainer.Endpoint(ctx, "6379")
    
    suite.providers.storage = postgres.NewPostgresStorage(postgres.Config{
        ConnectionString: pgConnStr,
    })
    
    suite.providers.eventBus = nats.NewNATSEventBus(nats.Config{
        URL: fmt.Sprintf("nats://%s", natsURL),
    })
    
    suite.providers.auth = ory.NewOryAuth(ory.Config{
        // Mock Ory endpoints for testing
        KratosURL: "http://mock-kratos:4433",
        HydraURL:  "http://mock-hydra:4444",
    })
    
    suite.providers.billing = mock.NewMockBilling()
    suite.providers.metering = mock.NewMockMetering()
    
    // Initialize Control Plane
    suite.planes.control = NewControlPlane(ControlPlaneConfig{
        Storage:  suite.providers.storage,
        EventBus: suite.providers.eventBus,
        Auth:     suite.providers.auth,
        Billing:  suite.providers.billing,
    })
    
    // Initialize Application Plane
    suite.planes.application = NewApplicationPlane(ApplicationPlaneConfig{
        EventBus:    suite.providers.eventBus,
        Provisioner: mock.NewMockProvisioner(),
        Metering:    suite.providers.metering,
    })
    
    // Start both planes
    go suite.planes.control.Start(ctx)
    go suite.planes.application.Start(ctx)
    
    // Wait for startup
    time.Sleep(2 * time.Second)
    
    return nil
}

func (suite *IntegrationTestSuite) TearDownSuite() error {
    ctx := context.Background()
    
    // Stop planes
    if suite.planes.control != nil {
        suite.planes.control.Stop(ctx)
    }
    if suite.planes.application != nil {
        suite.planes.application.Stop(ctx)
    }
    
    // Stop containers
    if suite.containers.postgres != nil {
        suite.containers.postgres.Terminate(ctx)
    }
    if suite.containers.nats != nil {
        suite.containers.nats.Terminate(ctx)
    }
    if suite.containers.redis != nil {
        suite.containers.redis.Terminate(ctx)
    }
    
    return nil
}

// Full workflow integration test
func (suite *IntegrationTestSuite) TestCompleteOnboardingWorkflow() {
    ctx := context.Background()
    
    // Create tenant registration
    registration, err := suite.planes.control.CreateTenantRegistration(ctx, TenantRegistrationRequest{
        Name:  "Test Corp",
        Email: "admin@testcorp.com",
        Tier:  "premium",
    })
    suite.Require().NoError(err)
    suite.Equal("pending", registration.Status)
    
    // Wait for provisioning to complete
    var tenant *Tenant
    for i := 0; i < 30; i++ { // Wait up to 30 seconds
        time.Sleep(1 * time.Second)
        
        t, err := suite.providers.storage.GetTenant(ctx, registration.TenantID)
        if err == nil && t.Status == "active" {
            tenant = t
            break
        }
    }
    
    suite.Require().NotNil(tenant, "Tenant should be provisioned and active")
    suite.Equal("active", tenant.Status)
    suite.Equal("premium", tenant.Tier)
    
    // Create user in tenant
    user, err := suite.providers.auth.CreateUser(ctx, User{
        Email:    "user@testcorp.com",
        Name:     "Test User",
        TenantID: tenant.ID,
        Roles:    []string{"user"},
    })
    suite.Require().NoError(err)
    
    // Test authentication
    token, err := suite.providers.auth.AuthenticateUser(ctx, Credentials{
        Email:    user.Email,
        Password: "test-password",
    })
    suite.Require().NoError(err)
    suite.NotEmpty(token.AccessToken)
    
    // Validate token contains tenant context
    claims, err := suite.providers.auth.ValidateToken(ctx, token.AccessToken)
    suite.Require().NoError(err)
    suite.Equal(tenant.ID, claims.TenantID)
    suite.Equal("premium", claims.TenantTier)
    
    // Test tenant isolation
    userCtx := context.WithValue(ctx, TenantContextKey, &TenantContext{
        TenantID: tenant.ID,
        UserID:   user.ID,
        Roles:    user.Roles,
    })
    
    // User should be able to access their tenant
    retrievedTenant, err := suite.providers.storage.GetTenant(userCtx, tenant.ID)
    suite.Require().NoError(err)
    suite.Equal(tenant.ID, retrievedTenant.ID)
    
    // Create another tenant to test isolation
    otherTenant, err := suite.planes.control.CreateTenant(ctx, CreateTenantRequest{
        Name:  "Other Corp",
        Email: "admin@othercorp.com",
        Tier:  "basic",
    })
    suite.Require().NoError(err)
    
    // User should NOT be able to access other tenant
    _, err = suite.providers.storage.GetTenant(userCtx, otherTenant.ID)
    suite.Error(err)
    suite.Contains(err.Error(), "cross-tenant access")
    
    // Test tier-based resource allocation
    // Premium tenant should have more resources than basic tenant
    premiumResources, err := suite.planes.application.provisioner.ListTenantResources(ctx, tenant.ID)
    suite.Require().NoError(err)
    
    basicResources, err := suite.planes.application.provisioner.ListTenantResources(ctx, otherTenant.ID)
    suite.Require().NoError(err)
    
    // Premium should have more CPU/memory allocation
    premiumCPU := getResourceValue(premiumResources, "cpu")
    basicCPU := getResourceValue(basicResources, "cpu")
    suite.Greater(premiumCPU, basicCPU, "Premium tier should have more CPU than basic tier")
    
    // Test offboarding
    err = suite.planes.control.DeleteTenant(ctx, otherTenant.ID)
    suite.Require().NoError(err)
    
    // Wait for offboarding to complete
    for i := 0; i < 30; i++ {
        time.Sleep(1 * time.Second)
        
        _, err := suite.providers.storage.GetTenant(ctx, otherTenant.ID)
        if err != nil && strings.Contains(err.Error(), "not found") {
            break
        }
    }
    
    // Tenant should be deleted
    _, err = suite.providers.storage.GetTenant(ctx, otherTenant.ID)
    suite.Error(err)
    suite.Contains(err.Error(), "not found")
}
```

### Testing Configuration

The toolkit provides standardized testing configuration for property-based tests:

- **Minimum 100 iterations** per property test to ensure adequate coverage
- **Tenant isolation tests** run with concurrent operations to detect race conditions
- **Event delivery tests** verify ordering and delivery guarantees under load
- **Interface compliance tests** validate that all providers implement interfaces correctly
- **Performance benchmarks** ensure acceptable latency and throughput characteristics

Each correctness property is implemented as a property-based test with appropriate generators for test data and comprehensive assertions for expected behavior.