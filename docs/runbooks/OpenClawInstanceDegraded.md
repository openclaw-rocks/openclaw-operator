# OpenClawInstanceDegraded

## Meaning

An OpenClaw instance is in `Failed` or `Degraded` phase for more than 5 minutes.

## Impact

The instance is not fully operational. Users may be unable to access the OpenClaw gateway or canvas.

## Diagnosis

```bash
# Check instance phase and conditions
kubectl get openclawinstance <name> -n <namespace> -o yaml

# Check pod status
kubectl get pods -n <namespace> -l app.kubernetes.io/instance=<name>

# Check pod events
kubectl describe pod <name>-0 -n <namespace>

# Check container logs
kubectl logs <name>-0 -n <namespace> -c openclaw --tail=100

# Check if the StatefulSet is progressing
kubectl get statefulset <name> -n <namespace> -o yaml
```

## Mitigation

1. **Pod not starting** - Check image pull errors, resource limits, node capacity
2. **Readiness probe failing** - The OpenClaw process may not be healthy; check container logs
3. **Configuration error** - Verify the ConfigMap content is valid JSON
4. **PVC issues** - Check if the PVC is bound and has sufficient space
5. **Missing API keys** - Ensure required secrets (e.g., ANTHROPIC_API_KEY) are present
