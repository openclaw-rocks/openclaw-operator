# OpenClawPodOOMKilled

## Meaning

An OpenClaw container was terminated by the Linux OOM killer because it exceeded its memory limit.

## Impact

The pod restarts, causing temporary unavailability. Repeated OOM kills lead to crash-looping.

## Diagnosis

```bash
# Confirm OOM kill
kubectl get pod <name>-0 -n <namespace> -o jsonpath='{.status.containerStatuses[0].lastState.terminated.reason}'

# Check current memory usage
kubectl top pod <name>-0 -n <namespace> --containers

# Check memory limits
kubectl get pod <name>-0 -n <namespace> -o jsonpath='{.spec.containers[0].resources.limits.memory}'

# Check if the Chromium sidecar is contributing to memory pressure
kubectl top pod <name>-0 -n <namespace> --containers
```

## Mitigation

1. **Increase memory limits** - Update `spec.resources.limits.memory` in the OpenClawInstance CR
2. **Check Chromium sidecar** - If enabled, the Chromium sidecar can be memory-hungry; set dedicated resource limits via `spec.chromium.resources`
3. **Check Ollama sidecar** - LLM inference requires significant memory; ensure appropriate limits via `spec.ollama.resources`
4. **Reduce workload** - Limit concurrent operations or large file processing
5. **Monitor trends** - Use the Grafana instance dashboard to identify memory growth patterns
