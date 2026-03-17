# Requirements Document

## Introduction

open-sbt is a Go-based SaaS builder toolkit that provides reusable abstractions for building multi-tenant SaaS applications. The toolkit enables SaaS builders to create production-grade backend infrastructure without vendor lock-in while following proven SaaS architecture patterns. It abstracts the complexity of multi-tenant systems through well-defined interfaces and provides standardized workflows for tenant lifecycle management.

## Glossary

- **Control_Plane**: The component responsible for tenant management, user authentication, and business logic coordination
- **Application_Plane**: The component responsible for resource provisioning, infrastructure management, and responding to Control Plane events
- **Tenant**: An isolated customer environment within the SaaS application with dedicated resources and data
- **Tier**: A classification system for tenants that determines resource allocation and feature availability (starter, enterprise, etc.)
- **Provider**: A concrete implementation of an interface (e.g., PostgreSQL implementation of IStorage)
- **Event_Bus**: The messaging system that enables asynchronous communication between Control Plane and Application Plane
- **Provisioner**: The component responsible for creating and managing tenant infrastructure resources
- **GitOps**: A deployment methodology where infrastructure changes are managed through Git commits and automated reconciliation

## Requirements

### Requirement 1: Provider-Agnostic Interface Abstraction

**User Story:** As a SaaS builder, I want to use standardized interfaces for core components, so that I can swap implementations without breaking my application code.

#### Acceptance Criteria

1. THE Toolkit SHALL provide eight core interfaces: IAuth, IEventBus, IProvisioner, IStorage, IBilling, IMetering, ITierManager, and ISecretManager
2. WHEN a provider implementation is swapped, THE Toolkit SHALL continue to function without code changes in user applications
3. THE Interface_Definitions SHALL be technology-agnostic and not prescribe specific implementations
4. WHEN multiple providers implement the same interface, THE Toolkit SHALL allow runtime selection of providers
5. THE Interfaces SHALL include comprehensive method signatures for all core SaaS operations
6. THE ISecretManager interface SHALL provide secure secret management capabilities for GitOps workflows
7. THE ITierManager interface SHALL provide tier configuration and quota management capabilities

### Requirement 2: Control Plane and Application Plane Separation

**User Story:** As a platform architect, I want clear separation between business logic and infrastructure management, so that I can scale and maintain each component independently.

#### Acceptance Criteria

1. THE Control_Plane SHALL handle tenant management, authentication, and business logic without direct infrastructure access
2. THE Application_Plane SHALL handle resource provisioning and infrastructure management without direct business logic
3. WHEN Control_Plane needs infrastructure changes, THE Control_Plane SHALL communicate through the Event_Bus only
4. THE Application_Plane SHALL respond to Control_Plane events and publish status updates through the Event_Bus
5. THE Separation SHALL prevent direct API calls between planes

### Requirement 3: Event-Driven Communication

**User Story:** As a system integrator, I want asynchronous communication between components, so that I can build resilient and scalable SaaS applications.

#### Acceptance Criteria

1. THE Event_Bus SHALL support publishing and subscribing to tenant lifecycle events
2. WHEN the Control_Plane publishes an onboarding event, THE Application_Plane SHALL receive and process it
3. THE Event_Bus SHALL provide standard event definitions for tenant operations (create, update, delete, activate, deactivate)
4. WHEN events are published, THE Event_Bus SHALL guarantee delivery to all subscribers
5. THE Event_Bus SHALL support event ordering for tenant operations
6. THE Event_Bus SHALL implement idempotency protection using the Inbox Pattern to prevent duplicate event processing
7. THE Event_Bus SHALL handle at-least-once delivery guarantees without causing duplicate operations
8. THE Event_Bus SHALL provide event deduplication based on unique event IDs
9. THE Storage_Interface SHALL provide RecordProcessedEvent and IsEventProcessed methods for implementing the Inbox Pattern
10. WHEN duplicate events are received, THE Application_Plane SHALL detect them using the processed_events table and skip processing
11. THE Event_Bus SHALL support Event-Driven State Machine events including GitCommitted, ArgoSyncStarted, ArgoSyncCompleted, and ArgoHealthChanged

### Requirement 4: Tenant Lifecycle Management

**User Story:** As a SaaS operator, I want automated tenant onboarding and offboarding workflows, so that I can scale my business without manual intervention.

#### Acceptance Criteria

1. WHEN a tenant registration is created, THE Control_Plane SHALL publish an onboarding request event
2. THE Application_Plane SHALL provision tenant resources and publish success or failure events
3. WHEN tenant offboarding is requested, THE Application_Plane SHALL deprovision resources and clean up data
4. THE Toolkit SHALL support tenant state transitions following the Event-Driven State Machine: CREATING → GIT_COMMITTED → SYNCING → READY/FAILED
5. WHEN tier upgrades occur, THE Application_Plane SHALL adjust resources without downtime
6. FOR basic and standard tier tenants, THE Provisioner SHALL support warm pool onboarding with sub-2-second response times
7. THE Provisioner SHALL maintain pre-provisioned warm slots and automatically refill the pool when slots are claimed
8. THE Provisioner SHALL provide warm pool status monitoring and capacity management capabilities
9. THE Control_Plane SHALL implement active reconciliation to detect and resolve orphaned infrastructure
10. WHEN tenants are stuck in transitional states, THE Control_Plane SHALL automatically verify status and transition to correct final states
11. THE Control_Plane SHALL support failed states for definitive provisioning failures with manual retry capabilities
12. THE Toolkit SHALL implement an Event-Driven State Machine where PostgreSQL serves as business truth, Git as desired state, and ArgoCD as executor
13. THE Storage_Interface SHALL support webhook-driven state management with ArgoCD status updates pushed to PostgreSQL
14. THE Tenant model SHALL include ArgoSyncStatus, ArgoHealthStatus, and LastObservedAt fields for real-time infrastructure observability

### Requirement 5: Multi-Tenant Security and Isolation

**User Story:** As a security engineer, I want strong tenant isolation guarantees, so that customer data remains secure and separated.

#### Acceptance Criteria

1. THE Storage_Interface SHALL enforce tenant context in all data operations
2. WHEN tenant A requests data, THE Storage_Interface SHALL never return tenant B's data
3. THE Auth_Interface SHALL bind user identity to tenant context in authentication tokens
4. THE Provisioner SHALL create isolated resources for each tenant based on their tier
5. IF cross-tenant access is attempted, THEN THE Toolkit SHALL log the attempt and deny access
6. THE SecretManager SHALL store sensitive data securely and never expose secrets in plain text in Git repositories
7. THE SecretManager SHALL integrate with HashiCorp Vault for enterprise-grade secret management
8. THE SecretManager SHALL provide tenant-scoped secret isolation and access controls

### Requirement 6: GitOps-First Infrastructure Management

**User Story:** As a DevOps engineer, I want all infrastructure changes to be managed through Git, so that I have audit trails and rollback capabilities.

#### Acceptance Criteria

1. THE Provisioner SHALL commit tenant configurations to Git repositories before applying changes
2. WHEN infrastructure changes are needed, THE Provisioner SHALL create Git commits rather than direct API calls
3. THE Provisioner SHALL support rollback through Git revert operations
4. THE GitOps_Workflow SHALL provide audit trails for all tenant infrastructure changes
5. THE Provisioner SHALL integrate with GitOps tools for automated reconciliation
6. THE Provisioner SHALL support webhook-triggered synchronization to eliminate polling delays
7. WHEN Git commits are made, THE Provisioner SHALL trigger immediate ArgoCD sync via webhooks
8. THE Provisioner SHALL provide sync trigger mechanisms for manual and automated reconciliation

### Requirement 7: Tier-Based Resource Management

**User Story:** As a product manager, I want different resource allocations for different customer tiers, so that I can offer tiered pricing and service levels.

#### Acceptance Criteria

1. THE Provisioner SHALL allocate resources based on tenant tier configuration
2. WHEN a starter tier tenant is created, THE Provisioner SHALL apply basic resource limits
3. WHEN an enterprise tier tenant is created, THE Provisioner SHALL apply premium resource allocations
4. THE Tier_Configuration SHALL be customizable through provider implementations
5. WHEN tier upgrades occur, THE Provisioner SHALL adjust resources without downtime
6. THE ITierManager SHALL provide formal tier definitions with quotas, features, and pricing configuration
7. THE ITierManager SHALL validate tenant resource usage against tier quotas before provisioning
8. THE ITierManager SHALL support tier feature management and quota enforcement

### Requirement 8: Comprehensive Testing Framework

**User Story:** As a toolkit maintainer, I want comprehensive testing patterns, so that I can ensure reliability across different provider implementations.

#### Acceptance Criteria

1. THE Toolkit SHALL provide E2E testing patterns using Testcontainers-Go
2. THE Test_Framework SHALL verify tenant isolation across all provider implementations
3. THE Test_Framework SHALL include property-based tests for core SaaS invariants
4. WHEN new providers are implemented, THE Test_Framework SHALL validate interface compliance
5. THE Test_Framework SHALL test multi-tenant scenarios with concurrent operations

### Requirement 9: Database Access Abstraction

**User Story:** As a backend developer, I want database-agnostic data access patterns, so that I can choose the best database for my use case.

#### Acceptance Criteria

1. THE Storage_Interface SHALL provide CRUD operations for tenants, users, and configurations
2. THE Storage_Interface SHALL support tenant-scoped queries without exposing database specifics
3. WHEN different database providers are used, THE Storage_Interface SHALL maintain consistent behavior
4. THE Storage_Interface SHALL support transaction management for complex operations
5. THE Storage_Interface SHALL provide tenant data isolation guarantees
6. THE Storage_Interface SHALL provide ListStuckTenants and ListUnobservedTenants methods for active reconciliation
7. THE Storage_Interface SHALL support UpdateTenantArgoStatus and TouchTenantObservation methods for webhook-driven state management
8. THE Storage_Interface SHALL serve as the business truth source in the Event-Driven State Machine architecture

### Requirement 10: Billing and Metering Integration

**User Story:** As a business owner, I want usage tracking and billing integration, so that I can monetize my SaaS application effectively.

#### Acceptance Criteria

1. THE Metering_Interface SHALL track tenant resource usage and custom metrics
2. THE Billing_Interface SHALL integrate with external billing systems for invoice generation
3. WHEN usage events occur, THE Metering_Interface SHALL record them with tenant context
4. THE Billing_Interface SHALL support webhook handling for billing system notifications
5. THE Metering_Interface SHALL provide usage aggregation and reporting capabilities

### Requirement 11: Observability and Monitoring

**User Story:** As a site reliability engineer, I want tenant-aware monitoring and logging, so that I can troubleshoot issues and monitor system health.

#### Acceptance Criteria

1. THE Toolkit SHALL provide tenant-aware logging helpers that automatically include tenant context
2. THE Toolkit SHALL provide metrics helpers that automatically tag metrics with tenant information
3. WHEN errors occur, THE Toolkit SHALL log them with sufficient context for debugging
4. THE Observability_Helpers SHALL work with popular monitoring systems (Prometheus, Grafana, etc.)
5. THE Toolkit SHALL provide health check endpoints for monitoring system status

### Requirement 12: Configuration Management

**User Story:** As a system administrator, I want centralized configuration management for tenant settings, so that I can customize behavior per tenant.

#### Acceptance Criteria

1. THE Storage_Interface SHALL support tenant-specific configuration storage and retrieval
2. THE Configuration_System SHALL allow global defaults with tenant-specific overrides
3. WHEN configuration changes occur, THE Configuration_System SHALL notify relevant components
4. THE Configuration_System SHALL validate configuration values before applying them
5. THE Configuration_System SHALL support configuration versioning and rollback

### Requirement 13: Secure Secret Management

**User Story:** As a security engineer, I want secure secret management for GitOps workflows, so that sensitive data is never exposed in plain text in Git repositories.

#### Acceptance Criteria

1. THE SecretManager SHALL integrate with HashiCorp Vault for secure secret storage
2. THE SecretManager SHALL provide tenant-scoped secret isolation and access controls
3. WHEN secrets are needed in GitOps workflows, THE SecretManager SHALL store vault references instead of plain text values
4. THE SecretManager SHALL support secret encryption and decryption for Git operations
5. THE SecretManager SHALL provide secret rotation capabilities and audit trails
6. THE SecretManager SHALL support both global and tenant-specific secret management
7. THE SecretManager SHALL integrate with GitOps provisioning workflows for secure credential handling

### Requirement 14: Active Reconciliation and Orphaned Infrastructure Recovery

**User Story:** As a platform operator, I want automatic detection and recovery of orphaned infrastructure, so that tenants don't get stuck in transitional states when the Application Plane fails to publish status events.

#### Acceptance Criteria

1. THE Control_Plane SHALL implement a ControlPlaneReconciler with configurable reconciliation intervals
2. THE Reconciler SHALL detect tenants stuck in transitional states (provisioning, deprovisioning) for longer than the configured threshold
3. WHEN stuck tenants are detected, THE Reconciler SHALL query actual infrastructure status via IProvisioner.GetProvisioningStatus
4. THE Reconciler SHALL automatically transition tenants to correct final states based on actual infrastructure status
5. WHEN infrastructure is healthy but tenant is stuck in provisioning, THE Reconciler SHALL transition tenant to active and publish synthetic success events
6. WHEN infrastructure is degraded or failed, THE Reconciler SHALL transition tenant to failed state and publish synthetic failure events
7. THE Reconciler SHALL support configurable retry limits and exponential backoff for failed reconciliation attempts
8. THE Storage_Interface SHALL provide methods to identify tenants that haven't received ArgoCD webhook updates within expected timeframes

### Requirement 15: Event-Driven State Machine Architecture

**User Story:** As a platform architect, I want a clean event-driven state machine where PostgreSQL serves as business truth, Git as desired state, and ArgoCD as executor, so that I can achieve sub-millisecond dashboard performance and eliminate polling overhead.

#### Acceptance Criteria

1. THE Toolkit SHALL implement an Event-Driven State Machine with clean state transitions: CREATING → GIT_COMMITTED → SYNCING → READY/FAILED
2. THE PostgreSQL database SHALL serve as the single source of business truth for tenant states
3. THE Git repository SHALL serve as the desired state for tenant infrastructure configurations
4. THE ArgoCD SHALL serve as the executor that reconciles desired state with actual infrastructure
5. THE Tenant model SHALL include ArgoSyncStatus, ArgoHealthStatus, and LastObservedAt fields updated by ArgoCD webhooks
6. THE Dashboard queries SHALL use PostgreSQL as the data source instead of Kubernetes API for sub-millisecond performance
7. THE Webhook-driven architecture SHALL eliminate polling overhead by pushing ArgoCD status updates directly to PostgreSQL
8. THE Event_Bus SHALL support Event-Driven State Machine events: GitCommitted, ArgoSyncStarted, ArgoSyncCompleted, ArgoHealthChanged
9. THE Storage_Interface SHALL provide UpdateTenantArgoStatus and TouchTenantObservation methods for webhook-driven state updates
10. THE State machine SHALL support failed and failed_cleanup states for definitive provisioning failures with manual retry capabilities