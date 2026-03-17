---
title: open-sbt toolkit to onboard tenants
description: The adapted ApplicationSet + Helm onboarding pattern for zero-ops platform using open-sbt
inclusion: always
---

# GitOps & Helm Tenant Onboarding (The Adapted Design)

**Version:** 1.0  
**Date:** 2026-03-17  
**Purpose:** Defines how the `zero-ops` platform utilizes the `open-sbt` toolkit to onboard tenants using an adapted, low-latency GitOps architecture.

## Executive Summary

To achieve an enterprise-grade, one-click SaaS factory without the operational burden of a massive platform team, the `zero-ops` platform adopts a highly optimized GitOps onboarding flow. By converging Red Hat/IBM GitOps patterns with `open-sbt`'s Control/Application Plane architecture, we eliminate custom Kubernetes operators, bypass traditional GitOps polling latency, and implement "Warm Pooling" to achieve sub-second onboarding for shared tiers.

## Pattern Analysis Overview

This design is the synthesis of our convergence analysis across six major architectural patterns:

1. **IBM GitOps Pattern (openshift-clusterconfig-gitops):** Adopted the concept of "T-Shirt Sizing" (Basic, Standard, Premium, Enterprise) injected via global Helm values.
2. **Red Hat GitOps Onboarding (Blog Pattern):** Adopted the core mechanism: an ArgoCD `ApplicationSet` using a Git Directory Generator to stamp out tenant Helm charts dynamically.
3. **Red Hat COP Projects (namespace-configuration-operator, etc.):** *Explicitly Rejected* the use of custom Go-based K8s operators for tenant K8s resource generation to reduce operational complexity. Replaced entirely by ArgoCD + Helm.
4. **open-sbt Design Principles:** Leveraged the strict separation of Control Plane (API, Auth) and Application Plane (GitOps, NATS events, Provisioning).
5. **SaaS Architecture Principles (AWS SaaS Lens):** Enforced identity-tenant binding (Ory), defense-in-depth isolation (PostgreSQL RLS, Namespaces), and tier-based resource allocation.
6. **Kubernetes SaaS Patterns:** Mapped user needs to K8s primitives (NetworkPolicies, ResourceQuotas, Crossplane Compositions).

---

## The Adapted Design: Architecture

The `open-sbt` toolkit provides the `IProvisioner` and `IEventBus` interfaces. The `zero-ops` platform implements these to execute the following architecture:

### 1. The GitOps Repository Structure
All tenant infrastructure state lives in a single, centralized Git repository managed by the Application Plane.

```text
gitops-repo/
├── base-charts/
│   └── tenant-factory/         # Universal Helm chart for all tenants
├── tenants/
│   ├── warm-pool-01/           # Pre-provisioned unassigned tenant
│   │   └── values.yaml
│   ├── warm-pool-02/           # Pre-provisioned unassigned tenant
│   │   └── values.yaml
│   ├── acme-corp-123/          # Active assigned tenant
│   │   └── values.yaml
│   └── stark-ind-456/          # Enterprise BYOC tenant
│       └── values.yaml
└── applicationset.yaml         # The ArgoCD AppSet generator
```

### 2. The ArgoCD ApplicationSet
A single ArgoCD `ApplicationSet` watches the `tenants/` directory. Whenever the `open-sbt` Application Plane commits a new folder, ArgoCD automatically generates an `Application` custom resource.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: zero-ops-tenant-factory
spec:
  generators:
  - git:
      repoURL: https://github.com/zero-ops/gitops-repo
      revision: HEAD
      directories:
      - path: tenants/*
  template:
    metadata:
      name: 'tenant-{{path.basename}}'
    spec:
      project: tenants
      source:
        repoURL: https://github.com/zero-ops/gitops-repo
        targetRevision: HEAD
        path: base-charts/tenant-factory
        helm:
          valueFiles:
          - ../../tenants/{{path.basename}}/values.yaml
      destination:
        server: https://kubernetes.default.svc
        namespace: 'tenant-{{path.basename}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
```

### 3. The Universal Tenant Helm Chart
Instead of creating Go-based K8s operators, a single Helm chart handles all resource manifestation based on the T-Shirt size defined in `values.yaml`.

* **Kubernetes Primitives:** Generates `Namespace`, `ResourceQuota`, `NetworkPolicy`, and `RoleBindings`.
* **Crossplane XRs:** Generates Crossplane `CompositeResources` (e.g., `TenantDatabase`, `TenantCluster`) which Crossplane then resolves to provision CNPG Postgres databases or Hetzner CAPI clusters.

---

## Mitigating the Major Concerns

To successfully run this as a one-person AI-Native company, we implement three critical mitigations:

### Mitigation 1: Eliminating Operational Complexity
* **No Custom K8s Operators:** The entire K8s footprint is declarative via Helm and Crossplane.
* **GitOps DR:** Disaster recovery requires only bootstrapping ArgoCD and pointing it to the Git repository. Crossplane and ArgoCD will reconstruct the entire SaaS platform automatically.
* **AI-Operated Platform:** The platform is managed via the `zero-ops-api` MCP server. The Platform Admin's AI agent uses tools like `sync_application` to trigger GitOps syncs and `get_cluster_diagnostics` to debug.

### Mitigation 2: Performance at Scale
* **PostgreSQL RLS Optimization:** RLS is restricted to Shared Pool tiers. The `open-sbt` data layer uses `sqlc` to enforce strict `(tenant_id)` composite indexing on all queries, guaranteeing millisecond query times even with billions of rows.
* **Webhook-Driven Syncs (No Polling):** ArgoCD's 3-minute Git polling is disabled. The `open-sbt` Application Plane fires a direct Webhook to the ArgoCD API immediately after a Git commit, triggering instant reconciliation and eliminating API server thrashing.

### Mitigation 3: Solving Onboarding Latency (The Warm Pool Pattern)
Standard GitOps provisioning takes 45-80 seconds. `open-sbt` splits the onboarding flow into two latency-optimized paths:

#### Path A: Shared Pool Tiers (Basic / Standard) - Latency < 2 Seconds
1. The `open-sbt` App Plane maintains a baseline of 10 "warm" (pre-provisioned) namespaces and Postgres schemas in the cluster.
2. **API Call:** User requests a Basic tier SaaS instance.
3. **Instant Claim:** The Control Plane DB instantly marks `warm-pool-01` as belonging to `tenant-123`.
4. **Auth Binding:** Ory Keto instantly creates the relationship `tenant:123#admin@user:xyz`.
5. **Response:** API returns `200 OK` in < 2 seconds. The user can use the platform immediately.
6. **Async Refill:** The Control Plane publishes a NATS event. The App Plane updates `warm-pool-01`'s `values.yaml` in Git to reflect its new owner, and commits a new `warm-pool-11` folder to Git to replace the consumed warm slot.

#### Path B: Dedicated / BYOC Tiers (Premium / Enterprise) - Latency 2-5 Minutes
1. **API Call:** User requests an Enterprise tier (Dedicated Hetzner CAPI Cluster or Dedicated CNPG Database).
2. **Response:** API returns `202 Accepted` instantly.
3. **AI UX Masking:** The frontend AI agent engages the user in a configuration conversation while the provisioning happens.
4. **GitOps Execution:** The App Plane commits a new `tenants/<tenant-id>` folder to Git with `tier: enterprise`.
5. **Crossplane Provisioning:** ArgoCD syncs the Helm chart, which creates a Crossplane `TenantCluster` XR. Crossplane provisions the Hetzner nodes.
6. **Event Streaming:** Progress is streamed back to the frontend Agent via NATS WebSockets until the cluster is ready.

---

## Alignment with SaaS Architecture Principles

1. **Identity Binding:** The onboarding flow intrinsically links the user to the tenant via Ory Kratos (Identity) and Ory Keto (Relationships). The resulting JWT contains the `tenant_id` and `tenant_tier`.
2. **Defense-in-Depth Isolation:**
   * *Data Layer:* PostgreSQL RLS enforced by `open-sbt` middleware.
   * *Compute Layer:* Kubernetes Namespaces + NetworkPolicies (generated via Helm).
   * *Execution Layer:* gVisor (`runsc`) sandboxing for AI Agent execution workflows.
3. **Tier-Based Provisioning:** The entire infrastructure footprint is controlled by a single `tier` property in the tenant's `values.yaml`, feeding into the Crossplane Compositions and K8s Quotas.

## Next Steps for Implementation
With this design approved, the implementation phase begins with:
1. Initializing the Go module for `open-sbt`.
2. Scaffolding the core `pkg/interfaces` (`IProvisioner`, `IAuth`, `IEventBus`).
3. Setting up the base GitOps `tenant-factory` Helm chart structure.

