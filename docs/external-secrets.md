# External Secrets Operator (ESO) Integration

The [External Secrets Operator](https://external-secrets.io/) syncs secrets from external providers (AWS Secrets Manager, Vault, GCP Secret Manager, etc.) into Kubernetes Secrets. OpenClaw consumes these secrets via `envFrom`.

## Architecture

```
External Provider  →  ExternalSecret CR  →  Kubernetes Secret  →  OpenClaw Pod (envFrom)
```

The OpenClaw operator does not manage secrets directly — it only references them. ESO handles the lifecycle of the Kubernetes Secret, including rotation.

## Example: AWS Secrets Manager

### 1. Create a SecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets
  namespace: openclaw
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: eso-service-account
```

### 2. Create an ExternalSecret

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: openclaw-api-keys
  namespace: openclaw
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets
    kind: SecretStore
  target:
    name: openclaw-api-keys
    creationPolicy: Owner
  data:
    - secretKey: ANTHROPIC_API_KEY
      remoteRef:
        key: openclaw/api-keys
        property: anthropic
    - secretKey: OPENAI_API_KEY
      remoteRef:
        key: openclaw/api-keys
        property: openai
```

### 3. Reference in OpenClawInstance

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: my-assistant
  namespace: openclaw
spec:
  envFrom:
    - secretRef:
        name: openclaw-api-keys

  security:
    rbac:
      createServiceAccount: true
      serviceAccountAnnotations:
        eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/openclaw-role"
```

## Secret Rotation

When ESO refreshes a secret, the Kubernetes Secret's `resourceVersion` changes. The OpenClaw operator detects this change and triggers a rolling restart of the pod to pick up the new values.

This works because the operator watches Secrets referenced in `envFrom` and `env[].valueFrom.secretKeyRef`. No manual intervention is needed.

### Rotation Flow

1. ESO refreshes the secret based on `refreshInterval`
2. Kubernetes Secret is updated (new `resourceVersion`)
3. OpenClaw operator detects the change via its watch
4. Operator updates the config hash annotation on the pod template
5. StatefulSet controller rolls the pod with the new secret values

## Cloud Provider Authentication

ESO and OpenClaw may both need cloud provider credentials. Use `serviceAccountAnnotations` to configure workload identity:

### AWS (IRSA)

```yaml
spec:
  security:
    rbac:
      serviceAccountAnnotations:
        eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/openclaw-role"
```

### GCP (Workload Identity)

```yaml
spec:
  security:
    rbac:
      serviceAccountAnnotations:
        iam.gke.io/gcp-service-account: "openclaw@my-project.iam.gserviceaccount.com"
```

### Azure (Workload Identity)

```yaml
spec:
  security:
    rbac:
      serviceAccountAnnotations:
        azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
```

## Supported ESO Backends

| Backend                | SecretStore Provider | Notes                                    |
|------------------------|----------------------|------------------------------------------|
| AWS Secrets Manager    | `aws`                | Use IRSA for authentication              |
| AWS Parameter Store    | `aws`                | Set `service: ParameterStore`            |
| HashiCorp Vault        | `vault`              | Supports KV v1/v2, PKI, transit          |
| GCP Secret Manager     | `gcpsm`              | Use Workload Identity                    |
| Azure Key Vault        | `azurekv`            | Use Workload Identity                    |
| 1Password              | `onepassword`        | Via 1Password Connect server             |
| Doppler               | `doppler`            | Direct integration                       |
| Kubernetes (cross-ns)  | `kubernetes`         | Copy secrets across namespaces           |

## Troubleshooting

**Secret not syncing:**
```bash
kubectl get externalsecret openclaw-api-keys -n openclaw
kubectl describe externalsecret openclaw-api-keys -n openclaw
```

**Pod not restarting after rotation:**
Check that the secret name in `envFrom` matches the ExternalSecret's `target.name`.

**IRSA not working:**
Verify the ServiceAccount has the correct annotation and the IAM role trust policy allows the service account.
