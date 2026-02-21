# OpenClawSlowReconciliation

## Meaning

The p99 reconciliation duration exceeds 30 seconds. Reconciliation should normally complete in under 5 seconds.

## Impact

Slow reconciliation delays the propagation of spec changes to managed resources. During high load, the controller may fall behind on processing events.

## Diagnosis

```bash
# Check operator resource usage
kubectl top pod -n openclaw-operator-system

# Check operator logs for slow operations
kubectl logs -n openclaw-operator-system deploy/openclaw-operator-controller-manager -c manager --tail=200

# Check API server latency
kubectl get --raw /metrics | grep apiserver_request_duration

# Check number of managed instances
kubectl get openclawinstance --all-namespaces --no-headers | wc -l

# Check workqueue depth
kubectl get --raw /metrics | grep workqueue_depth
```

## Mitigation

1. **API server overload** - Check kube-apiserver health and latency
2. **Too many instances** - Consider running multiple operator replicas with leader election
3. **Network issues** - Check connectivity between operator pod and API server
4. **Resource starvation** - Increase operator CPU/memory limits
