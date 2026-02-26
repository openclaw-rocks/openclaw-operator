# Migration Guide: v1alpha1 to v1

## Overview

Starting with operator version 0.12.0, the OpenClaw CRD API has been graduated from `v1alpha1` to `v1`. The `v1` version is now the **storage version** - all resources are stored as `v1` in etcd.

## Backward Compatibility

The operator includes a **conversion webhook** that automatically converts between `v1alpha1` and `v1`. This means:

- Existing `v1alpha1` resources continue to work without any changes
- You can read and write resources using either `openclaw.rocks/v1alpha1` or `openclaw.rocks/v1`
- The Kubernetes API server handles conversion transparently

## What Changed

The `v1` API is **schema-identical** to `v1alpha1` - no fields were added, removed, or renamed. The version bump signals API stability:

- The API is considered stable and production-ready
- No breaking changes will be made within the `v1` version
- Deprecation of `v1alpha1` will follow standard Kubernetes deprecation policy

## Recommended Actions

### Update your manifests

While `v1alpha1` continues to work, we recommend updating your manifests to use `v1`:

```yaml
# Before
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance

# After
apiVersion: openclaw.rocks/v1
kind: OpenClawInstance
```

### Update Helm values

If you use the Helm chart, no changes are needed - the chart templates use the correct API version automatically.

### Update CI/CD pipelines

If your CI/CD pipelines create or modify OpenClawInstance resources, update the `apiVersion` in your templates.

### Update RBAC

If you have custom RBAC rules referencing the API group, they work with both versions since RBAC is group-scoped (not version-scoped).

## Deprecation Timeline

- **v0.12.0**: `v1` introduced as storage version, `v1alpha1` served as spoke
- **Future release**: `v1alpha1` will be marked as deprecated (still functional)
- **v2.0.0** (if ever): `v1alpha1` removal (with extended migration period)

## Troubleshooting

### "no matching version" errors

If you see version-related errors, ensure the operator is running version 0.12.0+ and the CRDs have been updated. Run:

```bash
kubectl get crd openclawinstances.openclaw.rocks -o jsonpath='{.spec.versions[*].name}'
```

Expected output: `v1 v1alpha1`

### Conversion webhook not working

The conversion webhook requires cert-manager or a manually provisioned TLS certificate. If conversion fails, check:

```bash
kubectl logs -n openclaw-operator-system -l control-plane=controller-manager
```
