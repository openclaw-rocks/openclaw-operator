# OpenClawPVCNearlyFull

## Meaning

The PersistentVolumeClaim for an OpenClaw instance is over 80% full.

## Impact

If the PVC fills up completely, the instance will be unable to write data, causing failures in workspace operations, config updates, and skill installations.

## Diagnosis

```bash
# Check PVC usage
kubectl exec <name>-0 -n <namespace> -c openclaw -- df -h /home/openclaw/.openclaw/

# Check PVC size
kubectl get pvc -n <namespace> -l app.kubernetes.io/instance=<name>

# Check what is consuming space
kubectl exec <name>-0 -n <namespace> -c openclaw -- du -sh /home/openclaw/.openclaw/*/

# Check if large files were created by the agent
kubectl exec <name>-0 -n <namespace> -c openclaw -- find /home/openclaw/.openclaw/ -size +100M -type f
```

## Mitigation

1. **Clean up workspace** - Remove unnecessary files from the workspace directory
2. **Increase PVC size** - If the StorageClass supports volume expansion, increase `spec.storage.persistence.size`
3. **Backup and recreate** - If volume expansion is not supported, backup data, delete the instance, increase size, and restore
4. **Check skills** - Installed npm packages can consume significant space; review `spec.skills`
