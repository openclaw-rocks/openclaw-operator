# API Reference

## OpenClawInstance (v1alpha1)

**Group**: `openclaw.rocks`
**Version**: `v1alpha1`
**Kind**: `OpenClawInstance`
**Scope**: Namespaced

An `OpenClawInstance` represents a single deployment of the OpenClaw AI assistant in a Kubernetes cluster. The operator watches these resources and reconciles a full stack of dependent objects (StatefulSet, Service, RBAC, NetworkPolicy, storage, and more).

### Print Columns

When listing resources with `kubectl get openclawinstances`, the following columns are displayed:

| Column    | JSON Path                                          |
|-----------|----------------------------------------------------|
| Phase     | `.status.phase`                                    |
| Ready     | `.status.conditions[?(@.type=='Ready')].status`    |
| Gateway   | `.status.gatewayEndpoint`                          |
| Age       | `.metadata.creationTimestamp`                      |

---

## Spec Fields

### spec.image

Container image configuration for the main OpenClaw workload.

| Field          | Type                         | Default                        | Description                                                       |
|----------------|------------------------------|--------------------------------|-------------------------------------------------------------------|
| `repository`   | `string`                     | `ghcr.io/openclaw/openclaw`    | Container image repository.                                       |
| `tag`          | `string`                     | `latest`                       | Container image tag.                                              |
| `digest`       | `string`                     | --                             | Image digest (overrides `tag` if set). Format: `sha256:abc...`.   |
| `pullPolicy`   | `string`                     | `IfNotPresent`                 | Image pull policy. One of: `Always`, `IfNotPresent`, `Never`.     |
| `pullSecrets`  | `[]LocalObjectReference`     | --                             | List of Secrets for pulling from private registries.              |

### spec.config

Configuration for the OpenClaw application (`openclaw.json`).

| Field          | Type                  | Default       | Description                                                                |
|----------------|-----------------------|---------------|----------------------------------------------------------------------------|
| `configMapRef` | `ConfigMapKeySelector`| --            | Reference to an external ConfigMap. If set, `raw` is ignored.              |
| `raw`          | `RawConfig`           | --            | Inline JSON configuration. The operator creates a managed ConfigMap.       |
| `mergeMode`    | `string`              | `overwrite`   | How config is applied to the PVC. `overwrite` replaces on every restart. `merge` deep-merges with existing PVC config, preserving runtime changes. |
| `format`       | `string`              | `json`        | Config file format. `json` (standard JSON) or `json5` (JSON5 with comments/trailing commas). JSON5 requires `configMapRef` - inline `raw` must be valid JSON. JSON5 is converted to standard JSON by the init container using npx json5. |

**ConfigMapKeySelector:**

| Field  | Type     | Default          | Description                            |
|--------|----------|------------------|----------------------------------------|
| `name` | `string` | (required)       | Name of the ConfigMap.                 |
| `key`  | `string` | `openclaw.json`  | Key within the ConfigMap to mount.     |

### spec.workspace

Configures initial workspace files seeded into the instance. Files are copied once on first boot and never overwritten, so agent modifications survive pod restarts.

| Field                | Type                  | Default | Description                                                                                       |
|----------------------|-----------------------|---------|---------------------------------------------------------------------------------------------------|
| `initialFiles`       | `map[string]string`   | --      | Maps filenames to their content. Each file is written to the workspace directory only if it does not already exist. Max 50 entries. |
| `initialDirectories` | `[]string`            | --      | Directories to create (`mkdir -p`) inside the workspace directory. Nested paths like `tools/scripts` are allowed. Max 20 items. |

```yaml
spec:
  workspace:
    initialDirectories:
      - tools/scripts
      - data
    initialFiles:
      README.md: |
        # My Workspace
        This workspace is managed by OpenClaw.
      tools/scripts/setup.sh: |
        #!/bin/bash
        echo "Hello from setup"
```

### spec.skills

| Field    | Type       | Default | Description                                                                                       |
|----------|------------|---------|---------------------------------------------------------------------------------------------------|
| `skills` | `[]string` | --      | Skills to install via init container. Each entry is a ClawHub skill identifier (e.g., `@anthropic/mcp-server-fetch`) or an npm package prefixed with `npm:` (e.g., `npm:@openclaw/matrix`). npm lifecycle scripts are disabled for security. Max 20 items. |

```yaml
spec:
  skills:
    - "@anthropic/mcp-server-fetch"
    - "npm:@openclaw/matrix"
```

### spec.envFrom

| Field     | Type                  | Default | Description                                                                       |
|-----------|-----------------------|---------|-----------------------------------------------------------------------------------|
| `envFrom` | `[]EnvFromSource`     | --      | Sources to populate environment variables (e.g., Secrets for `ANTHROPIC_API_KEY`). |

Standard Kubernetes `EnvFromSource`. Commonly used to inject API keys from a Secret:

```yaml
spec:
  envFrom:
    - secretRef:
        name: openclaw-api-keys
```

### spec.env

| Field | Type           | Default | Description                                         |
|-------|----------------|---------|-----------------------------------------------------|
| `env` | `[]EnvVar`     | --      | Individual environment variables to set.             |

Standard Kubernetes `EnvVar`. Example:

```yaml
spec:
  env:
    - name: LOG_LEVEL
      value: "debug"
```

### spec.resources

Compute resource requirements for the main OpenClaw container.

| Field                | Type     | Default  | Description                          |
|----------------------|----------|----------|--------------------------------------|
| `requests.cpu`       | `string` | `500m`   | Minimum CPU (e.g., `500m`, `2`).     |
| `requests.memory`    | `string` | `1Gi`    | Minimum memory (e.g., `512Mi`).      |
| `limits.cpu`         | `string` | `2000m`  | Maximum CPU.                         |
| `limits.memory`      | `string` | `4Gi`    | Maximum memory.                      |

### spec.security

Security-related configuration for the instance.

#### spec.security.podSecurityContext

| Field                 | Type                          | Default          | Description                                                                                |
|-----------------------|-------------------------------|------------------|--------------------------------------------------------------------------------------------|
| `runAsUser`           | `*int64`                      | `1000`           | UID to run as. Setting to `0` is rejected by webhook.                                      |
| `runAsGroup`          | `*int64`                      | `1000`           | GID to run as.                                                                             |
| `fsGroup`             | `*int64`                      | `1000`           | Supplemental group for volume ownership.                                                   |
| `fsGroupChangePolicy` | `*PodFSGroupChangePolicy`    | --               | Behavior for changing volume ownership. `OnRootMismatch` skips recursive chown when ownership already matches, improving startup time for large PVCs. `Always` recursively chowns on every mount (Kubernetes default). |
| `runAsNonRoot`        | `*bool`                       | `true`           | Require non-root execution. Warns if set to `false`.                                       |

#### spec.security.containerSecurityContext

| Field                      | Type              | Default | Description                                                    |
|----------------------------|-------------------|---------|----------------------------------------------------------------|
| `allowPrivilegeEscalation` | `*bool`           | `false` | Allow privilege escalation. Warns if set to `true`.            |
| `readOnlyRootFilesystem`   | `*bool`           | `true`  | Mount root filesystem as read-only. The PVC at `~/.openclaw/` and `/tmp` emptyDir provide writable paths. |
| `capabilities`             | `*Capabilities`   | Drop ALL | Linux capabilities to add or drop.                            |

#### spec.security.networkPolicy

| Field                      | Type                              | Default | Description                                                  |
|----------------------------|-----------------------------------|---------|--------------------------------------------------------------|
| `enabled`                  | `*bool`                           | `true`  | Create a NetworkPolicy. Warns if disabled.                   |
| `allowedIngressCIDRs`      | `[]string`                        | --      | CIDRs allowed to reach the instance.                         |
| `allowedIngressNamespaces` | `[]string`                        | --      | Namespaces allowed to reach the instance.                    |
| `allowedEgressCIDRs`       | `[]string`                        | --      | CIDRs the instance can reach (in addition to HTTPS/DNS).     |
| `allowDNS`                 | `*bool`                           | `true`  | Allow DNS resolution (UDP/TCP port 53).                      |
| `additionalEgress`         | `[]NetworkPolicyEgressRule`       | --      | Custom egress rules appended to the default DNS + HTTPS rules. Use this to allow traffic to cluster-internal services on non-standard ports. |

#### spec.security.rbac

| Field                        | Type                  | Default | Description                                                                              |
|------------------------------|-----------------------|---------|------------------------------------------------------------------------------------------|
| `createServiceAccount`       | `*bool`               | `true`  | Create a dedicated ServiceAccount for this instance.                                     |
| `serviceAccountName`         | `string`              | --      | Use an existing ServiceAccount (only when `createServiceAccount` is `false`).            |
| `serviceAccountAnnotations`  | `map[string]string`   | --      | Annotations to add to the managed ServiceAccount. Use for cloud provider integrations like AWS IRSA or GCP Workload Identity. |
| `additionalRules`            | `[]RBACRule`          | --      | Custom RBAC rules appended to the generated Role.                                        |

**RBACRule:**

| Field       | Type       | Description                                    |
|-------------|------------|------------------------------------------------|
| `apiGroups` | `[]string` | API groups (e.g., `[""]` for core, `["apps"]`).|
| `resources` | `[]string` | Resources (e.g., `["pods"]`).                  |
| `verbs`     | `[]string` | Verbs (e.g., `["get", "list"]`).               |

#### spec.security.caBundle

Injects a custom CA certificate bundle into all containers. Use this in environments with TLS-intercepting proxies or private CAs.

| Field            | Type     | Default         | Description                                                                              |
|------------------|----------|-----------------|------------------------------------------------------------------------------------------|
| `configMapName`  | `string` | --              | Name of a ConfigMap containing the CA bundle. The ConfigMap should have a key matching `key`. |
| `secretName`     | `string` | --              | Name of a Secret containing the CA bundle. Only one of `configMapName` or `secretName` should be set. |
| `key`            | `string` | `ca-bundle.crt` | Key in the ConfigMap or Secret containing the CA bundle PEM file.                        |

```yaml
spec:
  security:
    caBundle:
      configMapName: corporate-ca
      key: ca-bundle.crt
```

### spec.storage

Persistent storage configuration.

#### spec.storage.persistence

| Field           | Type                            | Default            | Description                                          |
|-----------------|---------------------------------|--------------------|------------------------------------------------------|
| `enabled`       | `*bool`                         | `true`             | Enable persistent storage via PVC.                   |
| `storageClass`  | `*string`                       | (cluster default)  | StorageClass name. Immutable after creation.         |
| `size`          | `string`                        | `10Gi`             | PVC size.                                            |
| `accessModes`   | `[]PersistentVolumeAccessMode`  | `[ReadWriteOnce]`  | PVC access modes.                                    |
| `existingClaim` | `string`                        | --                 | Name of an existing PVC to use instead of creating one. |

### spec.chromium

Optional Chromium sidecar for browser automation.

| Field                   | Type     | Default                          | Description                                         |
|-------------------------|----------|----------------------------------|-----------------------------------------------------|
| `enabled`               | `bool`   | `false`                          | Enable the Chromium sidecar container.              |
| `image.repository`      | `string` | `ghcr.io/browserless/chromium`   | Chromium container image repository.                |
| `image.tag`             | `string` | `latest`                         | Chromium image tag.                                 |
| `image.digest`          | `string` | --                               | Chromium image digest for supply chain security.    |
| `resources.requests.cpu`   | `string` | `250m`                        | Chromium minimum CPU.                               |
| `resources.requests.memory`| `string` | `512Mi`                       | Chromium minimum memory.                            |
| `resources.limits.cpu`     | `string` | `1000m`                       | Chromium maximum CPU.                               |
| `resources.limits.memory`  | `string` | `2Gi`                         | Chromium maximum memory.                            |

When enabled, the sidecar:

- Exposes Chrome DevTools Protocol on port 3000.
- Runs as UID 999 (blessuser).
- Mounts a memory-backed emptyDir at `/dev/shm` (1Gi) for shared memory.
- Mounts an emptyDir at `/tmp` for scratch space.

When Chromium is enabled, the operator also auto-configures browser profiles in the OpenClaw config. Both `"default"` and `"chrome"` profiles are set to point at the sidecar's CDP endpoint (`http://localhost:3000`). This ensures browser tool calls work regardless of which profile name the LLM passes.

### spec.tailscale

Optional Tailscale integration for secure tailnet access without Ingress or LoadBalancer.

| Field                | Type                     | Default          | Description                                                                |
|----------------------|--------------------------|------------------|----------------------------------------------------------------------------|
| `enabled`            | `bool`                   | `false`          | Enable Tailscale integration.                                              |
| `mode`               | `string`                 | `serve`          | Tailscale mode. `serve` exposes to tailnet members only. `funnel` exposes to the public internet via Tailscale Funnel. |
| `authKeySecretRef`   | `*LocalObjectReference`  | --               | Reference to a Secret containing the Tailscale auth key. Use ephemeral+reusable keys from the Tailscale admin console. |
| `authKeySecretKey`   | `string`                 | `authkey`        | Key in the referenced Secret containing the auth key.                      |
| `hostname`           | `string`                 | (instance name)  | Tailscale device name. Defaults to the OpenClawInstance name.              |
| `authSSO`            | `bool`                   | `false`          | Enable passwordless login for tailnet members. Sets `gateway.auth.allowTailscale=true` in the OpenClaw config. |

When enabled, the operator:

- Merges `gateway.tailscale` settings (mode, hostname) into the OpenClaw config.
- Injects the auth key from the referenced Secret.
- When `authSSO` is true, sets `gateway.auth.allowTailscale=true` so tailnet members can authenticate without a gateway token.

### spec.ollama

Optional Ollama sidecar for local LLM inference.

| Field                      | Type     | Default          | Description                                                                |
|----------------------------|----------|------------------|----------------------------------------------------------------------------|
| `enabled`                  | `bool`   | `false`          | Enable the Ollama sidecar container.                                       |
| `image.repository`         | `string` | `ollama/ollama`  | Ollama container image repository.                                         |
| `image.tag`                | `string` | `latest`         | Ollama image tag.                                                          |
| `image.digest`             | `string` | --               | Ollama image digest for supply chain security.                             |
| `models`                   | `[]string` | --             | Models to pre-pull during pod init (e.g., `["llama3.2", "nomic-embed-text"]`). Max 10 items. |
| `resources.requests.cpu`   | `string` | --               | Ollama minimum CPU.                                                        |
| `resources.requests.memory`| `string` | --               | Ollama minimum memory.                                                     |
| `resources.limits.cpu`     | `string` | --               | Ollama maximum CPU.                                                        |
| `resources.limits.memory`  | `string` | --               | Ollama maximum memory.                                                     |
| `storage.sizeLimit`        | `string` | `20Gi`           | Size limit for the emptyDir model cache volume.                            |
| `storage.existingClaim`    | `string` | --               | Name of an existing PVC for persistent model storage (overrides emptyDir). |
| `gpu`                      | `*int32` | --               | Number of NVIDIA GPUs to allocate (sets `nvidia.com/gpu` resource limit). Minimum: 0. |

When enabled, the operator:

- Adds an Ollama sidecar container to the pod.
- If `models` are specified, adds an init container (`init-ollama`) that pre-pulls the listed models so they are ready when the pod starts.
- The model cache uses an emptyDir by default (bounded by `storage.sizeLimit`). Set `storage.existingClaim` to use a PVC for persistent model storage across pod restarts.
- GPU allocation requires the NVIDIA device plugin to be installed on the cluster.

```yaml
spec:
  ollama:
    enabled: true
    models:
      - llama3.2
      - nomic-embed-text
    resources:
      requests:
        cpu: "2"
        memory: 4Gi
      limits:
        cpu: "8"
        memory: 16Gi
    storage:
      sizeLimit: 40Gi
    gpu: 1
```

### spec.webTerminal

Optional ttyd web terminal sidecar for browser-based shell access and debugging.

| Field                      | Type     | Default         | Description                                                                    |
|----------------------------|----------|-----------------|--------------------------------------------------------------------------------|
| `enabled`                  | `bool`   | `false`         | Enable the ttyd web terminal sidecar container.                                |
| `image.repository`         | `string` | `tsl0922/ttyd`  | Web terminal container image repository.                                       |
| `image.tag`                | `string` | `latest`        | Web terminal image tag.                                                        |
| `image.digest`             | `string` | --              | Web terminal image digest for supply chain security.                           |
| `resources.requests.cpu`   | `string` | `50m`           | Web terminal minimum CPU.                                                      |
| `resources.requests.memory`| `string` | `64Mi`          | Web terminal minimum memory.                                                   |
| `resources.limits.cpu`     | `string` | `200m`          | Web terminal maximum CPU.                                                      |
| `resources.limits.memory`  | `string` | `128Mi`         | Web terminal maximum memory.                                                   |
| `readOnly`                 | `bool`   | `false`         | Start ttyd in read-only mode (view-only, no input). Data volume mount is also set to read-only. |
| `credential.secretRef.name`| `string` | --              | Name of a Secret with `username` and `password` keys for basic auth.           |

When enabled, the operator:

- Adds a ttyd sidecar container on port 7681 to the pod.
- Mounts the instance data volume at `/home/openclaw/.openclaw` for inspection.
- Adds a `/tmp` emptyDir volume for the web terminal container.
- Adds the web terminal port to the Service and NetworkPolicy.
- Runs as UID 1000 (same as the main container) for shared file permissions on the data volume.

```yaml
spec:
  webTerminal:
    enabled: true
    readOnly: true
    credential:
      secretRef:
        name: terminal-credentials
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
```

### spec.initContainers

| Field            | Type            | Default | Description                                                              |
|------------------|-----------------|---------|--------------------------------------------------------------------------|
| `initContainers` | `[]Container`   | --      | Additional init containers to run before the main container. They run after the operator-managed init containers. Max 10 items. |

Standard Kubernetes `Container` spec. The following names are reserved by the operator and rejected by the webhook: `init-config`, `init-pnpm`, `init-python`, `init-skills`, `init-ollama`.

```yaml
spec:
  initContainers:
    - name: wait-for-db
      image: busybox:1.37
      command: ["sh", "-c", "until nc -z postgres.db.svc 5432; do sleep 2; done"]
    - name: seed-data
      image: my-seeder:latest
      volumeMounts:
        - name: data
          mountPath: /data
```

### spec.sidecars

| Field      | Type            | Default | Description                                                                                       |
|------------|-----------------|---------|---------------------------------------------------------------------------------------------------|
| `sidecars` | `[]Container`   | --      | Additional sidecar containers to inject into the pod. Use for custom sidecars like database proxies, log forwarders, or service meshes. |

Standard Kubernetes `Container` spec. Sidecar containers run alongside the main OpenClaw container.

```yaml
spec:
  sidecars:
    - name: cloud-sql-proxy
      image: gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.8.0
      args: ["--structured-logs", "my-project:us-central1:my-db"]
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
```

### spec.sidecarVolumes

| Field            | Type         | Default | Description                                                                |
|------------------|--------------|---------|----------------------------------------------------------------------------|
| `sidecarVolumes` | `[]Volume`   | --      | Additional volumes to make available to sidecar containers.                |

Standard Kubernetes `Volume` spec. Use this to mount ConfigMaps, Secrets, or other volumes that your custom sidecars need.

```yaml
spec:
  sidecarVolumes:
    - name: proxy-config
      configMap:
        name: cloud-sql-proxy-config
```

### spec.extraVolumes

| Field          | Type         | Default | Description                                                                       |
|----------------|--------------|---------|-----------------------------------------------------------------------------------|
| `extraVolumes` | `[]Volume`   | --      | Additional volumes to add to the pod. These volumes are available to the main container via `extraVolumeMounts`. Max 10 items. |

Standard Kubernetes `Volume` spec.

### spec.extraVolumeMounts

| Field               | Type              | Default | Description                                                                                       |
|---------------------|-------------------|---------|---------------------------------------------------------------------------------------------------|
| `extraVolumeMounts` | `[]VolumeMount`   | --      | Additional volume mounts to add to the main container. Use with `extraVolumes` to mount ConfigMaps, Secrets, NFS shares, or CSI volumes. Max 10 items. |

Standard Kubernetes `VolumeMount` spec.

```yaml
spec:
  extraVolumes:
    - name: shared-data
      nfs:
        server: nfs.example.com
        path: /exports/data
    - name: custom-certs
      secret:
        secretName: my-tls-certs
  extraVolumeMounts:
    - name: shared-data
      mountPath: /mnt/shared
      readOnly: true
    - name: custom-certs
      mountPath: /etc/custom-certs
      readOnly: true
```

### spec.networking

Network-related configuration for the instance.

#### spec.networking.service

| Field         | Type                  | Default      | Description                                               |
|---------------|-----------------------|--------------|-----------------------------------------------------------|
| `type`        | `string`              | `ClusterIP`  | Service type. One of: `ClusterIP`, `LoadBalancer`, `NodePort`. |
| `annotations` | `map[string]string`   | --           | Annotations to add to the Service.                        |
| `ports`       | `[]ServicePortSpec`   | --           | Custom ports exposed on the Service. When set, replaces the default gateway and canvas ports. |

**ServicePortSpec:**

| Field        | Type     | Default | Description                                        |
|--------------|----------|---------|----------------------------------------------------|
| `name`       | `string` | --      | Name of the port (required).                       |
| `port`       | `int32`  | --      | Port number exposed on the Service (required, 1-65535). |
| `targetPort` | `*int32` | `port`  | Port on the container to route to (defaults to `port`). |
| `protocol`   | `string` | `TCP`   | Protocol for the port. One of: `TCP`, `UDP`, `SCTP`. |

When `ports` is not set, the Service exposes these default ports:

| Port Name   | Port   | Description                     |
|-------------|--------|---------------------------------|
| `gateway`   | 18789  | OpenClaw WebSocket gateway.     |
| `canvas`    | 18793  | OpenClaw Canvas HTTP server.    |
| `chromium`  | 9222   | Chrome DevTools Protocol (only if Chromium sidecar is enabled). |

**Note:** Custom ports fully replace the defaults, including the Chromium port. If you use custom ports and have the Chromium sidecar enabled, include the Chromium port (9222) explicitly.

**Custom ports example:**

```yaml
networking:
  service:
    type: ClusterIP
    ports:
      - name: http
        port: 3978
        targetPort: 3978
```

#### spec.networking.ingress

| Field         | Type                | Default | Description                                         |
|---------------|---------------------|---------|-----------------------------------------------------|
| `enabled`     | `bool`              | `false` | Create an Ingress resource.                         |
| `className`   | `*string`           | --      | IngressClass to use (e.g., `nginx`, `traefik`).     |
| `annotations` | `map[string]string` | --      | Custom annotations added to the Ingress.            |
| `hosts`       | `[]IngressHost`     | --      | List of hosts to route traffic for.                 |
| `tls`         | `[]IngressTLS`      | --      | TLS termination configuration. Warns if empty.      |
| `security`    | `IngressSecuritySpec`| --     | Ingress security settings (HTTPS redirect, HSTS, rate limiting). |

**IngressHost:**

| Field   | Type            | Description                                 |
|---------|-----------------|---------------------------------------------|
| `host`  | `string`        | Fully qualified domain name.                |
| `paths` | `[]IngressPath` | Paths to route. Defaults to `[{path: "/"}]`.|

**IngressPath:**

| Field      | Type     | Default    | Description                                                              |
|------------|----------|------------|--------------------------------------------------------------------------|
| `path`     | `string` | `/`        | URL path.                                                                |
| `pathType` | `string` | `Prefix`   | Path matching. One of: `Prefix`, `Exact`, `ImplementationSpecific`.      |
| `port`     | `*int32` | `18789`    | Backend service port number. Defaults to the gateway port when not set.  |

**Custom backend port example:**

```yaml
networking:
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: aibot.example.com
        paths:
          - path: /api/messages
            pathType: Prefix
            port: 3978
    tls:
      - hosts:
          - aibot.example.com
        secretName: certificate-aibot-tls
```

**IngressTLS:**

| Field        | Type       | Description                                         |
|--------------|------------|-----------------------------------------------------|
| `hosts`      | `[]string` | Hostnames covered by the TLS certificate.           |
| `secretName` | `string`   | Secret containing the TLS key pair.                 |

**IngressSecuritySpec:**

| Field                       | Type     | Default | Description                                    |
|-----------------------------|----------|---------|------------------------------------------------|
| `forceHTTPS`                | `*bool`  | `true`  | Redirect HTTP to HTTPS.                        |
| `enableHSTS`                | `*bool`  | `true`  | Add HSTS headers.                              |
| `rateLimiting.enabled`      | `*bool`  | `true`  | Enable rate limiting.                          |
| `rateLimiting.requestsPerSecond` | `*int32` | `10` | Maximum requests per second.              |

The operator automatically adds WebSocket-related annotations for nginx-ingress (proxy timeouts, HTTP/1.1 upgrade).

### spec.probes

Health probe configuration for the main OpenClaw container. All probes use TCP socket checks against the gateway port (18789).

#### spec.probes.liveness

| Field                 | Type     | Default | Description                                           |
|-----------------------|----------|---------|-------------------------------------------------------|
| `enabled`             | `*bool`  | `true`  | Enable the liveness probe.                            |
| `initialDelaySeconds` | `*int32` | `30`    | Seconds to wait before the first check.               |
| `periodSeconds`       | `*int32` | `10`    | Seconds between checks.                              |
| `timeoutSeconds`      | `*int32` | `5`     | Seconds before the check times out.                  |
| `failureThreshold`    | `*int32` | `3`     | Consecutive failures before restarting the container. |

#### spec.probes.readiness

| Field                 | Type     | Default | Description                                           |
|-----------------------|----------|---------|-------------------------------------------------------|
| `enabled`             | `*bool`  | `true`  | Enable the readiness probe.                           |
| `initialDelaySeconds` | `*int32` | `5`     | Seconds to wait before the first check.               |
| `periodSeconds`       | `*int32` | `5`     | Seconds between checks.                              |
| `timeoutSeconds`      | `*int32` | `3`     | Seconds before the check times out.                  |
| `failureThreshold`    | `*int32` | `3`     | Consecutive failures before removing from endpoints. |

#### spec.probes.startup

| Field                 | Type     | Default | Description                                           |
|-----------------------|----------|---------|-------------------------------------------------------|
| `enabled`             | `*bool`  | `true`  | Enable the startup probe.                             |
| `initialDelaySeconds` | `*int32` | `0`     | Seconds to wait before the first check.               |
| `periodSeconds`       | `*int32` | `5`     | Seconds between checks.                              |
| `timeoutSeconds`      | `*int32` | `3`     | Seconds before the check times out.                  |
| `failureThreshold`    | `*int32` | `30`    | Consecutive failures before killing the container. Allows up to 150s startup. |

### spec.observability

Metrics and logging configuration.

#### spec.observability.metrics

| Field                       | Type                | Default | Description                                   |
|-----------------------------|---------------------|---------|-----------------------------------------------|
| `enabled`                   | `*bool`             | `true`  | Enable the metrics endpoint on the managed instance. |
| `port`                      | `*int32`            | `9090`  | Metrics port.                                 |
| `serviceMonitor.enabled`    | `*bool`             | `false` | Create a Prometheus `ServiceMonitor`.         |
| `serviceMonitor.interval`   | `string`            | `30s`   | Prometheus scrape interval.                   |
| `serviceMonitor.labels`     | `map[string]string` | --      | Labels to add to the ServiceMonitor (for Prometheus selector matching). |
| `prometheusRule.enabled`    | `*bool`             | `false` | Create a `PrometheusRule` with operator alerts. |
| `prometheusRule.labels`     | `map[string]string` | --      | Labels to add to the PrometheusRule (for Prometheus rule selector matching). |
| `prometheusRule.runbookBaseURL` | `string`         | `https://openclaw.rocks/docs/runbooks` | Base URL for alert runbook links. |
| `grafanaDashboard.enabled`  | `*bool`             | `false` | Create Grafana dashboard ConfigMaps (operator overview + instance detail). |
| `grafanaDashboard.labels`   | `map[string]string` | --      | Extra labels to add to dashboard ConfigMaps. |
| `grafanaDashboard.folder`   | `string`            | `OpenClaw` | Grafana folder for the dashboards. |

#### spec.observability.logging

| Field    | Type     | Default | Description                                              |
|----------|----------|---------|----------------------------------------------------------|
| `level`  | `string` | `info`  | Log level. One of: `debug`, `info`, `warn`, `error`.     |
| `format` | `string` | `json`  | Log format. One of: `json`, `text`.                      |

### spec.selfConfigure

Agent self-modification configuration. When enabled, the agent can create `OpenClawSelfConfig` resources to modify its own instance spec via the K8s API.

| Field            | Type                 | Default | Description                                                                     |
|------------------|----------------------|---------|---------------------------------------------------------------------------------|
| `enabled`        | `bool`               | `false` | Enable self-configuration for this instance.                                    |
| `allowedActions` | `[]SelfConfigAction` | --      | Action categories the agent is allowed to perform. If empty, no actions pass validation (fail-safe). Max 4 items. |

**SelfConfigAction values:**

| Value            | Description                                                       |
|------------------|-------------------------------------------------------------------|
| `skills`         | Add or remove skills (`spec.skills`).                             |
| `config`         | Deep-merge patches into the OpenClaw config (`spec.config.raw`).  |
| `workspaceFiles` | Add or remove initial workspace files (`spec.workspace.initialFiles`). |
| `envVars`        | Add or remove plain environment variables (`spec.env`).           |

When enabled, the operator:
- Grants the SA read access to its own `OpenClawInstance` and referenced Secrets (scoped by `resourceNames`)
- Grants `create`, `get`, `list` on `openclawselfconfigs`
- Sets `automountServiceAccountToken: true` on the SA and pod spec
- Injects `OPENCLAW_INSTANCE_NAME` and `OPENCLAW_NAMESPACE` environment variables
- Adds port 6443 egress to the NetworkPolicy for K8s API access
- Injects `SELFCONFIG.md` (skill documentation) and `selfconfig.sh` (helper script) into the workspace

### spec.availability

High availability and scheduling configuration.

| Field                             | Type                | Default | Description                                              |
|-----------------------------------|---------------------|---------|----------------------------------------------------------|
| `podDisruptionBudget.enabled`     | `*bool`             | `true`  | Create a PodDisruptionBudget.                            |
| `podDisruptionBudget.maxUnavailable` | `*int32`         | `1`     | Maximum pods that can be unavailable during disruption.  |
| `nodeSelector`                    | `map[string]string` | --      | Node labels for pod scheduling.                          |
| `tolerations`                     | `[]Toleration`      | --      | Tolerations for pod scheduling.                          |
| `affinity`                        | `*Affinity`         | --      | Affinity and anti-affinity rules.                        |
| `autoScaling.enabled`             | `*bool`             | `false` | Create a HorizontalPodAutoscaler.                        |
| `autoScaling.minReplicas`         | `*int32`            | `1`     | Minimum number of replicas.                              |
| `autoScaling.maxReplicas`         | `*int32`            | `5`     | Maximum number of replicas.                              |
| `autoScaling.targetCPUUtilization` | `*int32`           | `80`    | Target average CPU utilization (percentage).             |
| `autoScaling.targetMemoryUtilization` | `*int32`        | --      | Target average memory utilization (percentage).          |

### spec.restoreFrom

| Field         | Type     | Default | Description                                                                                       |
|---------------|----------|---------|---------------------------------------------------------------------------------------------------|
| `restoreFrom` | `string` | --      | Remote backup path to restore data from (e.g., `backups/{tenantId}/{instanceId}/{timestamp}`). When set, the operator restores PVC data from this S3 path before creating the StatefulSet. Cleared automatically after successful restore. |

### spec.runtimeDeps

Configures built-in init containers that install runtime dependencies to the data PVC for use by MCP servers and skills.

| Field    | Type   | Default | Description                                                                              |
|----------|--------|---------|------------------------------------------------------------------------------------------|
| `pnpm`   | `bool` | `false` | Install pnpm via corepack for npm-based MCP servers and skills. Adds the `init-pnpm` init container. |
| `python` | `bool` | `false` | Install Python 3.12 and uv for Python-based MCP servers and skills. Adds the `init-python` init container. |

```yaml
spec:
  runtimeDeps:
    pnpm: true
    python: true
```

### spec.gateway

Configures the gateway authentication token.

| Field            | Type     | Default | Description                                                                                       |
|------------------|----------|---------|---------------------------------------------------------------------------------------------------|
| `existingSecret` | `string` | --      | Name of a user-managed Secret containing the gateway token. The Secret must have a key named `token`. When set, the operator skips auto-generating a gateway token Secret and uses this Secret instead. |

When `existingSecret` is not set, the operator automatically generates a random gateway token Secret, which is tracked in `status.managedResources.gatewayTokenSecret`.

```yaml
spec:
  gateway:
    existingSecret: my-gateway-token
```

### spec.autoUpdate

Configures automatic version updates from the OCI registry.

| Field                | Type     | Default | Description                                                                                       |
|----------------------|----------|---------|---------------------------------------------------------------------------------------------------|
| `enabled`            | `*bool`  | `false` | Enable automatic version updates.                                                                 |
| `checkInterval`      | `string` | `24h`   | How often to check for new versions (Go duration). Minimum: `1h`, Maximum: `168h` (7 days).       |
| `backupBeforeUpdate` | `*bool`  | `true`  | Create a backup before applying updates.                                                          |
| `rollbackOnFailure`  | `*bool`  | `true`  | Automatically revert to the previous version if the updated pod fails to become ready within `healthCheckTimeout`. |
| `healthCheckTimeout` | `string` | `10m`   | How long to wait for the updated pod to become ready before triggering a rollback (Go duration). Minimum: `2m`, Maximum: `30m`. |

When enabled, the operator periodically checks the OCI registry for newer image tags. If a new version is found, it optionally creates a backup, updates the StatefulSet image tag, and monitors the rollout. If the pod fails to become ready within the health check timeout, the operator automatically rolls back to the previous version (if `rollbackOnFailure` is enabled).

Auto-update pauses after 3 consecutive rollbacks. It resumes when a newer version becomes available.

```yaml
spec:
  autoUpdate:
    enabled: true
    checkInterval: 12h
    backupBeforeUpdate: true
    rollbackOnFailure: true
    healthCheckTimeout: 15m
```

---

## Status Fields

### status.phase

| Field   | Type     | Description                                                                    |
|---------|----------|--------------------------------------------------------------------------------|
| `phase` | `string` | Current lifecycle phase: `Pending`, `Provisioning`, `Running`, `Degraded`, `Failed`, `Terminating`, `BackingUp`, `Restoring`, `Updating`. |

### status.conditions

Standard `metav1.Condition` array. Condition types:

| Type                  | Description                                                    |
|-----------------------|----------------------------------------------------------------|
| `Ready`               | Overall readiness of the instance.                             |
| `ConfigValid`         | Configuration is valid and loaded.                             |
| `StatefulSetReady`    | StatefulSet has ready replicas.                                |
| `DeploymentReady`     | **(Deprecated)** Legacy Deployment has ready replicas. Used during migration from Deployment to StatefulSet. |
| `ServiceReady`        | Service has been created.                                      |
| `NetworkPolicyReady`  | NetworkPolicy has been applied.                                |
| `RBACReady`           | RBAC resources are in place.                                   |
| `StorageReady`        | PVC has been provisioned and is bound.                         |
| `BackupComplete`      | The backup job completed successfully.                         |
| `RestoreComplete`     | The restore job completed successfully.                        |
| `AutoUpdateAvailable` | A newer version is available in the OCI registry.              |
| `SecretsReady`        | All referenced Secrets exist and are accessible.               |

### status.endpoints

| Field              | Type     | Description                                                  |
|--------------------|----------|--------------------------------------------------------------|
| `gatewayEndpoint`  | `string` | In-cluster endpoint for the gateway: `<name>.<ns>.svc:18789`.|
| `canvasEndpoint`   | `string` | In-cluster endpoint for canvas: `<name>.<ns>.svc:18793`.     |

### status.observedGeneration

| Field                | Type    | Description                                              |
|----------------------|---------|----------------------------------------------------------|
| `observedGeneration` | `int64` | The `.metadata.generation` last processed by the controller. |

### status.lastReconcileTime

| Field               | Type          | Description                                     |
|---------------------|---------------|-------------------------------------------------|
| `lastReconcileTime` | `*metav1.Time`| Timestamp of the last successful reconciliation.|

### status.managedResources

| Field                | Type     | Description                           |
|----------------------|----------|---------------------------------------|
| `statefulSet`        | `string` | Name of the managed StatefulSet.      |
| `deployment`         | `string` | Name of the legacy Deployment (deprecated, used during migration). |
| `service`            | `string` | Name of the managed Service.          |
| `configMap`          | `string` | Name of the managed ConfigMap.        |
| `pvc`                | `string` | Name of the managed PVC.             |
| `networkPolicy`      | `string` | Name of the managed NetworkPolicy.    |
| `podDisruptionBudget`| `string` | Name of the managed PDB.             |
| `serviceAccount`     | `string` | Name of the managed ServiceAccount.   |
| `role`               | `string` | Name of the managed Role.            |
| `roleBinding`        | `string` | Name of the managed RoleBinding.      |
| `gatewayTokenSecret` | `string` | Name of the auto-generated gateway token Secret. |
| `prometheusRule`     | `string` | Name of the managed PrometheusRule. |
| `grafanaDashboardOperator` | `string` | Name of the operator overview dashboard ConfigMap. |
| `grafanaDashboardInstance` | `string` | Name of the instance detail dashboard ConfigMap. |
| `horizontalPodAutoscaler` | `string` | Name of the managed HorizontalPodAutoscaler. |

### status.backup and restore

| Field            | Type           | Description                                              |
|------------------|----------------|----------------------------------------------------------|
| `backupJobName`  | `string`       | Name of the active backup Job.                           |
| `restoreJobName` | `string`       | Name of the active restore Job.                          |
| `lastBackupPath` | `string`       | S3 path of the last successful backup.                   |
| `lastBackupTime` | `*metav1.Time` | Timestamp of the last successful backup.                 |
| `restoredFrom`   | `string`       | S3 path this instance was restored from.                 |

### status.autoUpdate

Tracks the state of automatic version updates.

| Field                | Type           | Description                                                                              |
|----------------------|----------------|------------------------------------------------------------------------------------------|
| `lastCheckTime`      | `*metav1.Time` | When the registry was last checked for new versions.                                     |
| `latestVersion`      | `string`       | Latest version available in the registry.                                                |
| `currentVersion`     | `string`       | Version currently running.                                                               |
| `pendingVersion`     | `string`       | Set during an in-flight update to the version being applied.                             |
| `updatePhase`        | `string`       | Progress of an in-flight update. One of: `""`, `BackingUp`, `ApplyingUpdate`, `HealthCheck`, `RollingBack`. |
| `lastUpdateTime`     | `*metav1.Time` | When the last successful update was applied.                                             |
| `lastUpdateError`    | `string`       | Error message from the last failed update attempt.                                       |
| `previousVersion`    | `string`       | Version before the last update (used for rollback).                                      |
| `preUpdateBackupPath`| `string`       | S3 path of the pre-update backup (used for rollback restore).                            |
| `failedVersion`      | `string`       | Version that failed health checks and will be skipped in future checks. Cleared when a newer version becomes available. |
| `rollbackCount`      | `int32`        | Consecutive rollback count. Auto-update pauses after 3. Reset to 0 on any successful update. |

---

## Related Guides

- [Model Fallback Chains](model-fallback.md) - configure multi-provider fallback with `llmConfig`
- [Custom AI Providers](custom-providers.md) - Ollama sidecar, vLLM, and other self-hosted models
- [External Secrets Operator Integration](external-secrets.md) - sync API keys from AWS, Vault, GCP, etc.

---

## OpenClawSelfConfig (v1alpha1)

**Group**: `openclaw.rocks`
**Version**: `v1alpha1`
**Kind**: `OpenClawSelfConfig`
**Scope**: Namespaced
**Short name**: `ocsc`

An `OpenClawSelfConfig` represents a request from an agent to modify its own `OpenClawInstance` spec. The operator validates the request against the parent instance's `selfConfigure.allowedActions` policy and applies approved changes.

### Print Columns

| Column    | JSON Path                          |
|-----------|------------------------------------|
| Instance  | `.spec.instanceRef`                |
| Phase     | `.status.phase`                    |
| Age       | `.metadata.creationTimestamp`      |

### Spec Fields

| Field                | Type                      | Default | Description                                                                   |
|----------------------|---------------------------|---------|-------------------------------------------------------------------------------|
| `instanceRef`        | `string`                  | (required) | Name of the parent `OpenClawInstance` in the same namespace.               |
| `addSkills`          | `[]string`                | --      | Skills to add. Max 10 items.                                                  |
| `removeSkills`       | `[]string`                | --      | Skills to remove. Max 10 items.                                               |
| `configPatch`        | `RawConfig`               | --      | Partial JSON to deep-merge into `spec.config.raw`. Protected keys under `gateway` are rejected. |
| `addWorkspaceFiles`  | `map[string]string`       | --      | Filenames to content to add to workspace. Max 10 entries.                     |
| `removeWorkspaceFiles` | `[]string`              | --      | Workspace filenames to remove. Max 10 items.                                  |
| `addEnvVars`         | `[]SelfConfigEnvVar`      | --      | Environment variables to add (plain values only, no secret refs). Max 10 items. |
| `removeEnvVars`      | `[]string`                | --      | Environment variable names to remove. Max 10 items.                           |

**SelfConfigEnvVar:**

| Field   | Type     | Description                   |
|---------|----------|-------------------------------|
| `name`  | `string` | Environment variable name.    |
| `value` | `string` | Environment variable value.   |

### Status Fields

| Field            | Type          | Description                                                  |
|------------------|---------------|--------------------------------------------------------------|
| `phase`          | `string`      | Processing state: `Pending`, `Applied`, `Failed`, `Denied`.  |
| `message`        | `string`      | Human-readable details about the current phase.              |
| `completionTime` | `*metav1.Time`| Timestamp when the request reached a terminal phase.         |

### Lifecycle

1. Agent creates an `OpenClawSelfConfig` resource -- status starts as `Pending`
2. Operator fetches the parent `OpenClawInstance` and validates:
   - `selfConfigure.enabled` must be `true` (otherwise: `Denied`)
   - All requested action categories must be in `allowedActions` (otherwise: `Denied`)
   - Protected config keys (`gateway.*`) and env var names are rejected (otherwise: `Failed`)
3. Operator applies changes to the parent instance spec
4. Status transitions to `Applied` (success) or `Failed` (error)
5. An owner reference is set to the parent instance for garbage collection
6. Terminal requests are auto-deleted after 1 hour

### Protected Resources

The following are protected and cannot be modified via self-configure:

- **Config keys**: `gateway` (entire subtree) -- prevents breaking gateway auth
- **Environment variables**: `HOME`, `PATH`, `OPENCLAW_GATEWAY_TOKEN`, `OPENCLAW_INSTANCE_NAME`, `OPENCLAW_NAMESPACE`, `OPENCLAW_DISABLE_BONJOUR`, `CHROMIUM_URL`, `OLLAMA_HOST`, `TS_AUTHKEY`, `TS_HOSTNAME`, `NODE_EXTRA_CA_CERTS`, `NPM_CONFIG_CACHE`, `NPM_CONFIG_IGNORE_SCRIPTS`

### Example

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawSelfConfig
metadata:
  name: add-fetch-skill
spec:
  instanceRef: my-agent
  addSkills:
    - "@anthropic/mcp-server-fetch"
  addEnvVars:
    - name: MY_CUSTOM_VAR
      value: "hello"
```

---

## OpenClawSelfConfig (v1alpha1)

**Group**: `openclaw.rocks`
**Version**: `v1alpha1`
**Kind**: `OpenClawSelfConfig`
**Scope**: Namespaced
**Short name**: `ocsc`

An `OpenClawSelfConfig` represents a request from an agent to modify its own `OpenClawInstance` spec. The operator validates the request against the parent instance's `selfConfigure.allowedActions` policy and applies approved changes.

### Print Columns

| Column    | JSON Path                          |
|-----------|------------------------------------|
| Instance  | `.spec.instanceRef`                |
| Phase     | `.status.phase`                    |
| Age       | `.metadata.creationTimestamp`      |

### Spec Fields

| Field                | Type                      | Default | Description                                                                   |
|----------------------|---------------------------|---------|-------------------------------------------------------------------------------|
| `instanceRef`        | `string`                  | (required) | Name of the parent `OpenClawInstance` in the same namespace.               |
| `addSkills`          | `[]string`                | --      | Skills to add. Max 10 items.                                                  |
| `removeSkills`       | `[]string`                | --      | Skills to remove. Max 10 items.                                               |
| `configPatch`        | `RawConfig`               | --      | Partial JSON to deep-merge into `spec.config.raw`. Protected keys under `gateway` are rejected. |
| `addWorkspaceFiles`  | `map[string]string`       | --      | Filenames to content to add to workspace. Max 10 entries.                     |
| `removeWorkspaceFiles` | `[]string`              | --      | Workspace filenames to remove. Max 10 items.                                  |
| `addEnvVars`         | `[]SelfConfigEnvVar`      | --      | Environment variables to add (plain values only, no secret refs). Max 10 items. |
| `removeEnvVars`      | `[]string`                | --      | Environment variable names to remove. Max 10 items.                           |

**SelfConfigEnvVar:**

| Field   | Type     | Description                   |
|---------|----------|-------------------------------|
| `name`  | `string` | Environment variable name.    |
| `value` | `string` | Environment variable value.   |

### Status Fields

| Field            | Type          | Description                                                  |
|------------------|---------------|--------------------------------------------------------------|
| `phase`          | `string`      | Processing state: `Pending`, `Applied`, `Failed`, `Denied`.  |
| `message`        | `string`      | Human-readable details about the current phase.              |
| `completionTime` | `*metav1.Time`| Timestamp when the request reached a terminal phase.         |

### Lifecycle

1. Agent creates an `OpenClawSelfConfig` resource -- status starts as `Pending`
2. Operator fetches the parent `OpenClawInstance` and validates:
   - `selfConfigure.enabled` must be `true` (otherwise: `Denied`)
   - All requested action categories must be in `allowedActions` (otherwise: `Denied`)
   - Protected config keys (`gateway.*`) and env var names are rejected (otherwise: `Failed`)
3. Operator applies changes to the parent instance spec
4. Status transitions to `Applied` (success) or `Failed` (error)
5. An owner reference is set to the parent instance for garbage collection
6. Terminal requests are auto-deleted after 1 hour

### Protected Resources

The following are protected and cannot be modified via self-configure:

- **Config keys**: `gateway` (entire subtree) -- prevents breaking gateway auth
- **Environment variables**: `HOME`, `PATH`, `OPENCLAW_GATEWAY_TOKEN`, `OPENCLAW_INSTANCE_NAME`, `OPENCLAW_NAMESPACE`, `OPENCLAW_DISABLE_BONJOUR`, `CHROMIUM_URL`, `OLLAMA_HOST`, `TS_AUTHKEY`, `TS_HOSTNAME`, `NODE_EXTRA_CA_CERTS`, `NPM_CONFIG_CACHE`, `NPM_CONFIG_IGNORE_SCRIPTS`

### Example

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawSelfConfig
metadata:
  name: add-fetch-skill
spec:
  instanceRef: my-agent
  addSkills:
    - "@anthropic/mcp-server-fetch"
  addEnvVars:
    - name: MY_CUSTOM_VAR
      value: "hello"
```

---

## Full Example

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-assistant
  namespace: openclaw
spec:
  image:
    repository: ghcr.io/openclaw/openclaw
    tag: "0.5.0"
    pullPolicy: IfNotPresent
    pullSecrets:
      - name: ghcr-secret

  config:
    mergeMode: merge
    raw:
      mcpServers:
        filesystem:
          command: npx
          args: ["-y", "@modelcontextprotocol/server-filesystem", "/data"]

  workspace:
    initialDirectories:
      - tools/scripts
    initialFiles:
      CLAUDE.md: |
        # Project Instructions
        Use the filesystem MCP server for file operations.

  skills:
    - "@anthropic/mcp-server-fetch"
    - "npm:@openclaw/matrix"

  envFrom:
    - secretRef:
        name: openclaw-api-keys

  selfConfigure:
    enabled: true
    allowedActions:
      - skills
      - config

  resources:
    requests:
      cpu: "1"
      memory: 2Gi
    limits:
      cpu: "4"
      memory: 8Gi

  security:
    podSecurityContext:
      runAsUser: 1000
      runAsGroup: 1000
      fsGroup: 1000
      fsGroupChangePolicy: OnRootMismatch
      runAsNonRoot: true
    containerSecurityContext:
      allowPrivilegeEscalation: false
    networkPolicy:
      enabled: true
      allowedIngressNamespaces:
        - monitoring
      allowDNS: true
      additionalEgress:
        - to:
            - namespaceSelector:
                matchLabels:
                  app: postgres
          ports:
            - protocol: TCP
              port: 5432
    rbac:
      createServiceAccount: true
      serviceAccountAnnotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/openclaw-role
    caBundle:
      configMapName: corporate-ca
      key: ca-bundle.crt

  storage:
    persistence:
      enabled: true
      storageClass: gp3
      size: 50Gi
      accessModes:
        - ReadWriteOnce

  chromium:
    enabled: true
    image:
      repository: ghcr.io/browserless/chromium
      tag: "v2.0.0"
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: "2"
        memory: 4Gi

  ollama:
    enabled: true
    models:
      - llama3.2
      - nomic-embed-text
    resources:
      requests:
        cpu: "2"
        memory: 4Gi
      limits:
        cpu: "8"
        memory: 16Gi
    storage:
      sizeLimit: 40Gi
    gpu: 1

  networking:
    service:
      type: ClusterIP
    ingress:
      enabled: true
      className: nginx
      hosts:
        - host: openclaw.example.com
          paths:
            - path: /
              pathType: Prefix
      tls:
        - hosts:
            - openclaw.example.com
          secretName: openclaw-tls
      security:
        forceHTTPS: true
        enableHSTS: true
        rateLimiting:
          enabled: true
          requestsPerSecond: 20

  probes:
    liveness:
      enabled: true
      initialDelaySeconds: 60
      periodSeconds: 15
    readiness:
      enabled: true
      initialDelaySeconds: 10
    startup:
      enabled: true
      failureThreshold: 60

  observability:
    metrics:
      enabled: true
      serviceMonitor:
        enabled: true
        interval: 15s
        labels:
          release: prometheus
    logging:
      level: info
      format: json

  availability:
    podDisruptionBudget:
      enabled: true
      maxUnavailable: 1
    nodeSelector:
      node-type: compute
    tolerations:
      - key: dedicated
        operator: Equal
        value: openclaw
        effect: NoSchedule
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: openclaw

  runtimeDeps:
    pnpm: true
    python: true

  gateway:
    existingSecret: my-gateway-token

  autoUpdate:
    enabled: true
    checkInterval: 12h
    backupBeforeUpdate: true
    rollbackOnFailure: true
    healthCheckTimeout: 15m
```
