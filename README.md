<p align="center">
  <img src="docs/images/banner.svg" alt="OpenClaw Kubernetes Operator — OpenClaws sailing the Kubernetes seas" width="100%">
</p>

# OpenClaw Kubernetes Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/OpenClaw-rocks/k8s-operator)](https://goreportcard.com/report/github.com/OpenClaw-rocks/k8s-operator)
[![CI](https://github.com/OpenClaw-rocks/k8s-operator/actions/workflows/ci.yaml/badge.svg)](https://github.com/OpenClaw-rocks/k8s-operator/actions/workflows/ci.yaml)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28%2B-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev)

**Self-host [OpenClaw](https://openclaw.ai) AI agents on Kubernetes with production-grade security, observability, and lifecycle management.**

OpenClaw is an AI agent platform that acts on your behalf across Telegram, Discord, WhatsApp, and Signal. It manages your inbox, calendar, smart home, and more through 50+ integrations. While [OpenClaw.rocks](https://openclaw.rocks) offers fully managed hosting, this operator lets you run OpenClaw on your own infrastructure with the same operational rigor.

---

## Why an Operator?

Deploying AI agents to Kubernetes involves more than a Deployment and a Service. You need network isolation, secret management, persistent storage, health monitoring, optional browser automation, and config rollouts, all wired correctly. This operator encodes those concerns into a single `OpenClawInstance` custom resource so you can go from zero to production in minutes:

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-agent
spec:
  envFrom:
    - secretRef:
        name: openclaw-api-keys
  storage:
    persistence:
      enabled: true
      size: 10Gi
```

The operator reconciles this into a fully managed stack of 9+ Kubernetes resources: secured, monitored, and self-healing.

## Features

| | Feature | Details |
|---|---|---|
| **Declarative** | Single CRD | One resource defines the entire stack: StatefulSet, Service, RBAC, NetworkPolicy, PVC, PDB, Ingress, and more |
| **Secure** | Hardened by default | Non-root (UID 1000), read-only root filesystem, all capabilities dropped, seccomp RuntimeDefault, default-deny NetworkPolicy, validating webhook |
| **Observable** | Built-in metrics | Prometheus metrics, ServiceMonitor integration, structured JSON logging, Kubernetes events |
| **Flexible** | Provider-agnostic config | Use any AI provider (Anthropic, OpenAI, or others) via environment variables and inline or external config |
| **Config Modes** | Merge or overwrite | `overwrite` replaces config on restart; `merge` deep-merges with PVC config, preserving runtime changes |
| **Skills** | Declarative install | Install ClawHub skills via `spec.skills` — the operator runs an init container to fetch them before the agent starts |
| **Runtime Deps** | pnpm & Python/uv | Built-in init containers install pnpm (via corepack) or Python 3.12 + uv for MCP servers and skills |
| **Auto-Update** | OCI registry polling | Opt-in version tracking: checks the registry for new semver releases, backs up first, rolls out, and auto-rolls back if the new version fails health checks |
| **Resilient** | Self-healing lifecycle | PodDisruptionBudgets, health probes, automatic config rollouts via content hashing, 5-minute drift detection |
| **Backup/Restore** | B2-backed snapshots | Automatic backup to Backblaze B2 on instance deletion; restore into a new instance from any snapshot |
| **Workspace Seeding** | Initial files & dirs | Pre-populate the workspace with files and directories before the agent starts |
| **Gateway Auth** | Auto-generated tokens | Automatic gateway token Secret per instance, bypassing mDNS pairing (unusable in k8s) |
| **Extensible** | Sidecars & init containers | Chromium sidecar for browser automation, plus custom init containers and sidecars for proxies, log forwarders, etc. |
| **Cloud Native** | SA annotations & CA bundles | AWS IRSA / GCP Workload Identity via ServiceAccount annotations; CA bundle injection for corporate proxies |


## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  OpenClawInstance CR                                         │
│  (your declarative config)                                   │
└──────────────┬───────────────────────────────────────────────┘
               │ watch
               ▼
┌──────────────────────────────────────────────────────────────┐
│  OpenClaw Operator                                           │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Reconciler │  │   Webhooks   │  │  Prometheus Metrics  │  │
│  │            │  │  (validate   │  │  (reconcile count,   │  │
│  │  creates → │  │   & default) │  │   duration, phases)  │  │
│  └────────────┘  └──────────────┘  └──────────────────────┘  │
└──────────────┬───────────────────────────────────────────────┘
               │ manages
               ▼
┌──────────────────────────────────────────────────────────────┐
│  Managed Resources (per instance)                            │
│                                                              │
│  ServiceAccount ─► Role ─► RoleBinding    NetworkPolicy      │
│  ConfigMap        PVC      PDB            ServiceMonitor     │
│  GatewayToken Secret                                         │
│                                                              │
│  StatefulSet                                                 │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ Init: config -> pnpm* -> python* -> skills* -> custom  │  │
│  │                                        (* = opt-in)    │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │ OpenClaw Container        Chromium Sidecar (optional)  │  │
│  │ (AI agent runtime)        + custom sidecars            │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Service (ports 18789, 18793) ─► Ingress (optional)          │
└──────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Kubernetes 1.28+
- Helm 3

### 1. Install the operator

```bash
helm install openclaw-operator \
  oci://ghcr.io/openclaw-rocks/charts/openclaw-operator \
  --namespace openclaw-operator-system \
  --create-namespace
```

<details>
<summary>Alternative: install with Kustomize</summary>

```bash
# Install CRDs
make install

# Deploy the operator
make deploy IMG=ghcr.io/openclaw-rocks/openclaw-operator:latest
```

</details>

### 2. Create a secret with your API keys

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openclaw-api-keys
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-..."
```

### 3. Deploy an OpenClaw instance

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-agent
spec:
  envFrom:
    - secretRef:
        name: openclaw-api-keys
  storage:
    persistence:
      enabled: true
      size: 10Gi
```

```bash
kubectl apply -f secret.yaml -f openclawinstance.yaml
```

### 4. Verify

```bash
kubectl get openclawinstances
# NAME       PHASE     AGE
# my-agent   Running   2m

kubectl get pods
# NAME         READY   STATUS    AGE
# my-agent-0   1/1     Running   2m
```

## Configuration

### Inline config (openclaw.json)

```yaml
spec:
  config:
    raw:
      agents:
        defaults:
          model:
            primary: "anthropic/claude-sonnet-4-20250514"
          sandbox: true
      session:
        scope: "per-sender"
```

### External ConfigMap reference

```yaml
spec:
  config:
    configMapRef:
      name: my-openclaw-config
      key: openclaw.json
```

Config changes are detected via SHA-256 hashing and automatically trigger a rolling update. No manual restart needed.

### Gateway authentication

The operator automatically generates a gateway token Secret for each instance and injects it into both the config JSON (`gateway.auth.mode: token`) and the `OPENCLAW_GATEWAY_TOKEN` env var. This bypasses Bonjour/mDNS pairing, which is unusable in Kubernetes.

- The token is generated once and never overwritten — rotate it by editing the Secret directly
- If you set `gateway.auth.token` in your config or `OPENCLAW_GATEWAY_TOKEN` in `spec.env`, your value takes precedence
- `OPENCLAW_DISABLE_BONJOUR=1` is always set (mDNS does not work in k8s)
- To bring your own token Secret, set `spec.gateway.existingSecret` — the operator will use it instead of auto-generating one (the Secret must have a key named `token`)

### Chromium sidecar

Enable headless browser automation for web scraping, screenshots, and browser-based integrations:

```yaml
spec:
  chromium:
    enabled: true
    image:
      repository: ghcr.io/browserless/chromium
      tag: "v2.0.0"
    resources:
      requests:
        cpu: "250m"
        memory: "512Mi"
      limits:
        cpu: "1000m"
        memory: "2Gi"
```

When enabled, the operator automatically injects a `CHROMIUM_URL` environment variable into the main container and configures shared memory, security contexts, and health probes for the sidecar.

### Config merge mode

By default, the operator overwrites the config file on every pod restart. Set `mergeMode: merge` to deep-merge operator config with existing PVC config, preserving runtime changes made by the agent:

```yaml
spec:
  config:
    mergeMode: merge
    raw:
      agents:
        defaults:
          model:
            primary: "anthropic/claude-sonnet-4-20250514"
```

### Skill installation

Install ClawHub skills declaratively. The operator runs an init container that fetches each skill before the agent starts:

```yaml
spec:
  skills:
    - "@anthropic/mcp-server-fetch"
    - "@anthropic/mcp-server-filesystem"
```

### Runtime dependencies

Enable built-in init containers that install pnpm or Python/uv to the data PVC for MCP servers and skills:

```yaml
spec:
  runtimeDeps:
    pnpm: true    # Installs pnpm via corepack
    python: true  # Installs Python 3.12 + uv
```

### Custom init containers and sidecars

Add custom init containers (run after operator-managed ones) and sidecar containers:

```yaml
spec:
  initContainers:
    - name: fetch-models
      image: curlimages/curl:8.5.0
      command: ["sh", "-c", "curl -o /data/model.bin https://..."]
      volumeMounts:
        - name: data
          mountPath: /data
  sidecars:
    - name: cloud-sql-proxy
      image: gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.14.3
      args: ["--structured-logs", "my-project:us-central1:my-db"]
      ports:
        - containerPort: 5432
  sidecarVolumes:
    - name: proxy-creds
      secret:
        secretName: cloud-sql-proxy-sa
```

Reserved init container names (`init-config`, `init-pnpm`, `init-python`, `init-skills`) are rejected by the webhook.

### Extra volumes and mounts

Mount additional ConfigMaps, Secrets, or CSI volumes into the main container:

```yaml
spec:
  extraVolumes:
    - name: shared-data
      persistentVolumeClaim:
        claimName: shared-pvc
  extraVolumeMounts:
    - name: shared-data
      mountPath: /shared
```

### CA bundle injection

Inject a custom CA certificate bundle for environments with TLS-intercepting proxies or private CAs:

```yaml
spec:
  security:
    caBundle:
      configMapName: corporate-ca-bundle  # or secretName
      key: ca-bundle.crt                  # default key name
```

The bundle is mounted into all containers and the `SSL_CERT_FILE` / `NODE_EXTRA_CA_CERTS` environment variables are set automatically.

### ServiceAccount annotations

Add annotations to the managed ServiceAccount for cloud provider integrations:

```yaml
spec:
  security:
    rbac:
      serviceAccountAnnotations:
        # AWS IRSA
        eks.amazonaws.com/role-arn: "arn:aws:iam::123456789:role/openclaw"
        # GCP Workload Identity
        # iam.gke.io/gcp-service-account: "openclaw@project.iam.gserviceaccount.com"
```

### Auto-update

Opt into automatic version tracking so the operator detects new releases and rolls them out without manual intervention:

```yaml
spec:
  autoUpdate:
    enabled: true
    checkInterval: "24h"         # how often to poll the registry (1h–168h)
    backupBeforeUpdate: true     # back up the PVC before applying an update
    rollbackOnFailure: true      # auto-rollback if the new version fails health checks
    healthCheckTimeout: "10m"    # how long to wait for the pod to become ready (2m–30m)
```

When enabled, the operator:

1. **Resolves `latest` on creation** — if `spec.image.tag` is `latest`, the operator queries the OCI registry for the highest stable semver tag and pins the instance to it
2. **Polls for new versions** — on each `checkInterval`, the operator checks the registry for newer semver tags
3. **Backs up first** (optional) — if `backupBeforeUpdate` is true and persistence is enabled, a B2 backup job runs before updating
4. **Applies the update** — patches `spec.image.tag` to the new version, triggering a StatefulSet rolling update
5. **Health checks** — monitors the StatefulSet for `healthCheckTimeout`; if the pod becomes ready, the update is confirmed
6. **Auto-rollback** — if the pod fails to become ready within the timeout, the operator reverts the image tag (and optionally restores the PVC from the pre-update backup)

The rollback system includes safety mechanisms:

- **Failed version tracking** — a version that fails health checks is recorded and skipped in future checks (cleared when a newer version is published)
- **Circuit breaker** — after 3 consecutive rollbacks, auto-update pauses and emits a warning event; reset on any successful update
- **Backup restore** — when `backupBeforeUpdate` is true, rollback restores the PVC from the pre-update snapshot, fully reverting both code and data

Update progress is tracked in `status.autoUpdate`:

```bash
kubectl get openclawinstance my-agent -o jsonpath='{.status.autoUpdate}' | jq .
```

```json
{
  "currentVersion": "2026.2.15",
  "latestVersion": "2026.2.15",
  "lastCheckTime": "2026-02-16T10:30:00Z",
  "lastUpdateTime": "2026-02-16T10:30:05Z"
}
```

Auto-update is a no-op for digest-pinned images (`spec.image.digest`). The operator only considers stable releases (no pre-release tags).

### All configuration options

| Field | Description | Default |
|-------|-------------|---------|
| **Image** | | |
| `spec.image.repository` | Container image | `ghcr.io/openclaw/openclaw` |
| `spec.image.tag` | Image tag | `latest` |
| `spec.image.digest` | Image digest (overrides tag) | - |
| `spec.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `spec.image.pullSecrets` | Secrets for private registries | `[]` |
| **Config** | | |
| `spec.config.raw` | Inline openclaw.json | - |
| `spec.config.configMapRef` | External ConfigMap reference | - |
| `spec.config.mergeMode` | Config apply strategy: `overwrite` or `merge` | `overwrite` |
| `spec.config.format` | Config file format: `json` or `json5` | `json` |
| **Environment** | | |
| `spec.envFrom` | Secret/ConfigMap env sources | `[]` |
| `spec.env` | Additional environment variables | `[]` |
| **Skills & Runtime** | | |
| `spec.skills` | ClawHub skills to install via init container | `[]` |
| `spec.runtimeDeps.pnpm` | Install pnpm via corepack | `false` |
| `spec.runtimeDeps.python` | Install Python 3.12 + uv | `false` |
| **Custom Containers** | | |
| `spec.initContainers` | Additional init containers (max 10) | `[]` |
| `spec.sidecars` | Additional sidecar containers | `[]` |
| `spec.sidecarVolumes` | Volumes for sidecar containers | `[]` |
| `spec.extraVolumes` | Additional pod volumes (max 10) | `[]` |
| `spec.extraVolumeMounts` | Additional volume mounts for main container (max 10) | `[]` |
| **Resources** | | |
| `spec.resources.requests.cpu` | CPU request | `500m` |
| `spec.resources.requests.memory` | Memory request | `1Gi` |
| `spec.resources.limits.cpu` | CPU limit | `2000m` |
| `spec.resources.limits.memory` | Memory limit | `4Gi` |
| **Storage** | | |
| `spec.storage.persistence.enabled` | Persistent storage | `true` |
| `spec.storage.persistence.size` | PVC size | `10Gi` |
| `spec.storage.persistence.storageClass` | StorageClass name | cluster default |
| **Security** | | |
| `spec.security.containerSecurityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `spec.security.podSecurityContext.fsGroupChangePolicy` | Volume ownership change policy | - |
| `spec.security.networkPolicy.enabled` | NetworkPolicy | `true` |
| `spec.security.networkPolicy.additionalEgress` | Custom egress rules | `[]` |
| `spec.security.rbac.serviceAccountAnnotations` | ServiceAccount annotations (IRSA/WI) | `{}` |
| `spec.security.caBundle.configMapName` | CA bundle ConfigMap name | - |
| `spec.security.caBundle.secretName` | CA bundle Secret name | - |
| `spec.security.caBundle.key` | Key in ConfigMap/Secret containing CA bundle | `ca-bundle.crt` |
| **Gateway** | | |
| `spec.gateway.existingSecret` | Use existing Secret for gateway token (key: `token`) | auto-generated |
| **Chromium** | | |
| `spec.chromium.enabled` | Chromium sidecar | `false` |
| **Networking** | | |
| `spec.networking.service.type` | Service type | `ClusterIP` |
| `spec.networking.ingress.enabled` | Ingress | `false` |
| **Observability** | | |
| `spec.observability.metrics.enabled` | Prometheus metrics | `true` |
| `spec.observability.metrics.serviceMonitor.enabled` | ServiceMonitor | `false` |
| **Availability** | | |
| `spec.availability.podDisruptionBudget.enabled` | PDB | `true` |
| `spec.availability.nodeSelector` | Node label selector for scheduling | `{}` |
| `spec.availability.tolerations` | Pod tolerations | `[]` |
| `spec.availability.affinity` | Pod affinity/anti-affinity rules | - |
| **Workspace** | | |
| `spec.workspace.initialFiles` | Files to create in workspace before start | `{}` |
| `spec.workspace.initialDirectories` | Directories to create in workspace before start | `[]` |
| **Auto-Update** | | |
| `spec.autoUpdate.enabled` | Automatic version updates | `false` |
| `spec.autoUpdate.checkInterval` | Registry poll interval (Go duration) | `24h` |
| `spec.autoUpdate.backupBeforeUpdate` | Backup PVC before updating | `true` |
| `spec.autoUpdate.rollbackOnFailure` | Auto-rollback if update fails health check | `true` |
| `spec.autoUpdate.healthCheckTimeout` | Time to wait for pod readiness after update (2m–30m) | `10m` |
| **Backup/Restore** | | |
| `spec.restoreFrom` | B2 backup path to restore workspace from | - |

See the [full example](config/samples/openclaw_v1alpha1_openclawinstance_full.yaml) for every available field, or the [API reference](docs/api-reference.md) for detailed documentation.

## Security

The operator follows a **secure-by-default** philosophy. Every instance ships with hardened settings out of the box, with no extra configuration needed.

### Defaults

- **Non-root execution**: containers run as UID 1000; root (UID 0) is blocked by the validating webhook
- **Read-only root filesystem**: enabled by default for the main container and the Chromium sidecar; the PVC at `~/.openclaw/` provides writable home, and a `/tmp` emptyDir handles temp files
- **All capabilities dropped**: no ambient Linux capabilities
- **Seccomp RuntimeDefault**: syscall filtering enabled
- **Default-deny NetworkPolicy**: only DNS (53) and HTTPS (443) egress allowed; ingress limited to same namespace
- **Minimal RBAC**: each instance gets its own ServiceAccount with read-only access to its own ConfigMap; operator can create/update Secrets only for operator-managed gateway tokens
- **No automatic token mounting**: `automountServiceAccountToken: false` on both ServiceAccounts and pod specs
- **Secret validation**: the operator checks that all referenced Secrets exist and sets a `SecretsReady` condition

### Validating webhook

| Check | Severity | Behavior |
|-------|----------|----------|
| `runAsUser: 0` | Error | Blocked: root execution not allowed |
| Reserved init container name | Error | `init-config`, `init-pnpm`, `init-python`, `init-skills` are reserved |
| Invalid skill name | Error | Only alphanumeric, `-`, `_`, `/`, `.`, `@` allowed (max 128 chars) |
| Invalid CA bundle config | Error | Exactly one of `configMapName` or `secretName` must be set |
| JSON5 with inline raw config | Error | JSON5 requires `configMapRef` (inline must be valid JSON) |
| JSON5 with merge mode | Error | JSON5 is not compatible with `mergeMode: merge` |
| Invalid `checkInterval` | Error | Must be a valid Go duration between 1h and 168h |
| Invalid `healthCheckTimeout` | Error | Must be a valid Go duration between 2m and 30m |

| NetworkPolicy disabled | Warning | Deployment proceeds with a warning |
| Ingress without TLS | Warning | Deployment proceeds with a warning |
| Chromium without digest pinning | Warning | Deployment proceeds with a warning |
| Auto-update with digest pin | Warning | Digest overrides auto-update; updates won't apply |
| `readOnlyRootFilesystem` disabled | Warning | Proceeds with a security recommendation |
| No AI provider keys detected | Warning | Scans `env`/`envFrom` for known provider env vars |
| Unknown config keys | Warning | Warns on unrecognized top-level keys in `spec.config.raw` |

### Custom network rules

Allow egress to internal services or specific CIDRs:

```yaml
spec:
  security:
    networkPolicy:
      enabled: true
      additionalEgress:
        - ports:
            - protocol: TCP
              port: 5432
          to:
            - ipBlock:
                cidr: 10.0.0.0/8
```

## Observability

### Prometheus metrics

| Metric | Type | Description |
|--------|------|-------------|
| `openclaw_reconcile_total` | Counter | Reconciliations by result (success/error) |
| `openclaw_reconcile_duration_seconds` | Histogram | Reconciliation latency |
| `openclaw_instance_phase` | Gauge | Current phase per instance |
| `openclaw_autoupdate_checks_total` | Counter | Auto-update version checks by result |
| `openclaw_autoupdate_applied_total` | Counter | Successful auto-updates applied |
| `openclaw_autoupdate_rollbacks_total` | Counter | Auto-update rollbacks triggered |

### ServiceMonitor

```yaml
spec:
  observability:
    metrics:
      enabled: true
      serviceMonitor:
        enabled: true
        interval: 15s
        labels:
          release: prometheus
```

### Instance status

```bash
kubectl get openclawinstance my-agent -o yaml
```

```yaml
status:
  phase: Running
  conditions:
    - type: Ready
      status: "True"
    - type: StatefulSetReady
      status: "True"
    - type: NetworkPolicyReady
      status: "True"
  gatewayEndpoint: my-agent.default.svc:18789
  canvasEndpoint: my-agent.default.svc:18793
```

Phases: `Pending` → `Restoring` → `Provisioning` → `Running` | `Updating` | `BackingUp` | `Degraded` | `Failed` | `Terminating`

## Deployment Guides

Platform-specific deployment guides are available for:

- [AWS EKS](docs/deployment.md#aws-eks)
- [Google GKE](docs/deployment.md#google-gke)
- [Azure AKS](docs/deployment.md#azure-aks)
- [Kind (local development)](docs/deployment.md#kind)

## Development

```bash
# Clone and set up
git clone https://github.com/OpenClaw-rocks/k8s-operator.git
cd k8s-operator
go mod download

# Generate code and manifests
make generate manifests

# Run tests
make test

# Run linter
make lint

# Run locally against a Kind cluster
kind create cluster
make install
make run
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development guide.

## Roadmap

- **v1.0.0**: API graduation to `v1`, conformance test suite, semver constraints for auto-update, HPA integration, cert-manager integration, multi-cluster support

See the full [roadmap](ROADMAP.md) for details.

## Don't Want to Self-Host?

[OpenClaw.rocks](https://openclaw.rocks) offers fully managed hosting starting at **EUR 15/mo**. No Kubernetes cluster required. Setup, updates, and 24/7 uptime handled for you.

## Contributing

Contributions are welcome. Please open an issue to discuss significant changes before submitting a PR. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Disclaimer: AI-Assisted Development

This repository is developed and maintained collaboratively by a human and [Claude Code](https://claude.ai/claude-code). This includes writing code, reviewing and commenting on issues, triaging bugs, and merging pull requests. The human reads everything and acts as the final guard, but Claude does the heavy lifting — from diagnosis to implementation to CI.

In the future, this repo may be fully autonomously operated, whether we humans like that or not.

## License

Apache License 2.0, the same license used by Kubernetes, Prometheus, and most CNCF projects. See [LICENSE](LICENSE) for details.
