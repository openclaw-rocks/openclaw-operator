# Roadmap

> Current version: **0.9.x** — tracking toward 1.0 stable release.

## Shipped

### v0.5.0–v0.6.0
- Custom resource definition (`OpenClawInstance`) with validating/defaulting webhooks
- Deployment management with full lifecycle (create, update, delete)
- Inline config (`spec.config.raw`) and external ConfigMap reference
- Persistent storage (PVC) with configurable size and storage class
- Chromium sidecar for browser automation
- Ingress with TLS, HSTS, and rate limiting
- NetworkPolicy with deny-all baseline
- RBAC (ServiceAccount, Role, RoleBinding per instance)
- PodDisruptionBudget
- ServiceMonitor for Prometheus
- Helm chart + OLM bundle for OperatorHub

### v0.7.0–v0.8.0
- Native workspace file seeding (`spec.workspace`)
- Deployment to StatefulSet migration (zero-downtime)
- PVC backup-on-delete and restore-from-backup
- Pod security hardening (non-root, drop ALL capabilities, seccomp)
- OperatorHub automated submission workflow

### v0.9.0–v0.9.4
- Opt-in auto-update from OCI registry with rollback on failure
- Config merge mode (`spec.config.mergeMode: merge`)
- Declarative skill installation (`spec.skills`)
- Secret rotation detection (envFrom watch + rollout trigger)
- Read-only root filesystem (PVC + /tmp emptyDir for writable paths)
- Auto-generated gateway token auth (bypasses Bonjour/mDNS pairing)
- `OPENCLAW_DISABLE_BONJOUR=1` set unconditionally

### v0.10.0 (Phase 2+3)
- ServiceAccount annotations for IRSA / GCP Workload Identity
- Custom CA bundle injection (`spec.security.caBundle`)
- Extra volumes and volume mounts (`spec.extraVolumes`, `spec.extraVolumeMounts`)
- `fsGroupChangePolicy: OnRootMismatch` for faster PVC startup
- Secret existence validation (SecretsReady condition)
- Provider-aware webhook warnings (model config vs API key env vars)
- Config schema validation (warn on unknown top-level keys)
- Custom init containers (`spec.initContainers`)
- JSON5 config support (`spec.config.format: json5`)
- Docs: model fallback chains, custom AI providers (Ollama/vLLM), External Secrets Operator
- Runtime dependency init containers — `spec.runtimeDeps.pnpm` and `spec.runtimeDeps.python` ([#89](https://github.com/openclaw-rocks/k8s-operator/issues/89))
- Ollama sidecar pattern (documented example + NetworkPolicy rules)

## Planned

### v1.0.0 (Stable)
- API graduation to `v1`
- Comprehensive conformance test suite
- Performance benchmarks and optimization
- Operator SDK scorecard compliance

### Future
- Multi-cluster federation
- GitOps-native configuration management
- Advanced scheduling (topology spread constraints)
- Certificate management integration (cert-manager)
- AI provider health monitoring
- Cost optimization recommendations
- Multi-tenancy support
