package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OpenClawInstanceSpec defines the desired state of OpenClawInstance
type OpenClawInstanceSpec struct {
	// Image configuration for the OpenClaw container
	// +optional
	Image ImageSpec `json:"image,omitempty"`

	// Config specifies the OpenClaw configuration
	// +optional
	Config ConfigSpec `json:"config,omitempty"`

	// Workspace configures initial workspace files seeded into the instance.
	// Files are copied once on first boot and never overwritten, so agent
	// modifications survive pod restarts.
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// Skills is a list of skills to install via init container.
	// Each entry is either a ClawHub skill identifier (e.g., "@anthropic/mcp-server-fetch")
	// or an npm package prefixed with "npm:" (e.g., "npm:@openclaw/matrix").
	// npm lifecycle scripts are disabled for security (see #91).
	// +kubebuilder:validation:MaxItems=20
	// +optional
	Skills []string `json:"skills,omitempty"`

	// EnvFrom is a list of sources to populate environment variables from
	// Use this for API keys and other secrets (e.g., ANTHROPIC_API_KEY, OPENAI_API_KEY)
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Env is a list of environment variables to set in the container
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources specifies the compute resources for the OpenClaw container
	// +optional
	Resources ResourcesSpec `json:"resources,omitempty"`

	// Security specifies security-related configuration
	// +optional
	Security SecuritySpec `json:"security,omitempty"`

	// Storage specifies persistent storage configuration
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Chromium enables the Chromium sidecar for browser automation
	// +optional
	Chromium ChromiumSpec `json:"chromium,omitempty"`

	// Tailscale configures Tailscale integration for tailnet access and HTTPS
	// +optional
	Tailscale TailscaleSpec `json:"tailscale,omitempty"`

	// Ollama enables the Ollama sidecar for local LLM inference
	// +optional
	Ollama OllamaSpec `json:"ollama,omitempty"`

	// InitContainers is a list of additional init containers to run before the main container.
	// They run after the operator-managed init-config and init-skills containers.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// Sidecars is a list of additional sidecar containers to inject into the pod.
	// Use this for custom sidecars like database proxies, log forwarders, or service meshes.
	// +optional
	Sidecars []corev1.Container `json:"sidecars,omitempty"`

	// SidecarVolumes is a list of additional volumes to make available to sidecar containers.
	// +optional
	SidecarVolumes []corev1.Volume `json:"sidecarVolumes,omitempty"`

	// ExtraVolumes adds additional volumes to the pod.
	// These volumes are available to the main container via ExtraVolumeMounts.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts adds additional volume mounts to the main container.
	// Use with ExtraVolumes to mount ConfigMaps, Secrets, NFS shares, or CSI volumes.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Networking specifies network-related configuration
	// +optional
	Networking NetworkingSpec `json:"networking,omitempty"`

	// Probes configures health probes for the OpenClaw container
	// +optional
	Probes ProbesSpec `json:"probes,omitempty"`

	// Observability configures metrics and logging
	// +optional
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// Availability configures high availability settings
	// +optional
	Availability AvailabilitySpec `json:"availability,omitempty"`

	// RestoreFrom is the B2 backup path to restore data from (e.g. "backups/{tenantId}/{instanceId}/{timestamp}").
	// When set, the operator restores PVC data from this path before creating the StatefulSet.
	// Cleared automatically after successful restore.
	// +optional
	RestoreFrom string `json:"restoreFrom,omitempty"`

	// RuntimeDeps configures built-in init containers that install runtime
	// dependencies (pnpm, Python) for MCP servers and skills.
	// +optional
	RuntimeDeps RuntimeDepsSpec `json:"runtimeDeps,omitempty"`

	// Gateway configures the gateway authentication token
	// +optional
	Gateway GatewaySpec `json:"gateway,omitempty"`

	// AutoUpdate configures automatic version updates from the OCI registry
	// +optional
	AutoUpdate AutoUpdateSpec `json:"autoUpdate,omitempty"`
}

// ImageSpec defines the container image configuration
type ImageSpec struct {
	// Repository is the container image repository
	// +kubebuilder:default="ghcr.io/openclaw/openclaw"
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag is the container image tag
	// +kubebuilder:default="latest"
	// +optional
	Tag string `json:"tag,omitempty"`

	// Digest is the container image digest (overrides tag if specified)
	// +optional
	Digest string `json:"digest,omitempty"`

	// PullPolicy specifies when to pull the image
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default="IfNotPresent"
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`

	// PullSecrets is a list of secret names for pulling from private registries
	// +optional
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

// ConfigSpec defines the OpenClaw configuration
type ConfigSpec struct {
	// ConfigMapRef references a ConfigMap containing the openclaw.json configuration
	// +optional
	ConfigMapRef *ConfigMapKeySelector `json:"configMapRef,omitempty"`

	// Raw is inline openclaw.json configuration (used if ConfigMapRef is not set)
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Raw *RawConfig `json:"raw,omitempty"`

	// MergeMode controls how operator-managed config is applied to the PVC.
	// "overwrite" replaces the config file on every pod restart.
	// "merge" deep-merges operator config with existing PVC config, preserving runtime changes.
	// +kubebuilder:validation:Enum=overwrite;merge
	// +kubebuilder:default="overwrite"
	// +optional
	MergeMode string `json:"mergeMode,omitempty"`

	// Format specifies the config file format.
	// "json" (default) expects standard JSON. "json5" accepts JSON5 (comments, trailing commas).
	// JSON5 is converted to standard JSON by the init container using npx json5.
	// JSON5 requires configMapRef (inline raw config must be valid JSON).
	// +kubebuilder:validation:Enum=json;json5
	// +kubebuilder:default="json"
	// +optional
	Format string `json:"format,omitempty"`
}

// ConfigMapKeySelector selects a key from a ConfigMap
type ConfigMapKeySelector struct {
	// Name of the ConfigMap
	Name string `json:"name"`

	// Key in the ConfigMap to use
	// +kubebuilder:default="openclaw.json"
	// +optional
	Key string `json:"key,omitempty"`
}

// RawConfig holds arbitrary JSON configuration for openclaw.json
// +kubebuilder:pruning:PreserveUnknownFields
type RawConfig struct {
	runtime.RawExtension `json:",inline"`
}

// WorkspaceSpec configures initial workspace files for the instance.
// Files listed in InitialFiles are seeded once (only if they don't already
// exist on the PVC), so agent modifications survive pod restarts.
type WorkspaceSpec struct {
	// InitialFiles maps filenames to their content. Each file is written
	// to the workspace directory only if it does not already exist.
	// +kubebuilder:validation:MaxProperties=50
	// +optional
	InitialFiles map[string]string `json:"initialFiles,omitempty"`

	// InitialDirectories is a list of directories to create (mkdir -p)
	// inside the workspace directory. Nested paths like "tools/scripts" are allowed.
	// +kubebuilder:validation:MaxItems=20
	// +optional
	InitialDirectories []string `json:"initialDirectories,omitempty"`
}

// ResourcesSpec defines compute resource requirements
type ResourcesSpec struct {
	// Requests describes the minimum amount of compute resources required
	// +optional
	Requests ResourceList `json:"requests,omitempty"`

	// Limits describes the maximum amount of compute resources allowed
	// +optional
	Limits ResourceList `json:"limits,omitempty"`
}

// ResourceList defines CPU and memory resources
type ResourceList struct {
	// CPU resource (e.g., "500m", "2")
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource (e.g., "512Mi", "2Gi")
	// +optional
	Memory string `json:"memory,omitempty"`
}

// SecuritySpec defines security-related configuration
type SecuritySpec struct {
	// PodSecurityContext holds pod-level security attributes
	// +optional
	PodSecurityContext *PodSecurityContextSpec `json:"podSecurityContext,omitempty"`

	// ContainerSecurityContext holds container-level security attributes
	// +optional
	ContainerSecurityContext *ContainerSecurityContextSpec `json:"containerSecurityContext,omitempty"`

	// NetworkPolicy configures network isolation
	// +optional
	NetworkPolicy NetworkPolicySpec `json:"networkPolicy,omitempty"`

	// RBAC configures role-based access control
	// +optional
	RBAC RBACSpec `json:"rbac,omitempty"`

	// CABundle injects a custom CA certificate bundle into all containers.
	// Use this in environments with TLS-intercepting proxies or private CAs.
	// +optional
	CABundle *CABundleSpec `json:"caBundle,omitempty"`
}

// CABundleSpec configures custom CA certificate injection.
type CABundleSpec struct {
	// ConfigMapName is the name of a ConfigMap containing the CA bundle.
	// The ConfigMap should have a key matching the Key field.
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// SecretName is the name of a Secret containing the CA bundle.
	// The Secret should have a key matching the Key field.
	// Only one of ConfigMapName or SecretName should be set.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Key is the key in the ConfigMap or Secret containing the CA bundle.
	// +kubebuilder:default="ca-bundle.crt"
	// +optional
	Key string `json:"key,omitempty"`
}

// PodSecurityContextSpec defines pod-level security context
type PodSecurityContextSpec struct {
	// RunAsUser is the UID to run the entrypoint of the container process
	// +kubebuilder:default=1000
	// +optional
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// RunAsGroup is the GID to run the entrypoint of the container process
	// +kubebuilder:default=1000
	// +optional
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	// FSGroup is a special supplemental group that applies to all containers
	// +kubebuilder:default=1000
	// +optional
	FSGroup *int64 `json:"fsGroup,omitempty"`

	// FSGroupChangePolicy defines the behavior of changing ownership and permission of the volume.
	// "OnRootMismatch" skips recursive chown when ownership already matches, improving startup
	// time for large PVCs. "Always" recursively chowns on every mount (Kubernetes default).
	// +kubebuilder:validation:Enum=OnRootMismatch;Always
	// +optional
	FSGroupChangePolicy *corev1.PodFSGroupChangePolicy `json:"fsGroupChangePolicy,omitempty"`

	// RunAsNonRoot indicates that the container must run as a non-root user
	// +kubebuilder:default=true
	// +optional
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`
}

// ContainerSecurityContextSpec defines container-level security context
type ContainerSecurityContextSpec struct {
	// AllowPrivilegeEscalation controls whether a process can gain more privileges
	// +kubebuilder:default=false
	// +optional
	AllowPrivilegeEscalation *bool `json:"allowPrivilegeEscalation,omitempty"`

	// ReadOnlyRootFilesystem mounts the container's root filesystem as read-only
	// The PVC at ~/.openclaw/ provides writable home, and a /tmp emptyDir handles temp files
	// +kubebuilder:default=true
	// +optional
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty"`

	// Capabilities to add/drop
	// +optional
	Capabilities *corev1.Capabilities `json:"capabilities,omitempty"`
}

// NetworkPolicySpec configures network isolation for the OpenClaw instance
type NetworkPolicySpec struct {
	// Enabled enables network policy creation
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// AllowedIngressCIDRs is a list of CIDRs allowed to access this instance
	// +optional
	AllowedIngressCIDRs []string `json:"allowedIngressCIDRs,omitempty"`

	// AllowedIngressNamespaces is a list of namespace names allowed to access this instance
	// +optional
	AllowedIngressNamespaces []string `json:"allowedIngressNamespaces,omitempty"`

	// AllowedEgressCIDRs is a list of CIDRs this instance can reach
	// Default allows all egress on port 443 for AI APIs
	// +optional
	AllowedEgressCIDRs []string `json:"allowedEgressCIDRs,omitempty"`

	// AllowDNS allows DNS resolution (port 53)
	// +kubebuilder:default=true
	// +optional
	AllowDNS *bool `json:"allowDNS,omitempty"`

	// AdditionalEgress appends custom egress rules to the default DNS + HTTPS rules.
	// Use this to allow traffic to cluster-internal services on non-standard ports.
	// +optional
	AdditionalEgress []networkingv1.NetworkPolicyEgressRule `json:"additionalEgress,omitempty"`
}

// RBACSpec configures RBAC for the OpenClaw instance
type RBACSpec struct {
	// CreateServiceAccount creates a dedicated ServiceAccount for the instance
	// +kubebuilder:default=true
	// +optional
	CreateServiceAccount *bool `json:"createServiceAccount,omitempty"`

	// ServiceAccountName is the name of an existing ServiceAccount to use
	// Only used if CreateServiceAccount is false
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// ServiceAccountAnnotations are annotations to add to the managed ServiceAccount.
	// Use this for cloud provider integrations like AWS IRSA or GCP Workload Identity.
	// +optional
	ServiceAccountAnnotations map[string]string `json:"serviceAccountAnnotations,omitempty"`

	// AdditionalRules adds custom RBAC rules to the generated Role
	// +optional
	AdditionalRules []RBACRule `json:"additionalRules,omitempty"`
}

// RBACRule represents a RBAC rule
type RBACRule struct {
	// APIGroups is the name of the APIGroup that contains the resources
	APIGroups []string `json:"apiGroups"`
	// Resources is a list of resources this rule applies to
	Resources []string `json:"resources"`
	// Verbs is a list of verbs that apply to the resources
	Verbs []string `json:"verbs"`
}

// StorageSpec defines persistent storage configuration
type StorageSpec struct {
	// Persistence configures the PersistentVolumeClaim
	// +optional
	Persistence PersistenceSpec `json:"persistence,omitempty"`
}

// PersistenceSpec defines PVC configuration
type PersistenceSpec struct {
	// Enabled enables persistent storage
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// StorageClass is the name of the StorageClass to use
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// Size is the size of the PVC (e.g., "10Gi")
	// +kubebuilder:default="10Gi"
	// +optional
	Size string `json:"size,omitempty"`

	// AccessModes contains the desired access modes for the PVC
	// +kubebuilder:default={"ReadWriteOnce"}
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// ExistingClaim is the name of an existing PVC to use
	// +optional
	ExistingClaim string `json:"existingClaim,omitempty"`
}

// ChromiumSpec defines the Chromium sidecar configuration
type ChromiumSpec struct {
	// Enabled enables the Chromium sidecar for browser automation
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image configures the Chromium container image
	// +optional
	Image ChromiumImageSpec `json:"image,omitempty"`

	// Resources specifies compute resources for the Chromium container
	// +optional
	Resources ResourcesSpec `json:"resources,omitempty"`
}

// ChromiumImageSpec defines the Chromium container image
type ChromiumImageSpec struct {
	// Repository is the container image repository
	// +kubebuilder:default="ghcr.io/browserless/chromium"
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag is the container image tag
	// +kubebuilder:default="latest"
	// +optional
	Tag string `json:"tag,omitempty"`

	// Digest is the container image digest for supply chain security
	// +optional
	Digest string `json:"digest,omitempty"`
}

// TailscaleSpec configures Tailscale integration for secure tailnet access.
// When enabled, the operator merges gateway.tailscale settings into the
// OpenClaw config and injects the auth key.
type TailscaleSpec struct {
	// Enabled enables Tailscale integration
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Mode selects the Tailscale mode.
	// "serve" exposes the instance to tailnet members only (default).
	// "funnel" exposes the instance to the public internet via Tailscale Funnel.
	// +kubebuilder:validation:Enum=serve;funnel
	// +kubebuilder:default="serve"
	// +optional
	Mode string `json:"mode,omitempty"`

	// AuthKeySecretRef references a Secret containing the Tailscale auth key.
	// The Secret must have a key matching AuthKeySecretKey (default: "authkey").
	// Use ephemeral+reusable keys from the Tailscale admin console.
	// +optional
	AuthKeySecretRef *corev1.LocalObjectReference `json:"authKeySecretRef,omitempty"`

	// AuthKeySecretKey is the key in the referenced Secret.
	// +kubebuilder:default="authkey"
	// +optional
	AuthKeySecretKey string `json:"authKeySecretKey,omitempty"`

	// Hostname sets the Tailscale device name (defaults to instance name).
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// AuthSSO enables passwordless login for tailnet members.
	// Sets gateway.auth.allowTailscale=true in the OpenClaw config.
	// +kubebuilder:default=false
	// +optional
	AuthSSO bool `json:"authSSO,omitempty"`
}

// OllamaSpec defines the Ollama sidecar configuration
type OllamaSpec struct {
	// Enabled enables the Ollama sidecar
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image configures the Ollama container image
	// +optional
	Image OllamaImageSpec `json:"image,omitempty"`

	// Models is a list of models to pre-pull during pod init (e.g. ["llama3.2", "nomic-embed-text"])
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Models []string `json:"models,omitempty"`

	// Resources specifies compute resources for the Ollama container
	// +optional
	Resources ResourcesSpec `json:"resources,omitempty"`

	// Storage configures the model cache volume
	// +optional
	Storage OllamaStorageSpec `json:"storage,omitempty"`

	// GPU is the number of NVIDIA GPUs to allocate (sets nvidia.com/gpu resource limit)
	// +kubebuilder:validation:Minimum=0
	// +optional
	GPU *int32 `json:"gpu,omitempty"`
}

// OllamaImageSpec defines the Ollama container image
type OllamaImageSpec struct {
	// Repository is the container image repository
	// +kubebuilder:default="ollama/ollama"
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag is the container image tag
	// +kubebuilder:default="latest"
	// +optional
	Tag string `json:"tag,omitempty"`

	// Digest is the container image digest for supply chain security
	// +optional
	Digest string `json:"digest,omitempty"`
}

// OllamaStorageSpec configures the Ollama model cache volume
type OllamaStorageSpec struct {
	// SizeLimit is the size limit for the emptyDir model cache (default "20Gi")
	// +kubebuilder:default="20Gi"
	// +optional
	SizeLimit string `json:"sizeLimit,omitempty"`

	// ExistingClaim is the name of an existing PVC for persistent model storage
	// +optional
	ExistingClaim string `json:"existingClaim,omitempty"`
}

// NetworkingSpec defines network-related configuration
type NetworkingSpec struct {
	// Service configures the Kubernetes Service
	// +optional
	Service ServiceSpec `json:"service,omitempty"`

	// Ingress configures the Kubernetes Ingress
	// +optional
	Ingress IngressSpec `json:"ingress,omitempty"`
}

// ServiceSpec defines the Service configuration
type ServiceSpec struct {
	// Type is the Kubernetes Service type
	// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer;NodePort
	// +kubebuilder:default="ClusterIP"
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// Annotations to add to the Service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Ports defines custom ports exposed on the Service.
	// When set, these replace the default gateway and canvas ports.
	// When empty, the operator creates default gateway (18789) and canvas (18793) ports.
	// +kubebuilder:validation:MaxItems=20
	// +optional
	Ports []ServicePortSpec `json:"ports,omitempty"`
}

// ServicePortSpec defines a port exposed by the Service
type ServicePortSpec struct {
	// Name is the name of the port
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Port is the port number exposed on the Service
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// TargetPort is the port on the container to route to (defaults to Port)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	TargetPort *int32 `json:"targetPort,omitempty"`

	// Protocol is the protocol for the port
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	// +kubebuilder:default="TCP"
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`
}

// IngressSpec defines the Ingress configuration
type IngressSpec struct {
	// Enabled enables Ingress creation
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// ClassName is the name of the IngressClass to use
	// +optional
	ClassName *string `json:"className,omitempty"`

	// Annotations to add to the Ingress
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Hosts is a list of hosts to route traffic for
	// +optional
	Hosts []IngressHost `json:"hosts,omitempty"`

	// TLS configuration
	// +optional
	TLS []IngressTLS `json:"tls,omitempty"`

	// Security configures ingress security settings
	// +optional
	Security IngressSecuritySpec `json:"security,omitempty"`
}

// IngressHost defines a host for the Ingress
type IngressHost struct {
	// Host is the fully qualified domain name
	Host string `json:"host"`

	// Paths is a list of paths to route
	// +optional
	Paths []IngressPath `json:"paths,omitempty"`
}

// IngressPath defines a path for the Ingress
type IngressPath struct {
	// Path is the path to route
	// +kubebuilder:default="/"
	// +optional
	Path string `json:"path,omitempty"`

	// PathType determines how the path should be matched
	// +kubebuilder:validation:Enum=Prefix;Exact;ImplementationSpecific
	// +kubebuilder:default="Prefix"
	// +optional
	PathType string `json:"pathType,omitempty"`

	// Port is the backend service port number to route traffic to.
	// Defaults to the gateway port (18789) when not set.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`
}

// IngressTLS defines TLS configuration for the Ingress
type IngressTLS struct {
	// Hosts are a list of hosts included in the TLS certificate
	Hosts []string `json:"hosts,omitempty"`

	// SecretName is the name of the secret containing the TLS certificate
	SecretName string `json:"secretName,omitempty"`
}

// IngressSecuritySpec defines security settings for the Ingress
type IngressSecuritySpec struct {
	// ForceHTTPS redirects all HTTP traffic to HTTPS
	// +kubebuilder:default=true
	// +optional
	ForceHTTPS *bool `json:"forceHTTPS,omitempty"`

	// EnableHSTS enables HTTP Strict Transport Security
	// +kubebuilder:default=true
	// +optional
	EnableHSTS *bool `json:"enableHSTS,omitempty"`

	// RateLimiting configures rate limiting
	// +optional
	RateLimiting *RateLimitingSpec `json:"rateLimiting,omitempty"`
}

// RateLimitingSpec defines rate limiting configuration
type RateLimitingSpec struct {
	// Enabled enables rate limiting
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// RequestsPerSecond is the maximum requests per second
	// +kubebuilder:default=10
	// +optional
	RequestsPerSecond *int32 `json:"requestsPerSecond,omitempty"`
}

// ProbesSpec defines health probe configuration
type ProbesSpec struct {
	// Liveness probe configuration
	// +optional
	Liveness *ProbeSpec `json:"liveness,omitempty"`

	// Readiness probe configuration
	// +optional
	Readiness *ProbeSpec `json:"readiness,omitempty"`

	// Startup probe configuration
	// +optional
	Startup *ProbeSpec `json:"startup,omitempty"`
}

// ProbeSpec defines a health probe
type ProbeSpec struct {
	// Enabled enables the probe
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// InitialDelaySeconds is the number of seconds after the container starts before the probe is initiated
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`

	// PeriodSeconds is how often (in seconds) to perform the probe
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`

	// TimeoutSeconds is the number of seconds after which the probe times out
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// FailureThreshold is the number of times to retry before giving up
	// +optional
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`
}

// ObservabilitySpec defines observability configuration
type ObservabilitySpec struct {
	// Metrics configures Prometheus metrics
	// +optional
	Metrics MetricsSpec `json:"metrics,omitempty"`

	// Logging configures logging
	// +optional
	Logging LoggingSpec `json:"logging,omitempty"`
}

// MetricsSpec defines metrics configuration
type MetricsSpec struct {
	// Enabled enables metrics endpoint
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Port is the port to expose metrics on
	// +kubebuilder:default=9090
	// +optional
	Port *int32 `json:"port,omitempty"`

	// ServiceMonitor configures the Prometheus ServiceMonitor
	// +optional
	ServiceMonitor *ServiceMonitorSpec `json:"serviceMonitor,omitempty"`
}

// ServiceMonitorSpec defines the ServiceMonitor configuration
type ServiceMonitorSpec struct {
	// Enabled enables ServiceMonitor creation
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Interval is the scrape interval
	// +kubebuilder:default="30s"
	// +optional
	Interval string `json:"interval,omitempty"`

	// Labels to add to the ServiceMonitor
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// LoggingSpec defines logging configuration
type LoggingSpec struct {
	// Level is the log level
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default="info"
	// +optional
	Level string `json:"level,omitempty"`

	// Format is the log format
	// +kubebuilder:validation:Enum=json;text
	// +kubebuilder:default="json"
	// +optional
	Format string `json:"format,omitempty"`
}

// AvailabilitySpec defines high availability settings
type AvailabilitySpec struct {
	// PodDisruptionBudget configures the PDB
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// NodeSelector is a selector which must match a node's labels for the pod to be scheduled
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations are tolerations for pod scheduling
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity specifies affinity scheduling rules
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// PodDisruptionBudgetSpec defines PDB configuration
type PodDisruptionBudgetSpec struct {
	// Enabled enables PDB creation
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable during disruption
	// +kubebuilder:default=1
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`
}

// AutoUpdateSpec configures automatic version updates from the OCI registry
type AutoUpdateSpec struct {
	// Enabled enables automatic version updates
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// CheckInterval is how often to check for new versions (Go duration, e.g. "24h")
	// Minimum: 1h, Maximum: 168h (7 days)
	// +kubebuilder:default="24h"
	// +optional
	CheckInterval string `json:"checkInterval,omitempty"`

	// BackupBeforeUpdate creates a backup before applying updates
	// +kubebuilder:default=true
	// +optional
	BackupBeforeUpdate *bool `json:"backupBeforeUpdate,omitempty"`

	// RollbackOnFailure automatically reverts to the previous version if the
	// updated pod fails to become ready within HealthCheckTimeout
	// +kubebuilder:default=true
	// +optional
	RollbackOnFailure *bool `json:"rollbackOnFailure,omitempty"`

	// HealthCheckTimeout is how long to wait for the updated pod to become ready
	// before triggering a rollback (Go duration, e.g. "10m")
	// Minimum: 2m, Maximum: 30m
	// +kubebuilder:default="10m"
	// +optional
	HealthCheckTimeout string `json:"healthCheckTimeout,omitempty"`
}

// RuntimeDepsSpec configures built-in init containers that install runtime
// dependencies to the data PVC for use by MCP servers and skills.
type RuntimeDepsSpec struct {
	// Pnpm installs pnpm via corepack for npm-based MCP servers and skills.
	// +optional
	Pnpm bool `json:"pnpm,omitempty"`

	// Python installs Python 3.12 and uv for Python-based MCP servers and skills.
	// +optional
	Python bool `json:"python,omitempty"`
}

// GatewaySpec configures the gateway authentication token
type GatewaySpec struct {
	// ExistingSecret is the name of a user-managed Secret containing the gateway token.
	// The Secret must have a key named "token". When set, the operator skips
	// auto-generating a gateway token Secret and uses this Secret instead.
	// +optional
	ExistingSecret string `json:"existingSecret,omitempty"`
}

// AutoUpdateStatus tracks the state of automatic version updates
type AutoUpdateStatus struct {
	// LastCheckTime is when the registry was last checked for new versions
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// LatestVersion is the latest version available in the registry
	// +optional
	LatestVersion string `json:"latestVersion,omitempty"`

	// CurrentVersion is the version currently running
	// +optional
	CurrentVersion string `json:"currentVersion,omitempty"`

	// PendingVersion is set during an in-flight update
	// +optional
	PendingVersion string `json:"pendingVersion,omitempty"`

	// UpdatePhase tracks progress of an in-flight update
	// +kubebuilder:validation:Enum="";BackingUp;ApplyingUpdate;HealthCheck;RollingBack
	// +optional
	UpdatePhase string `json:"updatePhase,omitempty"`

	// LastUpdateTime is when the last successful update was applied
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastUpdateError records the error from the last failed update attempt
	// +optional
	LastUpdateError string `json:"lastUpdateError,omitempty"`

	// PreviousVersion is the version before the last update (used for rollback)
	// +optional
	PreviousVersion string `json:"previousVersion,omitempty"`

	// PreUpdateBackupPath is the B2 path of the pre-update backup (used for rollback restore)
	// +optional
	PreUpdateBackupPath string `json:"preUpdateBackupPath,omitempty"`

	// FailedVersion is a version that failed health checks and will be skipped in future checks
	// Cleared when a newer version becomes available
	// +optional
	FailedVersion string `json:"failedVersion,omitempty"`

	// RollbackCount tracks consecutive rollbacks; auto-update pauses after 3
	// Reset to 0 on any successful update
	// +optional
	RollbackCount int32 `json:"rollbackCount,omitempty"`
}

// OpenClawInstanceStatus defines the observed state of OpenClawInstance
type OpenClawInstanceStatus struct {
	// Phase represents the current lifecycle phase of the instance
	// +kubebuilder:validation:Enum=Pending;Provisioning;Running;Degraded;Failed;Terminating;BackingUp;Restoring;Updating
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the instance's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// GatewayEndpoint is the endpoint for the OpenClaw gateway
	// +optional
	GatewayEndpoint string `json:"gatewayEndpoint,omitempty"`

	// CanvasEndpoint is the endpoint for the OpenClaw canvas
	// +optional
	CanvasEndpoint string `json:"canvasEndpoint,omitempty"`

	// LastReconcileTime is the timestamp of the last reconciliation
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// ManagedResources tracks the resources created by the operator
	// +optional
	ManagedResources ManagedResourcesStatus `json:"managedResources,omitempty"`

	// BackupJobName is the name of the active backup Job
	// +optional
	BackupJobName string `json:"backupJobName,omitempty"`

	// RestoreJobName is the name of the active restore Job
	// +optional
	RestoreJobName string `json:"restoreJobName,omitempty"`

	// LastBackupPath is the B2 path of the last successful backup
	// +optional
	LastBackupPath string `json:"lastBackupPath,omitempty"`

	// LastBackupTime is the timestamp of the last successful backup
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// RestoredFrom is the B2 path this instance was restored from
	// +optional
	RestoredFrom string `json:"restoredFrom,omitempty"`

	// AutoUpdate tracks the state of automatic version updates
	// +optional
	AutoUpdate AutoUpdateStatus `json:"autoUpdate,omitempty"`
}

// ManagedResourcesStatus tracks resources created by the operator
type ManagedResourcesStatus struct {
	// StatefulSet is the name of the managed StatefulSet
	// +optional
	StatefulSet string `json:"statefulSet,omitempty"`

	// Deployment is the name of the legacy Deployment (deprecated, used during migration)
	// +optional
	Deployment string `json:"deployment,omitempty"`

	// Service is the name of the managed Service
	// +optional
	Service string `json:"service,omitempty"`

	// ConfigMap is the name of the managed ConfigMap
	// +optional
	ConfigMap string `json:"configMap,omitempty"`

	// PVC is the name of the managed PersistentVolumeClaim
	// +optional
	PVC string `json:"pvc,omitempty"`

	// NetworkPolicy is the name of the managed NetworkPolicy
	// +optional
	NetworkPolicy string `json:"networkPolicy,omitempty"`

	// PodDisruptionBudget is the name of the managed PDB
	// +optional
	PodDisruptionBudget string `json:"podDisruptionBudget,omitempty"`

	// ServiceAccount is the name of the managed ServiceAccount
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Role is the name of the managed Role
	// +optional
	Role string `json:"role,omitempty"`

	// RoleBinding is the name of the managed RoleBinding
	// +optional
	RoleBinding string `json:"roleBinding,omitempty"`

	// GatewayTokenSecret is the name of the auto-generated gateway token Secret
	// +optional
	GatewayTokenSecret string `json:"gatewayTokenSecret,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="Gateway",type=string,JSONPath=`.status.gatewayEndpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OpenClawInstance is the Schema for the openclawinstances API
type OpenClawInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenClawInstanceSpec   `json:"spec,omitempty"`
	Status OpenClawInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenClawInstanceList contains a list of OpenClawInstance
type OpenClawInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenClawInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenClawInstance{}, &OpenClawInstanceList{})
}

// Condition types for OpenClawInstance
const (
	// ConditionTypeReady indicates the overall readiness of the instance
	ConditionTypeReady = "Ready"

	// ConditionTypeConfigValid indicates the configuration is valid
	ConditionTypeConfigValid = "ConfigValid"

	// ConditionTypeStatefulSetReady indicates the StatefulSet is ready
	ConditionTypeStatefulSetReady = "StatefulSetReady"

	// ConditionTypeDeploymentReady indicates the Deployment is ready (deprecated)
	ConditionTypeDeploymentReady = "DeploymentReady"

	// ConditionTypeServiceReady indicates the Service is ready
	ConditionTypeServiceReady = "ServiceReady"

	// ConditionTypeNetworkPolicyReady indicates the NetworkPolicy is ready
	ConditionTypeNetworkPolicyReady = "NetworkPolicyReady"

	// ConditionTypeRBACReady indicates RBAC resources are ready
	ConditionTypeRBACReady = "RBACReady"

	// ConditionTypeStorageReady indicates the PVC is bound
	ConditionTypeStorageReady = "StorageReady"

	// ConditionTypeBackupComplete indicates the backup completed successfully
	ConditionTypeBackupComplete = "BackupComplete"

	// ConditionTypeRestoreComplete indicates the restore completed successfully
	ConditionTypeRestoreComplete = "RestoreComplete"

	// ConditionTypeAutoUpdateAvailable indicates a newer version is available
	ConditionTypeAutoUpdateAvailable = "AutoUpdateAvailable"

	// ConditionTypeSecretsReady indicates all referenced secrets exist
	ConditionTypeSecretsReady = "SecretsReady"
)

// Phase constants
const (
	PhasePending      = "Pending"
	PhaseProvisioning = "Provisioning"
	PhaseRunning      = "Running"
	PhaseDegraded     = "Degraded"
	PhaseFailed       = "Failed"
	PhaseTerminating  = "Terminating"
	PhaseBackingUp    = "BackingUp"
	PhaseRestoring    = "Restoring"
	PhaseUpdating     = "Updating"
)
