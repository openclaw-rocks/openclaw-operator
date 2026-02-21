# OpenClawAutoUpdateRollback

## Meaning

An automatic version update was applied but the updated pod failed health checks, triggering an automatic rollback to the previous version.

## Impact

The instance is running on the previous version. The failed version is recorded in `status.autoUpdate.failedVersion` and will be skipped in future checks until a newer version is released.

## Diagnosis

```bash
# Check auto-update status
kubectl get openclawinstance <name> -n <namespace> -o jsonpath='{.status.autoUpdate}' | jq .

# Check rollback count
kubectl get openclawinstance <name> -n <namespace> -o jsonpath='{.status.autoUpdate.rollbackCount}'

# Check the failed version
kubectl get openclawinstance <name> -n <namespace> -o jsonpath='{.status.autoUpdate.failedVersion}'

# Check events for details
kubectl describe openclawinstance <name> -n <namespace> | grep -A5 AutoUpdate

# Check pod logs from the failed version (if still available)
kubectl logs <name>-0 -n <namespace> -c openclaw --previous --tail=100
```

## Mitigation

1. **Wait for fix** - The failed version is automatically skipped; the next release will be tried
2. **Investigate failure** - Check pod logs to understand why the new version failed health checks
3. **Disable auto-update** - Set `spec.autoUpdate.enabled: false` to stop automatic updates
4. **Increase health check timeout** - If the new version needs more startup time, increase `spec.autoUpdate.healthCheckTimeout`
5. **Reset rollback count** - After 3 consecutive rollbacks, auto-update pauses. Fix the issue, then manually update `spec.image.tag` to reset
