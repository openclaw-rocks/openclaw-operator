/*
Copyright 2026 OpenClaw.rocks

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

const (
	// GatewayPort is the port for the OpenClaw gateway WebSocket server
	GatewayPort = 18789

	// CanvasPort is the port for the OpenClaw canvas HTTP server
	CanvasPort = 18793

	// ChromiumPort is the port declared on the container (metadata only).
	// The browserless image actually listens on BrowserlessCDPPort.
	ChromiumPort = 9222

	// BrowserlessCDPPort is the actual port the ghcr.io/browserless/chromium
	// image listens on for CDP and HTTP API requests.
	BrowserlessCDPPort = 3000

	// OllamaPort is the port for the Ollama API
	OllamaPort = 11434

	// WebTerminalPort is the port for the ttyd web terminal
	WebTerminalPort = 7681

	// ConfigMergeModeMerge is the merge mode that deep-merges config with existing PVC config
	ConfigMergeModeMerge = "merge"

	// ConfigFormatJSON5 is the config format that accepts JSON5 (comments, trailing commas)
	ConfigFormatJSON5 = "json5"

	// DefaultCABundleKey is the default key in a ConfigMap or Secret for the CA bundle
	DefaultCABundleKey = "ca-bundle.crt"

	// UvImage is the image used for Python/uv runtime dependency installation.
	// Must be a shell-capable variant (not distroless) since the init script uses sh -c.
	UvImage = "ghcr.io/astral-sh/uv:0.6-bookworm-slim"

	// RuntimeDepsLocalBin is the path where runtime dependency binaries are installed on the PVC
	RuntimeDepsLocalBin = "/home/openclaw/.openclaw/.local/bin"

	// DefaultImageTag is the default tag used for container images
	DefaultImageTag = "latest"

	// AppName is the application name used in labels
	AppName = "openclaw"

	// ComponentLabel is the component label key
	ComponentLabel = "app.kubernetes.io/component"

	// GatewayTokenSecretKey is the data key used in the gateway token Secret
	GatewayTokenSecretKey = "token"

	// DefaultTailscaleAuthKeySecretKey is the default key in the Tailscale auth key Secret
	DefaultTailscaleAuthKeySecretKey = "authkey"

	// TailscaleModeServe is the default Tailscale mode (tailnet-only access)
	TailscaleModeServe = "serve"

	// TailscaleModeFunnel exposes the instance to the public internet via Tailscale Funnel
	TailscaleModeFunnel = "funnel"

	// GatewayBindLoopback is the bind value for loopback mode (required for Tailscale serve/funnel)
	GatewayBindLoopback = "loopback"

	// GatewayBindLAN is the default bind value for LAN mode (pod IP, required for TCPSocket probes)
	GatewayBindLAN = "lan"

	// DefaultMetricsPort is the default port for the Prometheus metrics endpoint
	DefaultMetricsPort int32 = 9090
)

// Labels returns the standard labels for an OpenClawInstance
func Labels(instance *openclawv1alpha1.OpenClawInstance) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       AppName,
		"app.kubernetes.io/instance":   instance.Name,
		"app.kubernetes.io/managed-by": "openclaw-operator",
	}
}

// SelectorLabels returns the labels used for selecting pods
func SelectorLabels(instance *openclawv1alpha1.OpenClawInstance) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     AppName,
		"app.kubernetes.io/instance": instance.Name,
	}
}

// StatefulSetName returns the name of the StatefulSet
func StatefulSetName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// DeploymentName returns the name of the legacy Deployment (used during migration)
func DeploymentName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// ServiceName returns the name of the Service
func ServiceName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// ServiceAccountName returns the name of the ServiceAccount
func ServiceAccountName(instance *openclawv1alpha1.OpenClawInstance) string {
	if instance.Spec.Security.RBAC.ServiceAccountName != "" {
		return instance.Spec.Security.RBAC.ServiceAccountName
	}
	return instance.Name
}

// RoleName returns the name of the Role
func RoleName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// RoleBindingName returns the name of the RoleBinding
func RoleBindingName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// ConfigMapName returns the name of the ConfigMap
func ConfigMapName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-config"
}

// WorkspaceConfigMapName returns the name of the workspace ConfigMap
func WorkspaceConfigMapName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-workspace"
}

// PVCName returns the name of the PVC
func PVCName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-data"
}

// NetworkPolicyName returns the name of the NetworkPolicy
func NetworkPolicyName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// PDBName returns the name of the PodDisruptionBudget
func PDBName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// IngressName returns the name of the Ingress
func IngressName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name
}

// GatewayTokenSecretName returns the name of the auto-generated gateway token Secret
func GatewayTokenSecretName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-gateway-token"
}

// GetImageRepository returns the image repository with defaults
func GetImageRepository(instance *openclawv1alpha1.OpenClawInstance) string {
	if instance.Spec.Image.Repository != "" {
		return instance.Spec.Image.Repository
	}
	return "ghcr.io/openclaw/openclaw"
}

// GetImageTag returns the image tag with defaults
func GetImageTag(instance *openclawv1alpha1.OpenClawInstance) string {
	if instance.Spec.Image.Tag != "" {
		return instance.Spec.Image.Tag
	}
	return DefaultImageTag
}

// GetImage returns the full image reference
func GetImage(instance *openclawv1alpha1.OpenClawInstance) string {
	repo := GetImageRepository(instance)
	if instance.Spec.Image.Digest != "" {
		return repo + "@" + instance.Spec.Image.Digest
	}
	return repo + ":" + GetImageTag(instance)
}

// IsTailscaleServeOrFunnel returns true when Tailscale is enabled with serve
// or funnel mode. Both modes require gateway.bind=loopback.
func IsTailscaleServeOrFunnel(instance *openclawv1alpha1.OpenClawInstance) bool {
	if !instance.Spec.Tailscale.Enabled {
		return false
	}
	mode := instance.Spec.Tailscale.Mode
	return mode == "" || mode == TailscaleModeServe || mode == TailscaleModeFunnel
}

// IsMetricsEnabled returns true if the metrics endpoint is enabled for the instance
func IsMetricsEnabled(instance *openclawv1alpha1.OpenClawInstance) bool {
	return instance.Spec.Observability.Metrics.Enabled == nil || *instance.Spec.Observability.Metrics.Enabled
}

// MetricsPort returns the configured metrics port or the default
func MetricsPort(instance *openclawv1alpha1.OpenClawInstance) int32 {
	if instance.Spec.Observability.Metrics.Port != nil {
		return *instance.Spec.Observability.Metrics.Port
	}
	return DefaultMetricsPort
}

// Ptr returns a pointer to the given value
func Ptr[T any](v T) *T {
	return &v
}
