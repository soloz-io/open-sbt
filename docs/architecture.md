# Architecture

> Full documentation coming soon.

## Overview

open-sbt follows a Control Plane + Application Plane architecture inspired by AWS SBT-AWS, adapted for Kubernetes-native infrastructure.

- **Control Plane** — tenant lifecycle, auth, billing, event orchestration
- **Application Plane** — resource provisioning, GitOps, workload execution

Communication between planes is event-driven via NATS.

See the [design principles](../.kiro/steering/design-principles.md) for full architectural detail.
