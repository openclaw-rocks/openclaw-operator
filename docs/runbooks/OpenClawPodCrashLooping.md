# OpenClawPodCrashLooping

## Meaning

An OpenClaw pod has restarted more than 2 times in the last 10 minutes, indicating a crash loop.

## Impact

The instance is intermittently unavailable. Users experience connection drops and may lose in-flight work.

## Diagnosis

```bash
# Check pod status and restart count
kubectl get pods -n <namespace> -l app.kubernetes.io/instance=<name>

# Check last termination reason
kubectl get pod <name>-0 -n <namespace> -o jsonpath='{.status.containerStatuses[0].lastState.terminated}'

# Check container logs (including previous incarnation)
kubectl logs <name>-0 -n <namespace> -c openclaw --previous --tail=100
kubectl logs <name>-0 -n <namespace> -c openclaw --tail=100

# Check events
kubectl describe pod <name>-0 -n <namespace>

# Check resource limits
kubectl get pod <name>-0 -n <namespace> -o jsonpath='{.spec.containers[0].resources}'
```

## Mitigation

1. **OOMKilled** - Increase memory limits (see OpenClawPodOOMKilled runbook)
2. **Application error** - Check container logs for stack traces or startup errors
3. **Configuration error** - Verify the OpenClaw config is valid
4. **Missing dependencies** - Ensure required skills and MCP servers are available
5. **Liveness probe too aggressive** - Increase `failureThreshold` or `timeoutSeconds`
