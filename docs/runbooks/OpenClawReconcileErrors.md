# OpenClawReconcileErrors

## Meaning

The operator is failing to reconcile an OpenClawInstance. The `openclaw_reconcile_total{result="error"}` counter is increasing.

## Impact

Managed resources (StatefulSet, Service, ConfigMap, etc.) may be out of sync with the desired state defined in the CR. The instance may not be updated or may have stale configuration.

## Diagnosis

```bash
# Check operator logs for errors
kubectl logs -n openclaw-operator-system deploy/openclaw-operator-controller-manager -c manager --tail=100

# Check the instance status and conditions
kubectl get openclawinstance <name> -n <namespace> -o yaml

# Check events on the instance
kubectl describe openclawinstance <name> -n <namespace>

# Check if dependent resources exist (Secrets, ConfigMaps)
kubectl get secrets,configmaps -n <namespace>
```

## Mitigation

1. **Missing secrets** - Ensure all secrets referenced in `spec.envFrom` exist in the namespace
2. **RBAC issues** - Verify the operator has the required ClusterRole permissions
3. **Resource conflicts** - Check if another controller or process is modifying the same resources
4. **API server issues** - Check kube-apiserver health
5. If the error is transient, the operator will retry with exponential backoff
