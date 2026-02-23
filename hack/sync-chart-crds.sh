#!/usr/bin/env bash
# sync-chart-crds.sh - Sync CRDs from config/crd/bases/ to Helm chart templates.
#
# Helm does not upgrade CRDs placed in the charts/crds/ directory on
# `helm upgrade`. By placing them in templates/ instead, CRDs are managed
# as regular Helm resources and get updated on every upgrade.
#
# Usage:
#   bash hack/sync-chart-crds.sh          # generate chart CRD templates
#   bash hack/sync-chart-crds.sh --check  # verify they are in sync (CI mode)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CRD_SRC="${REPO_ROOT}/config/crd/bases"
CRD_DST="${REPO_ROOT}/charts/openclaw-operator/templates/crds"

CHECK_MODE=false
if [[ "${1:-}" == "--check" ]]; then
    CHECK_MODE=true
fi

# Generate a Helm template from a raw CRD YAML.
# Adds Helm template guards, resource-policy annotation, and chart labels.
generate_template() {
    local src="$1"
    local dst="$2"

    {
        echo '{{- if .Values.crds.install }}'
        # Process the CRD YAML:
        # - After the controller-gen annotation line, inject helm resource-policy
        # - After the name: line in metadata, inject chart labels
        awk '
        /^  annotations:$/ { print; getline; print; annotation_done=1; next }
        annotation_done == 1 && /^  name:/ {
            print "    {{- if .Values.crds.keep }}"
            print "    \"helm.sh/resource-policy\": keep"
            print "    {{- end }}"
            print $0
            print "  labels:"
            print "    {{- include \"openclaw-operator.labels\" . | nindent 4 }}"
            annotation_done=0
            next
        }
        { print }
        ' "$src"
        echo '{{- end }}'
    } > "$dst"
}

mkdir -p "$CRD_DST"

if $CHECK_MODE; then
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    for crd_file in "$CRD_SRC"/*.yaml; do
        name=$(basename "$crd_file")
        generate_template "$crd_file" "$TMPDIR/$name"

        if ! diff -q "$TMPDIR/$name" "$CRD_DST/$name" >/dev/null 2>&1; then
            echo "::error::Helm chart CRD template is out of sync: $name"
            echo "Run 'make sync-chart-crds' and commit the result."
            diff -u "$CRD_DST/$name" "$TMPDIR/$name" || true
            exit 1
        fi
    done

    echo "Helm chart CRD templates are in sync."
else
    for crd_file in "$CRD_SRC"/*.yaml; do
        name=$(basename "$crd_file")
        generate_template "$crd_file" "$CRD_DST/$name"
        echo "Generated: templates/crds/$name"
    done
fi
