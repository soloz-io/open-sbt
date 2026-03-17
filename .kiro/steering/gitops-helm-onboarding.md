---
title: GitOps & Helm Tenant Onboarding
description: The adapted ApplicationSet + Helm onboarding pattern for zero-ops platform using open-sbt
inclusion: always
---

# GitOps Helm Onboarding: Pattern Convergence Analysis

**Version:** 1.0  
**Date:** 2026-03-17  
**Author:** Kiro Analysis  
**Purpose:** Comprehensive analysis of GitOps patterns for SaaS tenant onboarding

## Executive Summary

This document analyzes the convergence of multiple GitOps and SaaS patterns to create a unified architecture for automated tenant onboarding. The analysis covers IBM's GitOps patterns, Red Hat's onboarding approaches, Red Hat Community of Practice (COP) projects, open-sbt design principles, and AWS SaaS architecture patterns.

**Key Finding:** The patterns converge into a three-layer architecture combining SaaS business logic, GitOps orchestration, and Kubernetes resource management.

## Pattern Analysis Overview

### Analyzed Patterns

1. **IBM GitOps Pattern** (openshift-clusterconfig-gitops)
2. **Red Hat GitOps Onboarding** (from Red Hat blog article)
3. **Red Hat COP Projects** (namespace-configuration-operator, gitops-generator, gitops-catalog)
4. **open-sbt Design Principles** (Control Plane + Application Plane architecture)
5. **SaaS Architecture Principles** (AWS Well-Architected SaaS Lens adaptation)
6. **Kubernetes SaaS Patterns** (User need mapping to technical implementations)

### Convergence Architecture

The patterns converge into a unified three-layer architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    Layer 1: SaaS Business Logic             │
│  (open-sbt Control Plane + Application Plane)               │
│  - Tenant lifecycle management                               │
│  - Event-driven communication (NATS)                        │
│  - MCP tools for agent integration                          │
│  - Billing, metering, observability                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                Layer 2: GitOps Orchestration                │
│  (Red Hat + IBM patterns)                                   │
│  - ApplicationSet-driven tenant discovery                   │
│  - Helm templating with T-Shirt sizing                      │
│  - Git as source of truth                                   │
│  - ArgoCD for continuous deployment                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Layer 3: Kubernetes Resource Management        │
│  (Red Hat COP projects)                                     │
│  - namespace-configuration-operator                         │
│  - gitops-generator for resource templating                 │
│  - gitops-catalog for component library                     │
│  - Automatic drift prevention and reconciliation            │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Pattern Analysis

### 1. IBM GitOps Pattern (openshift-clusterconfig-gitops)

**Architecture Overview:**
IBM's approach uses a dual-folder structure with T-Shirt sizing for standardized configurations.

**Key Components Analyzed:**
- **T-Shirt Sizing**: Global values with S/M/L/XL configurations defined in `values-global.yaml`
- **Helm Templating**: Values-based configuration management using helper-proj-onboarding chart
- **Folder Structure**: 
  - `tenant-onboarding/`: Team onboarding configurations
  - `tenants/`: Individual tenant application configurations
  - `clusters/`: Cluster-wide base configurations

**Observed Implementation:**
```yaml
# From tenant-onboarding/values-global.yaml
global:
  application_gitops_namespace: gitops-application
  envs:
    - name: in-cluster
      url: https://kubernetes.default.svc
  tshirt_sizes:
    - name: XL
      quota:
        pods: 100
        limits:
          cpu: 4
          memory: 4Gi
    - name: S
      quota:
        limits:
          cpu: 1
          memory: 1Gi
```

**Integration Analysis:**
The IBM T-Shirt sizing approach uses global values with predefined configurations that map to different resource allocations and limits.

### 2. Red Hat GitOps Onboarding Pattern

**Architecture Overview:**
Red Hat's approach emphasizes ApplicationSet-driven discovery with Helm templating for tenant-specific configurations, as described in their blog article "Project onboarding using GitOps and Helm".

**Key Components Analyzed:**
- **ApplicationSet Controller**: Uses Git Generator to walk folder structure and create Applications
- **Helm Charts**: Project-onboarding chart with helper-proj-onboarding dependency
- **Git Repository Structure**: `tenant-projects/{tenant}/{cluster}/values.yaml` pattern
- **T-Shirt Sizing**: Global values-file with predefined S/M/L/XL configurations

**Observed Tenant Onboarding Flow:**
```
1. Create folder: tenant-projects/{tenant-name}/{cluster-name}/
2. Add values.yaml with tenant configuration
3. ApplicationSet Git Generator detects new folder
4. ApplicationSet creates ArgoCD Application
5. ArgoCD deploys using Helm chart with values.yaml
6. Tenant resources provisioned automatically
```

**Integration Analysis:**
The Red Hat ApplicationSet approach uses Git Generators to automatically detect folder structures and create ArgoCD Applications for tenant provisioning.

**ApplicationSet Template (from Red Hat article):**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: tenant-onboarding
spec:
  generators:
  - git:
      repoURL: https://github.com/org/tenant-configs
      revision: HEAD
      directories:
      - path: tenants/*
  template:
    metadata:
      name: '{{path.basename}}'
    spec:
      project: tenants
      source:
        repoURL: https://github.com/org/tenant-configs
        targetRevision: HEAD
        path: '{{path}}'
        helm:
          valueFiles:
          - values.yaml
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{path.basename}}'
```

### 3. Red Hat COP Projects Analysis

#### 3.1 namespace-configuration-operator

**Purpose:** Event-driven namespace provisioning with Go templating

**Analyzed Features:**
- **Event-Driven**: Watches for Group, User, and Namespace creation events
- **Go Templates**: Uses text/template for flexible resource generation
- **CRDs**: GroupConfig, UserConfig, NamespaceConfig for configuration
- **Drift Prevention**: Continuously reconciles desired state
- **Team Onboarding Example**: Creates 4 namespaces per team (build, dev, qa, prod)

**Observed Team Onboarding Implementation:**
```yaml
# From examples/team-onboarding/group-config.yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupConfig
metadata:
  name: team-onboarding
spec:
  labelSelector:
    matchLabels:
      type: devteam    
  templates:
    - objectTemplate: |
        apiVersion: v1
        kind: Namespace
        metadata:
          name: {{ .Name }}-build
        labels:
          team: {{ .Name }}
          type: build
```

**Integration Analysis:**
The namespace-configuration-operator provides event-driven namespace provisioning that reacts to Kubernetes resource creation events.

#### 3.2 gitops-generator

**Purpose:** Resource generation using GeneratorOptions pattern

**Analyzed Features:**
- **GeneratorOptions API**: Structured configuration for resource generation
- **Multiple Resource Types**: Deployments, Services, Routes, Ingresses
- **Template-based**: Uses Go templates for resource generation
- **Kustomize Integration**: Generates Kustomize-compatible structures

**Observed API Structure:**
```go
// From api/v1alpha1/generator_options.go
type GeneratorOptions struct {
    Name        string `json:"name"`
    Namespace   string `json:"namespace,omitempty"`
    Application string `json:"application"`
    Replicas    int    `json:"replicas,omitempty"`
    TargetPort  int    `json:"targetPort,omitempty"`
    ContainerImage string `json:"containerImage,omitempty"`
    // ... additional fields
}
```

**Integration Analysis:**
The gitops-generator provides structured resource generation through its GeneratorOptions API and template-based approach.

#### 3.3 gitops-catalog

**Purpose:** Reusable GitOps patterns and components library

**Key Features:**
- **Component Library**: Pre-built GitOps patterns
- **Best Practices**: Proven configurations
- **Composability**: Mix and match components

**Integration Analysis:**
The gitops-catalog provides a library of reusable GitOps patterns and pre-built configurations for common use cases.

## Pattern Convergence Analysis

### Three-Layer Architecture Deep Dive

#### Layer 1: SaaS Business Logic (open-sbt)

**Responsibilities:**
- Tenant lifecycle management (CRUD operations)
- Event-driven communication between Control Plane and Application Plane
- Authentication and authorization (Ory Stack integration)
- Billing and metering abstractions
- MCP tools for agent integration
- Multi-tenant observability

**Observed open-sbt Interface Alignment:**
The analyzed patterns show compatibility with open-sbt's interface-based architecture through the IAuth, IEventBus, and IProvisioner interfaces.

**Observed GitOps Integration Points:**
The analyzed patterns show integration opportunities where Control Plane operations could trigger GitOps workflows, while Application Plane components could respond to Git-driven deployment events.

#### Layer 2: GitOps Orchestration (Red Hat + IBM)

**Responsibilities:**
- Git-based configuration management
- ApplicationSet-driven tenant discovery
- Helm templating and T-Shirt sizing
- ArgoCD continuous deployment
- Configuration drift prevention

**Key Components:**
```yaml
# ApplicationSet for tenant discovery
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: tenant-factory
spec:
  generators:
  - git:
      repoURL: https://github.com/org/tenant-configs
      revision: HEAD
      directories:
      - path: tenants/*
  template:
    metadata:
      name: 'tenant-{{path.basename}}'
    spec:
      source:
        helm:
          valueFiles:
          - values-{{path.basename}}.yaml
          parameters:
          - name: tenantId
            value: '{{path.basename}}'
          - name: tier
            value: '{{path.basename}}'
```

**Observed GitOps Orchestration Components:**
The Red Hat and IBM patterns demonstrate GitOps orchestration through ApplicationSet controllers, Helm templating, and ArgoCD continuous deployment for configuration management and deployment automation.

#### Layer 3: Kubernetes Resource Management (Red Hat COP)

**Responsibilities:**
- Automatic namespace configuration
- Resource template application
- RBAC and security policy enforcement
- Monitoring and logging setup
- Resource quota management

**Key Operators:**
```go
// namespace-configuration-operator watching for tenant namespaces
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    namespace := &corev1.Namespace{}
    if err := r.Get(ctx, req.NamespacedName, namespace); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Check if this is a tenant namespace
    if tenantID, exists := namespace.Labels["tenant-id"]; exists {
        tier := namespace.Labels["tier"]
        
        // Apply tier-based configuration
        return r.applyTenantConfiguration(ctx, namespace, tenantID, tier)
    }
    
    return ctrl.Result{}, nil
}
```

**Observed Kubernetes Resource Management:**
The Red Hat COP projects demonstrate Kubernetes resource management through operators that automatically configure namespaces, apply security policies, and manage resource quotas based on labels and events.

### Analysis Summary

The convergence analysis reveals that the IBM GitOps patterns, Red Hat onboarding approaches, Red Hat COP projects, and open-sbt design principles show compatibility across three architectural layers for SaaS tenant management.

**Key Findings:**

#### 1. Pattern Compatibility
- **Business Logic Layer**: open-sbt provides SaaS-specific abstractions
- **Orchestration Layer**: GitOps handles deployment and configuration management  
- **Resource Management Layer**: Kubernetes operators handle low-level resource lifecycle

#### 2. Technology Consistency
- **Go-based**: All analyzed projects use Go for implementation consistency
- **Event-driven**: NATS events, Git events, and Kubernetes events provide coordination
- **Declarative**: Configuration defined as code in Git repositories

#### 3. Operational Characteristics
- **GitOps**: All infrastructure changes tracked in Git with audit trails
- **Automation**: Minimal manual intervention through operator-driven reconciliation
- **Scalability**: Each layer operates independently with clear interfaces

#### 4. Implementation Flexibility
- **Pluggable**: Each layer can use different implementations
- **Extensible**: New patterns can be added at appropriate layers
- **Cloud-agnostic**: Works across different Kubernetes distributions

## Analysis Conclusion

The analysis of IBM GitOps patterns, Red Hat onboarding approaches, Red Hat COP projects, and open-sbt design principles reveals compatibility across different aspects of SaaS tenant management:

- **Proven patterns** from IBM and Red Hat implementations
- **Operational automation** through GitOps workflows and Kubernetes operators
- **Interface abstractions** through open-sbt's design
- **Event coordination** through event-driven communication
- **Component flexibility** through pluggable architectures

The analyzed projects demonstrate approaches that operate with defined responsibilities and interfaces across different architectural layers.