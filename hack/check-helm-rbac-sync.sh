#!/usr/bin/env bash
# hack/check-helm-rbac-sync.sh
#
# Verifies that every RBAC permission from the kubebuilder-generated role
# (config/rbac/role.yaml) is present in the Helm chart ClusterRole
# (charts/openclaw-operator/templates/rbac.yaml).
#
# The generated role is the source of truth (derived from +kubebuilder:rbac
# markers). The Helm chart may be a superset but must not be missing any
# permissions.

set -euo pipefail

GENERATED="config/rbac/role.yaml"
HELM="charts/openclaw-operator/templates/rbac.yaml"

if [ ! -f "$GENERATED" ]; then
  echo "::error::Generated RBAC not found at $GENERATED — run 'make manifests' first"
  exit 1
fi
if [ ! -f "$HELM" ]; then
  echo "::error::Helm chart RBAC not found at $HELM"
  exit 1
fi

# Parse kubebuilder-generated role.yaml (multi-line YAML) into
# sorted "apiGroup|resource|verb" triples, one per line.
parse_generated() {
  awk '
    /^- apiGroups:/ {
      # Flush previous rule
      for (r = 0; r < nres; r++)
        for (v = 0; v < nvrb; v++)
          print grp "|" res[r] "|" vrb[v]
      delete res; delete vrb; grp = ""; nres = 0; nvrb = 0
      section = "groups"
      next
    }
    /^  resources:/ { section = "resources"; next }
    /^  verbs:/     { section = "verbs";     next }

    section == "groups" && /^  - / {
      s = $0; sub(/^  - /, "", s); gsub(/"/, "", s); grp = s
    }
    section == "resources" && /^  - / {
      s = $0; sub(/^  - /, "", s); gsub(/"/, "", s); res[nres++] = s
    }
    section == "verbs" && /^  - / {
      s = $0; sub(/^  - /, "", s); gsub(/"/, "", s); vrb[nvrb++] = s
    }

    END {
      for (r = 0; r < nres; r++)
        for (v = 0; v < nvrb; v++)
          print grp "|" res[r] "|" vrb[v]
    }
  ' "$GENERATED" | sort -u
}

# Parse Helm chart ClusterRole (inline JSON arrays) into the same
# "apiGroup|resource|verb" triple format.
# Reads only the first YAML document (before ---) and skips Helm templates.
parse_helm() {
  awk '
    /^---/ { exit }
    /\{\{/ { next }
    /^\s*#/ { next }

    /apiGroups:/ {
      s = $0; sub(/.*\[/, "", s); sub(/\].*/, "", s)
      ngroups = split(s, arr, ",")
      for (i = 1; i <= ngroups; i++) {
        g = arr[i]; gsub(/[ "'\''"]/, "", g); groups[i] = g
      }
      next
    }
    /resources:/ {
      s = $0; sub(/.*\[/, "", s); sub(/\].*/, "", s)
      nresources = split(s, arr, ",")
      for (i = 1; i <= nresources; i++) {
        r = arr[i]; gsub(/[ "'\''"]/, "", r); resources[i] = r
      }
      next
    }
    /verbs:/ {
      s = $0; sub(/.*\[/, "", s); sub(/\].*/, "", s)
      nverbs = split(s, arr, ",")
      for (i = 1; i <= nverbs; i++) {
        v = arr[i]; gsub(/[ "'\''"]/, "", v); verbs[i] = v
      }
      # Emit triples for this rule
      for (g = 1; g <= ngroups; g++)
        for (r = 1; r <= nresources; r++)
          for (v = 1; v <= nverbs; v++)
            print groups[g] "|" resources[r] "|" verbs[v]
    }
  ' "$HELM" | sort -u
}

GENERATED_TRIPLES=$(parse_generated)
HELM_TRIPLES=$(parse_helm)

# Find triples in generated but not in Helm (i.e., missing permissions)
MISSING=$(comm -23 <(echo "$GENERATED_TRIPLES") <(echo "$HELM_TRIPLES"))

if [ -n "$MISSING" ]; then
  echo "::error::Helm chart RBAC is missing permissions from kubebuilder markers."
  echo ""
  echo "The following (apiGroup | resource | verb) triples are in"
  echo "config/rbac/role.yaml but NOT in the Helm chart ClusterRole:"
  echo ""
  echo "$MISSING" | while IFS='|' read -r g r v; do
    group="$g"
    if [ -z "$group" ]; then group='""'; fi
    echo "  apiGroup=$group  resource=$r  verb=$v"
  done
  echo ""
  echo "Fix: update charts/openclaw-operator/templates/rbac.yaml to match"
  echo "the kubebuilder markers in internal/controller/."
  exit 1
fi

echo "Helm chart RBAC is in sync with kubebuilder markers."
