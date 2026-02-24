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

## Agents That Adapt Themselves

Agents can autonomously install skills, patch their config, add environment variables, and seed workspace files - all through the Kubernetes API, validated by the operator on every request.

```yaml
# 1. Enable self-configure on the instance
spec:
  selfConfigure:
    enabled: true
    allowedActions: [skills, config, envVars, workspaceFiles]
```

```yaml
# 2. The agent creates this to install a skill at runtime
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawSelfConfig
metadata:
  name: add-fetch-skill
spec:
  instanceRef: my-agent
  addSkills:
    - "@anthropic/mcp-server-fetch"
```

Every request is validated against the instance's allowlist policy. Protected config keys cannot be overwritten, and denied requests are logged with a reason. See [Self-configure](#self-configure) for details.

## Features

| | Feature | Details |
|---|---|---|
| **Declarative** | Single CRD | One resource defines the entire stack: StatefulSet, Service, RBAC, NetworkPolicy, PVC, PDB, Ingress, and more |
| **Adaptive** | Agent self-configure | Agents autonomously install skills, patch config, and adapt their environment via the K8s API - every change validated against an allowlist policy |
| **Secure** | Hardened by default | Non-root (UID 1000), read-only root filesystem, all capabilities dropped, seccomp RuntimeDefault, default-deny NetworkPolicy, validating webhook |
| **Observable** | Built-in metrics | Prometheus metrics, ServiceMonitor integration, structured JSON logging, Kubernetes events |
| **Flexible** | Provider-agnostic config | Use any AI provider (Anthropic, OpenAI, or others) via environment variables and inline or external config |
| **Config Modes** | Merge or overwrite | `overwrite` replaces config on restart; `merge` deep-merges with PVC config, preserving runtime changes. Config is restored on every container restart via init container. |
| **Skills** | Declarative install | Install ClawHub skills or npm packages via `spec.skills` - supports `npm:` prefix for npmjs.com packages |
| **Runtime Deps** | pnpm & Python/uv | Built-in init containers install pnpm (via corepack) or Python 3.12 + uv for MCP servers and skills |
| **Auto-Update** | OCI registry polling | Opt-in version tracking: checks the registry for new semver releases, backs up first, rolls out, and auto-rolls back if the new version fails health checks |
| **Scalable** | Auto-scaling | HPA integration with CPU and memory metrics, min/max replica bounds, automatic StatefulSet replica management |
| **Resilient** | Self-healing lifecycle | PodDisruptionBudgets, health probes, automatic config rollouts via content hashing, 5-minute drift detection |
| **Backup/Restore** | S3-backed snapshots | Automatic backup to S3-compatible storage on instance deletion; restore into a new instance from any snapshot |
| **Workspace Seeding** | Initial files & dirs | Pre-populate the workspace with files and directories before the agent starts |
| **Gateway Auth** | Auto-generated tokens | Automatic gateway token Secret per instance, bypassing mDNS pairing (unusable in k8s) |
| **Tailscale** | Tailnet access | Expose via Tailscale Serve or Funnel with SSO auth - no Ingress needed |
| **Extensible** | Sidecars & init containers | Chromium for browser automation, Ollama for local LLMs, Tailscale for tailnet access, plus custom init containers and sidecars |
| **Cloud Native** | SA annotations & CA bundles | AWS IRSA / GCP Workload Identity via ServiceAccount annotations; CA bundle injection for corporate proxies |


## Architecture

```
+-----------------------------------------------------------------+
|  OpenClawInstance CR          OpenClawSelfConfig CR              |
|  (your declarative config)   (agent self-modification requests) |
+---------------+-------------------------------------------------+
                | watch
                v
+-----------------------------------------------------------------+
|  OpenClaw Operator                                              |
|  +-----------+  +-------------+  +----------------------------+ |
|  | Reconciler|  |   Webhooks  |  |   Prometheus Metrics       | |
|  |           |  |  (validate  |  |  (reconcile count,         | |
|  |  creates ->  |   & default)|  |   duration, phases)        | |
|  +-----------+  +-------------+  +----------------------------+ |
+---------------+-------------------------------------------------+
                | manages
                v
+-----------------------------------------------------------------+
|  Managed Resources (per instance)                               |
|                                                                 |
|  ServiceAccount -> Role -> RoleBinding    NetworkPolicy         |
|  ConfigMap        PVC      PDB            ServiceMonitor        |
|  GatewayToken Secret                                            |
|                                                                 |
|  StatefulSet                                                    |
|  +-----------------------------------------------------------+ |
|  | Init: config -> pnpm* -> python* -> skills* -> custom      | |
|  |                                        (* = opt-in)        | |
|  +------------------------------------------------------------+ |
|  | OpenClaw Container  Gateway Proxy (nginx)                  | |
|  |                     Chromium (opt) / Ollama (opt)          | |
|  |                     Tailscale (opt) + custom sidecars      | |
|  +------------------------------------------------------------+ |
|                                                                 |
|  Service (default: 18789, 18793 or custom) -> Ingress (opt)     |
+-----------------------------------------------------------------+
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

- The token is generated once and never overwritten - rotate it by editing the Secret directly
- If you set `gateway.auth.token` in your config or `OPENCLAW_GATEWAY_TOKEN` in `spec.env`, your value takes precedence
- To bring your own token Secret, set `spec.gateway.existingSecret` - the operator will use it instead of auto-generating one (the Secret must have a key named `token`)

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

When enabled, the operator automatically:
- Injects a `CHROMIUM_URL` environment variable into the main container
- Configures browser profiles in the OpenClaw config - both `"default"` and `"chrome"` profiles are set to point at the sidecar's CDP endpoint, so browser tool calls work regardless of which profile name the LLM passes
- Sets up shared memory, security contexts, and health probes for the sidecar

### Ollama sidecar

Run local LLMs alongside your agent for private, low-latency inference without external API calls:

```yaml
spec:
  ollama:
    enabled: true
    models:
      - llama3.2
      - nomic-embed-text
    gpu: 1
    storage:
      sizeLimit: 30Gi
    resources:
      requests:
        cpu: "1"
        memory: "4Gi"
      limits:
        cpu: "4"
        memory: "16Gi"
```

When enabled, the operator:
- Injects an `OLLAMA_HOST` environment variable into the main container
- Pre-pulls specified models via an init container before the agent starts
- Configures GPU resource limits when `gpu` is set (`nvidia.com/gpu`)
- Mounts a model cache volume (emptyDir by default, or an existing PVC via `storage.existingClaim`)

See [Custom AI Providers](docs/custom-providers.md) for configuring OpenClaw to use Ollama models via `llmConfig`.

### Web terminal sidecar

Provide browser-based shell access to running instances for debugging and inspection without requiring `kubectl exec`:

```yaml
spec:
  webTerminal:
    enabled: true
    readOnly: false
    credential:
      secretRef:
        name: my-terminal-creds
    resources:
      requests:
        cpu: "50m"
        memory: "64Mi"
      limits:
        cpu: "200m"
        memory: "128Mi"
```

When enabled, the operator:
- Injects a [ttyd](https://github.com/tsl0922/ttyd) sidecar container on port 7681
- Mounts the instance data volume at `/home/openclaw/.openclaw` so you can inspect config, logs, and data files
- Adds the web terminal port to the Service and NetworkPolicy for external access
- Supports basic auth via a Secret with `username` and `password` keys
- Supports read-only mode (`readOnly: true`) for production environments where shell input should be disabled

### Tailscale integration

Expose your instance via [Tailscale](https://tailscale.com) Serve (tailnet-only) or Funnel (public internet) - no Ingress or LoadBalancer needed:

```yaml
spec:
  tailscale:
    enabled: true
    mode: serve          # "serve" (tailnet only) or "funnel" (public internet)
    authKeySecretRef:
      name: tailscale-auth
    authSSO: true        # allow passwordless login for tailnet members
    hostname: my-agent   # defaults to instance name
    image:
      repository: ghcr.io/tailscale/tailscale  # default
      tag: latest
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 256Mi
```

When enabled, the operator runs a **Tailscale sidecar** (`tailscaled`) that handles serve/funnel declaratively via `TS_SERVE_CONFIG`. An **init container** copies the `tailscale` CLI binary to a shared volume so the main container can call `tailscale whois` for SSO authentication. The sidecar runs in userspace mode (`TS_USERSPACE=true`) - no `NET_ADMIN` capability needed.

Use ephemeral+reusable auth keys from the [Tailscale admin console](https://login.tailscale.com/admin/settings/keys). When `authSSO` is enabled, tailnet members can authenticate without a gateway token.

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

Install skills declaratively. The operator runs an init container that fetches each skill before the agent starts. Entries use ClawHub by default, or prefix with `npm:` to install from npmjs.com:

```yaml
spec:
  skills:
    - "@anthropic/mcp-server-fetch"       # ClawHub (default)
    - "npm:@openclaw/matrix"              # npm package from npmjs.com
```

npm lifecycle scripts are disabled globally on the init container (`NPM_CONFIG_IGNORE_SCRIPTS=true`) to mitigate supply chain attacks.

### Self-configure

Allow agents to modify their own configuration by creating `OpenClawSelfConfig` resources via the K8s API. The operator validates each request against the instance's `allowedActions` policy before applying changes:

```yaml
spec:
  selfConfigure:
    enabled: true
    allowedActions:
      - skills        # add/remove skills
      - config        # patch openclaw.json
      - workspaceFiles # add/remove workspace files
      - envVars       # add/remove environment variables
```

When enabled, the operator:
- Grants the instance's ServiceAccount RBAC permissions to read its own CRD and create `OpenClawSelfConfig` resources
- Enables SA token automounting so the agent can authenticate with the K8s API
- Injects a `SELFCONFIG.md` skill file and `selfconfig.sh` helper script into the workspace
- Opens port 6443 egress in the NetworkPolicy for K8s API access

The agent creates a request like:

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawSelfConfig
metadata:
  name: add-fetch-skill
spec:
  instanceRef: my-agent
  addSkills:
    - "@anthropic/mcp-server-fetch"
```

The operator validates the request, applies it to the parent `OpenClawInstance`, and sets the request's status to `Applied`, `Denied`, or `Failed`. Terminal requests are auto-deleted after 1 hour.

See the [API reference](docs/api-reference.md) for the full `OpenClawSelfConfig` CRD spec and `spec.selfConfigure` fields.

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

Reserved init container names (`init-config`, `init-pnpm`, `init-python`, `init-skills`, `init-ollama`) are rejected by the webhook.

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

### Custom service ports

By default the operator creates a Service with the gateway (18789) and canvas (18793) ports. To expose custom ports instead (e.g., for a non-default application), set `spec.networking.service.ports`:

```yaml
spec:
  networking:
    service:
      type: ClusterIP
      ports:
        - name: http
          port: 3978
          targetPort: 3978
```

When `ports` is set, it fully replaces the default ports -- including the Chromium port if the sidecar is enabled. To keep the defaults alongside custom ports, include them explicitly. If `targetPort` is omitted it defaults to `port`. See the [API reference](docs/api-reference.md#specnetworkingservice) for all fields.

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
    checkInterval: "24h"         # how often to poll the registry (1h-168h)
    backupBeforeUpdate: true     # back up the PVC before applying an update
    rollbackOnFailure: true      # auto-rollback if the new version fails health checks
    healthCheckTimeout: "10m"    # how long to wait for the pod to become ready (2m-30m)
```

When enabled, the operator resolves `latest` to the highest stable semver tag on creation, then polls for newer versions on each `checkInterval`. Before updating, it optionally runs an S3 backup, then patches the image tag and monitors the rollout. If the pod fails to become ready within `healthCheckTimeout`, it reverts the image tag and (optionally) restores the PVC from the pre-update snapshot.

Safety mechanisms include failed-version tracking (skips versions that failed health checks), a circuit breaker (pauses after 3 consecutive rollbacks), and full data restore when `backupBeforeUpdate` is enabled. Auto-update is a no-op for digest-pinned images (`spec.image.digest`).

See `status.autoUpdate` for update progress: `kubectl get openclawinstance my-agent -o jsonpath='{.status.autoUpdate}'`

### What the operator manages automatically

These behaviors are always applied - no configuration needed:

| Behavior | Details |
|----------|---------|
| `gateway.bind=loopback` | Always injected into config; an nginx reverse proxy sidecar exposes the gateway and canvas ports for external access |
| Gateway auth token | Auto-generated Secret per instance; injected into config and env |
| `OPENCLAW_DISABLE_BONJOUR=1` | Always set (mDNS does not work in Kubernetes) |
| Browser profiles | When Chromium is enabled, `"default"` and `"chrome"` profiles are auto-configured with the sidecar's CDP endpoint |
| Tailscale serve config | When Tailscale is enabled, a `tailscale-serve.json` key is added to the ConfigMap for the sidecar's `TS_SERVE_CONFIG` |
| Config hash rollouts | Config changes trigger rolling updates via SHA-256 hash annotation |
| Config restoration | The init container restores config on every pod restart (overwrite or merge mode) |

For the full list of configuration options, see the [API reference](docs/api-reference.md) and the [full sample YAML](config/samples/openclaw_v1alpha1_openclawinstance_full.yaml).

## Security

The operator follows a **secure-by-default** philosophy. Every instance ships with hardened settings out of the box, with no extra configuration needed.

### Defaults

- **Non-root execution**: containers run as UID 1000; root (UID 0) is blocked by the validating webhook (exception: Ollama sidecar requires root per the official image)
- **Read-only root filesystem**: enabled by default for the main container and the Chromium sidecar; the PVC at `~/.openclaw/` provides writable home, and a `/tmp` emptyDir handles temp files
- **All capabilities dropped**: no ambient Linux capabilities
- **Seccomp RuntimeDefault**: syscall filtering enabled
- **Default-deny NetworkPolicy**: only DNS (53) and HTTPS (443) egress allowed; ingress limited to same namespace
- **Minimal RBAC**: each instance gets its own ServiceAccount with read-only access to its own ConfigMap; operator can create/update Secrets only for operator-managed gateway tokens
- **No automatic token mounting**: `automountServiceAccountToken: false` on both ServiceAccounts and pod specs (enabled only when `selfConfigure` is active)
- **Secret validation**: the operator checks that all referenced Secrets exist and sets a `SecretsReady` condition

### Validating webhook

| Check | Severity | Behavior |
|-------|----------|----------|
| `runAsUser: 0` | Error | Blocked: root execution not allowed |
| Reserved init container name | Error | `init-config`, `init-pnpm`, `init-python`, `init-skills`, `init-ollama` are reserved |
| Invalid skill name | Error | Only alphanumeric, `-`, `_`, `/`, `.`, `@` allowed (max 128 chars). `npm:` prefix is allowed for npm packages; bare `npm:` is rejected |
| Invalid CA bundle config | Error | Exactly one of `configMapName` or `secretName` must be set |
| JSON5 with inline raw config | Error | JSON5 requires `configMapRef` (inline must be valid JSON) |
| JSON5 with merge mode | Error | JSON5 is not compatible with `mergeMode: merge` |
| Invalid `checkInterval` | Error | Must be a valid Go duration between 1h and 168h |
| Invalid `healthCheckTimeout` | Error | Must be a valid Go duration between 2m and 30m |

<details>
<summary>Warning-level checks (deployment proceeds with a warning)</summary>

| Check | Behavior |
|-------|----------|
| NetworkPolicy disabled | Deployment proceeds with a warning |
| Ingress without TLS | Deployment proceeds with a warning |
| Chromium without digest pinning | Deployment proceeds with a warning |
| Ollama without digest pinning | Deployment proceeds with a warning |
| Web terminal without digest pinning | Deployment proceeds with a warning |
| Ollama runs as root | Required by official image; informational |
| Auto-update with digest pin | Digest overrides auto-update; updates won't apply |
| `readOnlyRootFilesystem` disabled | Proceeds with a security recommendation |
| No AI provider keys detected | Scans `env`/`envFrom` for known provider env vars |
| Unknown config keys | Warns on unrecognized top-level keys in `spec.config.raw` |

</details>

## Observability

### Prometheus metrics

| Metric | Type | Description |
|--------|------|-------------|
| `openclaw_reconcile_total` | Counter | Reconciliations by result (success/error) |
| `openclaw_reconcile_duration_seconds` | Histogram | Reconciliation latency |
| `openclaw_instance_phase` | Gauge | Current phase per instance |
| `openclaw_instance_info` | Gauge | Instance metadata for PromQL joins (always 1) |
| `openclaw_instance_ready` | Gauge | Whether instance pod is ready (1/0) |
| `openclaw_managed_instances` | Gauge | Total number of managed instances |
| `openclaw_resource_creation_failures_total` | Counter | Resource creation failures |
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

### PrometheusRule (alerts)

Auto-provisions a PrometheusRule with 7 alerts including runbook URLs:

```yaml
spec:
  observability:
    metrics:
      prometheusRule:
        enabled: true
        labels:
          release: kube-prometheus-stack  # must match Prometheus ruleSelector
        runbookBaseURL: https://openclaw.rocks/docs/runbooks  # default
```

Alerts: `OpenClawReconcileErrors`, `OpenClawInstanceDegraded`, `OpenClawSlowReconciliation`, `OpenClawPodCrashLooping`, `OpenClawPodOOMKilled`, `OpenClawPVCNearlyFull`, `OpenClawAutoUpdateRollback`

### Grafana dashboards

Auto-provisions two Grafana dashboard ConfigMaps (discovered via the `grafana_dashboard: "1"` label):

```yaml
spec:
  observability:
    metrics:
      grafanaDashboard:
        enabled: true
        folder: OpenClaw  # Grafana folder (default)
        labels:
          grafana_dashboard_instance: my-grafana  # optional extra labels
```

Dashboards:
- **OpenClaw Operator** - fleet overview with reconciliation metrics, instance table, workqueue, and auto-update panels
- **OpenClaw Instance** - per-instance detail with CPU, memory, storage, network, and pod health panels

### Auto-Scaling (HPA)

Enable horizontal pod auto-scaling to automatically adjust the number of replicas based on CPU and memory utilization:

```yaml
spec:
  availability:
    autoScaling:
      enabled: true
      minReplicas: 1
      maxReplicas: 10
      targetCPUUtilization: 80
      targetMemoryUtilization: 70  # optional
```

When enabled, the operator creates a `HorizontalPodAutoscaler` targeting the StatefulSet and sets the StatefulSet's replica count to nil so the HPA manages scaling. The HPA is deleted when auto-scaling is disabled.

Phases: `Pending` -> `Restoring` -> `Provisioning` -> `Running` | `Updating` | `BackingUp` | `Degraded` | `Failed` | `Terminating`

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

This repository is developed and maintained collaboratively by a human and [Claude Code](https://claude.ai/claude-code). This includes writing code, reviewing and commenting on issues, triaging bugs, and merging pull requests. The human reads everything and acts as the final guard, but Claude does the heavy lifting - from diagnosis to implementation to CI.

In the future, this repo may be fully autonomously operated, whether we humans like that or not.

## License

Apache License 2.0, the same license used by Kubernetes, Prometheus, and most CNCF projects. See [LICENSE](LICENSE) for details.
