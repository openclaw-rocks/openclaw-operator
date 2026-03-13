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
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

var rLog = ctrl.Log.WithName("resources")

const (
	// GatewayPort is the port for the OpenClaw gateway WebSocket server
	GatewayPort = 18789

	// CanvasPort is the port for the OpenClaw canvas HTTP server
	CanvasPort = 18793

	// GatewayProxyPort is the port the nginx reverse proxy listens on for
	// gateway traffic. The Service targets this port instead of GatewayPort
	// because the gateway binds to loopback only.
	GatewayProxyPort = 18790

	// CanvasProxyPort is the port the nginx reverse proxy listens on for
	// canvas traffic. The Service targets this port instead of CanvasPort.
	CanvasProxyPort = 18794

	// DefaultGatewayProxyImage is the default image for the gateway proxy sidecar
	DefaultGatewayProxyImage = "nginx:1.27-alpine"

	// NginxConfigKey is the ConfigMap data key for the nginx stream config
	NginxConfigKey = "nginx.conf"

	// ChromiumPort is the port the chromium sidecar listens on.
	// The browserless image defaults to 3000, but we override it to 9222
	// via the PORT env var to avoid conflicting with the OpenClaw gateway's
	// built-in browser control service on port 3000.
	ChromiumPort = 9222

	// ChromiumProxyPort is the port the nginx CDP proxy listens on.
	// It sits between OpenClaw and the browserless sidecar, injecting
	// Chrome launch args (anti-bot flags) into WebSocket connections.
	ChromiumProxyPort = 9223

	// ChromiumProxyNginxConfigKey is the ConfigMap data key for the
	// chromium CDP proxy nginx config
	ChromiumProxyNginxConfigKey = "chromium-proxy-nginx.conf"

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

	// DefaultTailscaleImage is the default image for the Tailscale sidecar
	DefaultTailscaleImage = "ghcr.io/tailscale/tailscale"

	// TailscaleServeConfigKey is the ConfigMap data key for the Tailscale serve config JSON
	TailscaleServeConfigKey = "tailscale-serve.json"

	// TailscaleStatePath is the path for Tailscale state storage inside the sidecar.
	// Placed under /tmp (an emptyDir) so that tailscaled creates and owns the
	// directory, avoiding a chmod failure on a kubelet-owned mount point.
	TailscaleStatePath = "/tmp/tailscale"

	// TailscaleSocketDir is the directory containing the tailscaled Unix socket
	TailscaleSocketDir = "/var/run/tailscale"

	// TailscaleSocketPath is the full path to the tailscaled Unix socket
	TailscaleSocketPath = "/var/run/tailscale/tailscaled.sock"

	// TailscaleBinPath is the shared volume path where the tailscale CLI binary is copied
	TailscaleBinPath = "/tailscale-bin"

	// TailscaleModeServe is the default Tailscale mode (tailnet-only access)
	TailscaleModeServe = "serve"

	// TailscaleModeFunnel exposes the instance to the public internet via Tailscale Funnel
	TailscaleModeFunnel = "funnel"

	// GatewayBindLoopback is the bind value for loopback mode. The gateway
	// proxy sidecar handles external access; binding to loopback prevents
	// CWE-319 plaintext ws:// errors on non-loopback addresses.
	GatewayBindLoopback = "loopback"

	// DefaultMetricsPort is the default port for the Prometheus metrics endpoint
	DefaultMetricsPort int32 = 9090
)

// DefaultChromiumLaunchArgs are anti-bot Chrome flags injected via the
// chromium CDP proxy's `launch` query parameter. The proxy routes all
// WebSocket connections to browserless's /chromium endpoint with these
// flags (+ user ExtraArgs), ensuring every browser session gets a fresh
// Chrome launched with the correct args. See chromiumProxyNginxConfig.
var DefaultChromiumLaunchArgs = []string{
	"--disable-blink-features=AutomationControlled",
	"--disable-features=AutomationControlled",
	"--no-first-run",
}

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

// ChromiumCDPServiceName returns the name of the headless Service used for
// the Chromium CDP endpoint. A separate headless Service with
// publishNotReadyAddresses is needed so the CDP URL resolves before the pod
// is fully Ready (the main container may still be starting).
func ChromiumCDPServiceName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-cdp"
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

// IsPersistenceEnabled returns true if persistent storage is enabled for the instance.
// Defaults to true when not explicitly set.
func IsPersistenceEnabled(instance *openclawv1alpha1.OpenClawInstance) bool {
	return instance.Spec.Storage.Persistence.Enabled == nil || *instance.Spec.Storage.Persistence.Enabled
}

// ChromiumPVCName returns the name of the Chromium browser profile PVC
func ChromiumPVCName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-chromium-data"
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

// BasicAuthSecretName returns the name of the auto-generated Ingress Basic Auth Secret
func BasicAuthSecretName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-basic-auth"
}

// TailscaleStateSecretName returns the name of the Tailscale state Secret
func TailscaleStateSecretName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-ts-state"
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
	var image string
	if instance.Spec.Image.Digest != "" {
		image = repo + "@" + instance.Spec.Image.Digest
	} else {
		image = repo + ":" + GetImageTag(instance)
	}
	return ApplyRegistryOverride(image, instance.Spec.Registry)
}

// GetTailscaleImage returns the full Tailscale sidecar image reference
func GetTailscaleImage(instance *openclawv1alpha1.OpenClawInstance) string {
	repo := instance.Spec.Tailscale.Image.Repository
	if repo == "" {
		repo = DefaultTailscaleImage
	}

	var image string
	if instance.Spec.Tailscale.Image.Digest != "" {
		image = repo + "@" + instance.Spec.Tailscale.Image.Digest
	} else {
		tag := instance.Spec.Tailscale.Image.Tag
		if tag == "" {
			tag = DefaultImageTag
		}
		image = repo + ":" + tag
	}
	return ApplyRegistryOverride(image, instance.Spec.Registry)
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

// ParseQuantity parses a string into a resource.Quantity, falling back to
// the default if parsing fails. Prevents panics from resource.MustParse.
func ParseQuantity(s, defaultValue string) resource.Quantity {
	if s == "" {
		return resource.MustParse(defaultValue)
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		rLog.Error(err, "Invalid quantity, falling back to default", "value", s, "default", defaultValue)
		return resource.MustParse(defaultValue)
	}
	return q
}

// ApplyRegistryOverride replaces the registry part of an image reference with
// the given registry if the registry is not empty.
//
// Examples:
//
//	ApplyRegistryOverride("ghcr.io/openclaw/openclaw:latest", "my-registry.example.com")
//	→ "my-registry.example.com/openclaw/openclaw:latest"
//
//	ApplyRegistryOverride("ollama/ollama:latest", "my-registry.example.com")
//	→ "my-registry.example.com/ollama/ollama:latest"
//
//	ApplyRegistryOverride("nginx:1.27-alpine", "my-registry.example.com")
//	→ "my-registry.example.com/nginx:1.27-alpine"
//
//	ApplyRegistryOverride("ghcr.io/openclaw/openclaw:latest", "")
//	→ "ghcr.io/openclaw/openclaw:latest" (unchanged)
func ApplyRegistryOverride(image, registry string) string {
	if registry == "" {
		return image
	}
	registry = strings.TrimRight(registry, "/")

	slashIndex := strings.Index(image, "/")
	if slashIndex == -1 {
		// No slashes - it's just an image name (possibly with tag/digest)
		return registry + "/" + image
	}

	firstPart := image[:slashIndex]
	if looksLikeRegistry(firstPart) {
		// Replace the registry part
		return registry + image[slashIndex:]
	}

	// The first part isn't a registry, just prepend the registry
	return registry + "/" + image
}

// looksLikeRegistry checks if a string looks like a container registry hostname.
func looksLikeRegistry(s string) bool {
	// Contains a dot - definitely a registry (e.g., ghcr.io, docker.io)
	if strings.Contains(s, ".") {
		return true
	}

	// IPv6 address (e.g., [::1], [::1]:5000)
	if strings.HasPrefix(s, "[") && strings.Contains(s, "]") {
		return true
	}

	// Has a colon and everything after is digits - it's a registry with port (e.g., localhost:5000)
	colonIdx := strings.LastIndex(s, ":")
	if colonIdx != -1 && colonIdx < len(s)-1 {
		allDigits := true
		for _, c := range s[colonIdx+1:] {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}

	return false
}
