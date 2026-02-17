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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// BuildStatefulSet creates a StatefulSet for the OpenClawInstance.
// If gatewayTokenSecretName is non-empty and the user hasn't already set
// OPENCLAW_GATEWAY_TOKEN in spec.env, the env var is injected via SecretKeyRef.
func BuildStatefulSet(instance *openclawv1alpha1.OpenClawInstance, gatewayTokenSecretName ...string) *appsv1.StatefulSet {
	labels := Labels(instance)
	selectorLabels := SelectorLabels(instance)

	// Calculate config hash for rollout trigger
	configHash := calculateConfigHash(instance)

	// Resolve optional gateway token secret name
	var gwSecretName string
	if len(gatewayTokenSecretName) > 0 {
		gwSecretName = gatewayTokenSecretName[0]
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StatefulSetName(instance),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             Ptr(int32(1)), // OpenClaw is single-instance
			RevisionHistoryLimit: Ptr(int32(10)),
			ServiceName:          ServiceName(instance),
			PodManagementPolicy:  appsv1.ParallelPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"openclaw.rocks/config-hash": configHash,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            ServiceAccountName(instance),
					AutomountServiceAccountToken:  Ptr(false),
					SecurityContext:               buildPodSecurityContext(instance),
					InitContainers:                buildInitContainers(instance),
					Containers:                    buildContainers(instance, gwSecretName),
					Volumes:                       buildVolumes(instance),
					NodeSelector:                  instance.Spec.Availability.NodeSelector,
					Tolerations:                   instance.Spec.Availability.Tolerations,
					Affinity:                      instance.Spec.Availability.Affinity,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					DNSPolicy:                     corev1.DNSClusterFirst,
					SchedulerName:                 corev1.DefaultSchedulerName,
					TerminationGracePeriodSeconds: Ptr(int64(30)),
				},
			},
		},
	}

	// Add image pull secrets
	sts.Spec.Template.Spec.ImagePullSecrets = append(
		sts.Spec.Template.Spec.ImagePullSecrets,
		instance.Spec.Image.PullSecrets...,
	)

	return sts
}

// buildPodSecurityContext creates the pod-level security context
func buildPodSecurityContext(instance *openclawv1alpha1.OpenClawInstance) *corev1.PodSecurityContext {
	psc := &corev1.PodSecurityContext{
		RunAsNonRoot: Ptr(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	// Apply user overrides or defaults
	spec := instance.Spec.Security.PodSecurityContext
	if spec != nil {
		if spec.RunAsUser != nil {
			psc.RunAsUser = spec.RunAsUser
		} else {
			psc.RunAsUser = Ptr(int64(1000))
		}
		if spec.RunAsGroup != nil {
			psc.RunAsGroup = spec.RunAsGroup
		} else {
			psc.RunAsGroup = Ptr(int64(1000))
		}
		if spec.FSGroup != nil {
			psc.FSGroup = spec.FSGroup
		} else {
			psc.FSGroup = Ptr(int64(1000))
		}
		if spec.FSGroupChangePolicy != nil {
			psc.FSGroupChangePolicy = spec.FSGroupChangePolicy
		}
		if spec.RunAsNonRoot != nil {
			psc.RunAsNonRoot = spec.RunAsNonRoot
		}
	} else {
		psc.RunAsUser = Ptr(int64(1000))
		psc.RunAsGroup = Ptr(int64(1000))
		psc.FSGroup = Ptr(int64(1000))
	}

	return psc
}

// buildContainerSecurityContext creates the container-level security context
func buildContainerSecurityContext(instance *openclawv1alpha1.OpenClawInstance) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		AllowPrivilegeEscalation: Ptr(false),
		ReadOnlyRootFilesystem:   Ptr(true), // PVC at ~/.openclaw/ + /tmp emptyDir provide writable paths
		RunAsNonRoot:             Ptr(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	// Apply user overrides
	spec := instance.Spec.Security.ContainerSecurityContext
	if spec != nil {
		if spec.AllowPrivilegeEscalation != nil {
			sc.AllowPrivilegeEscalation = spec.AllowPrivilegeEscalation
		}
		if spec.ReadOnlyRootFilesystem != nil {
			sc.ReadOnlyRootFilesystem = spec.ReadOnlyRootFilesystem
		}
		if spec.Capabilities != nil {
			sc.Capabilities = spec.Capabilities
		}
	}

	return sc
}

// buildContainers creates the container specs
func buildContainers(instance *openclawv1alpha1.OpenClawInstance, gatewayTokenSecretName string) []corev1.Container {
	containers := []corev1.Container{
		buildMainContainer(instance, gatewayTokenSecretName),
	}

	// Add Chromium sidecar if enabled
	if instance.Spec.Chromium.Enabled {
		containers = append(containers, buildChromiumContainer(instance))
	}

	// Add custom sidecars
	containers = append(containers, instance.Spec.Sidecars...)

	return containers
}

// buildMainContainer creates the main OpenClaw container
func buildMainContainer(instance *openclawv1alpha1.OpenClawInstance, gatewayTokenSecretName string) corev1.Container {
	container := corev1.Container{
		Name:                     "openclaw",
		Image:                    GetImage(instance),
		ImagePullPolicy:          getPullPolicy(instance),
		SecurityContext:          buildContainerSecurityContext(instance),
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		Ports: []corev1.ContainerPort{
			{
				Name:          "gateway",
				ContainerPort: GatewayPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "canvas",
				ContainerPort: CanvasPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env:       buildMainEnv(instance, gatewayTokenSecretName),
		EnvFrom:   instance.Spec.EnvFrom,
		Resources: buildResourceRequirements(instance),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/home/openclaw/.openclaw",
			},
			{
				Name:      "tmp",
				MountPath: "/tmp",
			},
		},
	}

	// Add CA bundle mount and env if configured
	if cab := instance.Spec.Security.CABundle; cab != nil {
		key := cab.Key
		if key == "" {
			key = DefaultCABundleKey
		}
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/custom-ca-bundle.crt",
			SubPath:   key,
			ReadOnly:  true,
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  "NODE_EXTRA_CA_CERTS",
			Value: "/etc/ssl/certs/custom-ca-bundle.crt",
		})
	}

	// Add extra volume mounts from spec
	container.VolumeMounts = append(container.VolumeMounts, instance.Spec.ExtraVolumeMounts...)

	// Add probes
	container.LivenessProbe = buildLivenessProbe(instance)
	container.ReadinessProbe = buildReadinessProbe(instance)
	container.StartupProbe = buildStartupProbe(instance)

	return container
}

// buildMainEnv creates the environment variables for the main container
func buildMainEnv(instance *openclawv1alpha1.OpenClawInstance, gatewayTokenSecretName string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "HOME", Value: "/home/openclaw"},
		// mDNS/Bonjour pairing is unusable in Kubernetes — always disable it
		{Name: "OPENCLAW_DISABLE_BONJOUR", Value: "1"},
	}

	if instance.Spec.Chromium.Enabled {
		env = append(env, corev1.EnvVar{
			Name:  "CHROMIUM_URL",
			Value: "ws://localhost:9222",
		})
	}

	// Inject OPENCLAW_GATEWAY_TOKEN from Secret unless the user already set it in spec.env
	if gatewayTokenSecretName != "" && !hasUserEnv(instance, "OPENCLAW_GATEWAY_TOKEN") {
		env = append(env, corev1.EnvVar{
			Name: "OPENCLAW_GATEWAY_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: gatewayTokenSecretName},
					Key:                  GatewayTokenSecretKey,
				},
			},
		})
	}

	// Prepend runtime deps bin directory to PATH so pnpm/python are discoverable
	if instance.Spec.RuntimeDeps.Pnpm || instance.Spec.RuntimeDeps.Python {
		env = append(env, corev1.EnvVar{
			Name:  "PATH",
			Value: RuntimeDepsLocalBin + ":/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		})
	}

	return append(env, instance.Spec.Env...)
}

// hasUserEnv checks whether the user has defined a specific env var in spec.env.
func hasUserEnv(instance *openclawv1alpha1.OpenClawInstance, name string) bool {
	for _, e := range instance.Spec.Env {
		if e.Name == name {
			return true
		}
	}
	return false
}

// buildInitContainers creates init containers that seed config and workspace
// files into the data volume. Config is always overwritten (operator-managed),
// while workspace files use seed-once semantics (only copied if not present).
// Skills are installed via a separate init container using the OpenClaw image.
func buildInitContainers(instance *openclawv1alpha1.OpenClawInstance) []corev1.Container {
	var initContainers []corev1.Container

	// Config/workspace init container (only if there's something to do)
	if script := BuildInitScript(instance); script != "" {
		mounts := []corev1.VolumeMount{
			{Name: "data", MountPath: "/data"},
		}

		// Config volume mount (only if config exists)
		if configMapKey(instance) != "" {
			mounts = append(mounts, corev1.VolumeMount{Name: "config", MountPath: "/config"})
		}

		// Tmp mount for merge mode (jq writes to /tmp/merged.json) or JSON5 mode (npx writes to /tmp/converted.json)
		if instance.Spec.Config.MergeMode == ConfigMergeModeMerge || instance.Spec.Config.Format == ConfigFormatJSON5 {
			mounts = append(mounts, corev1.VolumeMount{Name: "init-tmp", MountPath: "/tmp"})
		}

		// Workspace volume mount (only if workspace files exist)
		if hasWorkspaceFiles(instance) {
			mounts = append(mounts, corev1.VolumeMount{Name: "workspace-init", MountPath: "/workspace-init", ReadOnly: true})
		}

		// Use jq image for merge mode, OpenClaw image for JSON5 (has Node.js + npx), busybox for overwrite
		initImage := "busybox:1.37"
		if instance.Spec.Config.MergeMode == ConfigMergeModeMerge {
			initImage = JqImage
		} else if instance.Spec.Config.Format == ConfigFormatJSON5 {
			initImage = GetImage(instance)
		}

		// JSON5 mode needs writable rootfs (npx writes to node_modules) and HOME env
		readOnlyRoot := true
		var initEnv []corev1.EnvVar
		initPullPolicy := corev1.PullIfNotPresent
		if instance.Spec.Config.Format == ConfigFormatJSON5 {
			readOnlyRoot = false
			initEnv = []corev1.EnvVar{
				{Name: "HOME", Value: "/tmp"},
				{Name: "NPM_CONFIG_CACHE", Value: "/tmp/.npm"},
			}
			initPullPolicy = getPullPolicy(instance)
		}

		initContainers = append(initContainers, corev1.Container{
			Name:                     "init-config",
			Image:                    initImage,
			Command:                  []string{"sh", "-c", script},
			ImagePullPolicy:          initPullPolicy,
			Env:                      initEnv,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: Ptr(false),
				ReadOnlyRootFilesystem:   Ptr(readOnlyRoot),
				RunAsNonRoot:             Ptr(true),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
			VolumeMounts: mounts,
		})
	}

	// Runtime dependency init containers (run before skills so skills can use pnpm/python)
	if instance.Spec.RuntimeDeps.Pnpm {
		initContainers = append(initContainers, buildPnpmInitContainer(instance))
	}
	if instance.Spec.RuntimeDeps.Python {
		initContainers = append(initContainers, buildPythonInitContainer(instance))
	}

	// Skills init container (only if skills are defined)
	if skillsContainer := buildSkillsInitContainer(instance); skillsContainer != nil {
		initContainers = append(initContainers, *skillsContainer)
	}

	// Custom init containers (user-defined, run after operator-managed ones)
	initContainers = append(initContainers, instance.Spec.InitContainers...)

	return initContainers
}

// shellQuote escapes a string for safe use inside single-quoted shell arguments.
// Single quotes are escaped as '\” (end quote, escaped quote, start quote).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildInitScript generates the shell script for the init container.
// It handles config copy or merge, directory creation (idempotent),
// and workspace file seeding (only if not present).
// Returns "" if there is nothing to do.
func BuildInitScript(instance *openclawv1alpha1.OpenClawInstance) string {
	var lines []string

	// 1. Config handling — overwrite or merge, with optional JSON5 conversion
	if key := configMapKey(instance); key != "" {
		switch {
		case instance.Spec.Config.MergeMode == ConfigMergeModeMerge:
			// Deep-merge operator config with existing PVC config, preserving runtime changes
			lines = append(lines, fmt.Sprintf(
				"if [ -f /data/openclaw.json ]; then\n"+
					"  jq -s '.[0] * .[1]' /data/openclaw.json /config/%s > /tmp/merged.json && mv /tmp/merged.json /data/openclaw.json\n"+
					"else\n"+
					"  cp /config/%s /data/openclaw.json\n"+
					"fi",
				shellQuote(key), shellQuote(key)))
		case instance.Spec.Config.Format == ConfigFormatJSON5:
			// JSON5 overwrite — convert to standard JSON via npx json5
			lines = append(lines, fmt.Sprintf(
				"npx -y json5 /config/%s > /tmp/converted.json && mv /tmp/converted.json /data/openclaw.json",
				shellQuote(key)))
		default:
			// Overwrite (default) — operator-managed config always wins
			lines = append(lines, fmt.Sprintf("cp /config/%s /data/openclaw.json", shellQuote(key)))
		}
	}

	ws := instance.Spec.Workspace

	// 2. Create workspace directories (idempotent)
	if ws != nil {
		// Sort for deterministic output
		dirs := make([]string, len(ws.InitialDirectories))
		copy(dirs, ws.InitialDirectories)
		sort.Strings(dirs)
		for _, dir := range dirs {
			lines = append(lines, fmt.Sprintf("mkdir -p /data/workspace/%s", shellQuote(dir)))
		}
	}

	// 3. Seed workspace files (only if not present)
	if hasWorkspaceFiles(instance) {
		// Ensure the workspace directory exists (may not on first run with emptyDir)
		lines = append(lines, "mkdir -p /data/workspace")
		// Sort keys for deterministic output
		files := make([]string, 0, len(ws.InitialFiles))
		for name := range ws.InitialFiles {
			files = append(files, name)
		}
		sort.Strings(files)
		for _, name := range files {
			q := shellQuote(name)
			lines = append(lines, fmt.Sprintf("[ -f /data/workspace/%s ] || cp /workspace-init/%s /data/workspace/%s", q, q, q))
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}

// BuildSkillsScript generates the shell script for the skills init container.
// It produces `npx -y clawhub install <skill>` for each skill (sorted for determinism).
// Returns "" if no skills are defined.
func BuildSkillsScript(instance *openclawv1alpha1.OpenClawInstance) string {
	if len(instance.Spec.Skills) == 0 {
		return ""
	}

	skills := make([]string, len(instance.Spec.Skills))
	copy(skills, instance.Spec.Skills)
	sort.Strings(skills)

	var lines []string
	for _, skill := range skills {
		lines = append(lines, fmt.Sprintf("npx -y clawhub install %s", shellQuote(skill)))
	}
	return strings.Join(lines, "\n")
}

// buildSkillsInitContainer creates the init container that installs ClawHub skills.
func buildSkillsInitContainer(instance *openclawv1alpha1.OpenClawInstance) *corev1.Container {
	script := BuildSkillsScript(instance)
	if script == "" {
		return nil
	}

	mounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/home/openclaw/.openclaw"},
		{Name: "skills-tmp", MountPath: "/tmp"},
	}

	env := []corev1.EnvVar{
		{Name: "HOME", Value: "/tmp"},
		{Name: "NPM_CONFIG_CACHE", Value: "/tmp/.npm"},
	}

	// CA bundle for skills install (makes network calls)
	if cab := instance.Spec.Security.CABundle; cab != nil {
		key := cab.Key
		if key == "" {
			key = DefaultCABundleKey
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/custom-ca-bundle.crt",
			SubPath:   key,
			ReadOnly:  true,
		})
		env = append(env, corev1.EnvVar{
			Name:  "NODE_EXTRA_CA_CERTS",
			Value: "/etc/ssl/certs/custom-ca-bundle.crt",
		})
	}

	return &corev1.Container{
		Name:                     "init-skills",
		Image:                    GetImage(instance),
		Command:                  []string{"sh", "-c", script},
		ImagePullPolicy:          getPullPolicy(instance),
		Env:                      env,
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: Ptr(false),
			ReadOnlyRootFilesystem:   Ptr(false), // npx needs to write to node_modules
			RunAsNonRoot:             Ptr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		VolumeMounts: mounts,
	}
}

// buildPnpmInitContainer creates the init container that installs pnpm via corepack.
func buildPnpmInitContainer(instance *openclawv1alpha1.OpenClawInstance) corev1.Container {
	script := `set -e
INSTALL_DIR=/home/openclaw/.openclaw/.local
mkdir -p "$INSTALL_DIR/bin"
if [ -x "$INSTALL_DIR/bin/pnpm" ]; then echo "pnpm already installed"; exit 0; fi
export COREPACK_HOME="$INSTALL_DIR/corepack"
corepack enable pnpm --install-directory "$INSTALL_DIR/bin"
pnpm --version`

	mounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/home/openclaw/.openclaw"},
		{Name: "pnpm-tmp", MountPath: "/tmp"},
	}

	env := []corev1.EnvVar{
		{Name: "HOME", Value: "/tmp"},
		{Name: "NPM_CONFIG_CACHE", Value: "/tmp/.npm"},
	}

	// CA bundle for pnpm init (may make network calls)
	if cab := instance.Spec.Security.CABundle; cab != nil {
		key := cab.Key
		if key == "" {
			key = DefaultCABundleKey
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/custom-ca-bundle.crt",
			SubPath:   key,
			ReadOnly:  true,
		})
		env = append(env, corev1.EnvVar{
			Name:  "NODE_EXTRA_CA_CERTS",
			Value: "/etc/ssl/certs/custom-ca-bundle.crt",
		})
	}

	return corev1.Container{
		Name:                     "init-pnpm",
		Image:                    GetImage(instance),
		Command:                  []string{"sh", "-c", script},
		ImagePullPolicy:          getPullPolicy(instance),
		Env:                      env,
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: Ptr(false),
			ReadOnlyRootFilesystem:   Ptr(false), // corepack writes to node internals
			RunAsNonRoot:             Ptr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		VolumeMounts: mounts,
	}
}

// buildPythonInitContainer creates the init container that installs Python 3.12 and uv.
func buildPythonInitContainer(instance *openclawv1alpha1.OpenClawInstance) corev1.Container {
	script := `set -e
INSTALL_DIR=/home/openclaw/.openclaw/.local
mkdir -p "$INSTALL_DIR/bin"
if [ -x "$INSTALL_DIR/bin/python3" ]; then echo "Python already installed"; exit 0; fi
export UV_PYTHON_INSTALL_DIR="$INSTALL_DIR/python"
uv python install 3.12
ln -sf "$INSTALL_DIR/python/"cpython-3.12*/bin/python3 "$INSTALL_DIR/bin/python3"
ln -sf "$INSTALL_DIR/python/"cpython-3.12*/bin/python3 "$INSTALL_DIR/bin/python"
cp /usr/local/bin/uv "$INSTALL_DIR/bin/uv"
python3 --version
uv --version`

	mounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/home/openclaw/.openclaw"},
		{Name: "python-tmp", MountPath: "/tmp"},
	}

	env := []corev1.EnvVar{
		{Name: "HOME", Value: "/tmp"},
		{Name: "XDG_CACHE_HOME", Value: "/tmp/.cache"},
	}

	// CA bundle for uv python install (downloads from the internet)
	if cab := instance.Spec.Security.CABundle; cab != nil {
		key := cab.Key
		if key == "" {
			key = DefaultCABundleKey
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/custom-ca-bundle.crt",
			SubPath:   key,
			ReadOnly:  true,
		})
		env = append(env, corev1.EnvVar{
			Name:  "SSL_CERT_FILE",
			Value: "/etc/ssl/certs/custom-ca-bundle.crt",
		})
	}

	return corev1.Container{
		Name:                     "init-python",
		Image:                    UvImage,
		Command:                  []string{"sh", "-c", script},
		ImagePullPolicy:          corev1.PullIfNotPresent,
		Env:                      env,
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: Ptr(false),
			ReadOnlyRootFilesystem:   Ptr(false), // uv needs writable paths
			RunAsNonRoot:             Ptr(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		VolumeMounts: mounts,
	}
}

// hasWorkspaceFiles returns true if the instance has workspace files to seed.
func hasWorkspaceFiles(instance *openclawv1alpha1.OpenClawInstance) bool {
	return instance.Spec.Workspace != nil && len(instance.Spec.Workspace.InitialFiles) > 0
}

// configMapKey returns the ConfigMap key for the config file.
// Always returns "openclaw.json" for operator-managed configs (including vanilla
// deployments), since the operator always creates a ConfigMap with gateway.bind.
func configMapKey(instance *openclawv1alpha1.OpenClawInstance) string {
	if instance.Spec.Config.ConfigMapRef != nil {
		if instance.Spec.Config.ConfigMapRef.Key != "" {
			return instance.Spec.Config.ConfigMapRef.Key
		}
		return "openclaw.json"
	}
	return "openclaw.json"
}

// buildChromiumContainer creates the Chromium sidecar container
func buildChromiumContainer(instance *openclawv1alpha1.OpenClawInstance) corev1.Container {
	repo := instance.Spec.Chromium.Image.Repository
	if repo == "" {
		repo = "ghcr.io/browserless/chromium"
	}

	tag := instance.Spec.Chromium.Image.Tag
	if tag == "" {
		tag = "latest"
	}

	image := repo + ":" + tag
	if instance.Spec.Chromium.Image.Digest != "" {
		image = repo + "@" + instance.Spec.Chromium.Image.Digest
	}

	chromiumMounts := []corev1.VolumeMount{
		{
			Name:      "chromium-tmp",
			MountPath: "/tmp",
		},
		{
			Name:      "chromium-shm",
			MountPath: "/dev/shm",
		},
	}

	var chromiumEnv []corev1.EnvVar

	// Add CA bundle mount and env if configured
	if cab := instance.Spec.Security.CABundle; cab != nil {
		key := cab.Key
		if key == "" {
			key = DefaultCABundleKey
		}
		chromiumMounts = append(chromiumMounts, corev1.VolumeMount{
			Name:      "ca-bundle",
			MountPath: "/etc/ssl/certs/custom-ca-bundle.crt",
			SubPath:   key,
			ReadOnly:  true,
		})
		chromiumEnv = append(chromiumEnv, corev1.EnvVar{
			Name:  "NODE_EXTRA_CA_CERTS",
			Value: "/etc/ssl/certs/custom-ca-bundle.crt",
		})
	}

	return corev1.Container{
		Name:                     "chromium",
		Image:                    image,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: Ptr(false),
			ReadOnlyRootFilesystem:   Ptr(false), // Chromium needs writable dirs for profiles, cache, crash dumps
			RunAsNonRoot:             Ptr(true),
			RunAsUser:                Ptr(int64(999)), // browserless built-in user (blessuser)
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "cdp",
				ContainerPort: ChromiumPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources:    buildChromiumResourceRequirements(instance),
		Env:          chromiumEnv,
		VolumeMounts: chromiumMounts,
	}
}

// buildVolumes creates the volume specs
func buildVolumes(instance *openclawv1alpha1.OpenClawInstance) []corev1.Volume {
	volumes := []corev1.Volume{}

	// Data volume (PVC or emptyDir)
	persistenceEnabled := instance.Spec.Storage.Persistence.Enabled == nil || *instance.Spec.Storage.Persistence.Enabled
	if persistenceEnabled {
		pvcName := PVCName(instance)
		if instance.Spec.Storage.Persistence.ExistingClaim != "" {
			pvcName = instance.Spec.Storage.Persistence.ExistingClaim
		}
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Config volume
	defaultMode := int32(0o644)
	if instance.Spec.Config.ConfigMapRef != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.Config.ConfigMapRef.Name,
					},
					DefaultMode: &defaultMode,
				},
			},
		})
	} else {
		// Always mount the operator-managed ConfigMap (even for vanilla
		// deployments) — it contains gateway.bind=lan for health probes.
		volumes = append(volumes, corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ConfigMapName(instance),
					},
					DefaultMode: &defaultMode,
				},
			},
		})
	}

	// Workspace init volume (ConfigMap with seed files)
	if hasWorkspaceFiles(instance) {
		volumes = append(volumes, corev1.Volume{
			Name: "workspace-init",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: WorkspaceConfigMapName(instance),
					},
					DefaultMode: &defaultMode,
				},
			},
		})
	}

	// Skills-tmp volume for skills init container
	if len(instance.Spec.Skills) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "skills-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Runtime dep tmp volumes
	if instance.Spec.RuntimeDeps.Pnpm {
		volumes = append(volumes, corev1.Volume{
			Name: "pnpm-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	if instance.Spec.RuntimeDeps.Python {
		volumes = append(volumes, corev1.Volume{
			Name: "python-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Init-tmp volume for merge mode (jq writes to /tmp/merged.json) or JSON5 mode (npx writes to /tmp/converted.json)
	if instance.Spec.Config.MergeMode == ConfigMergeModeMerge || instance.Spec.Config.Format == ConfigFormatJSON5 {
		volumes = append(volumes, corev1.Volume{
			Name: "init-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Tmp volume for main container (read-only rootfs needs a writable /tmp)
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Chromium volumes
	if instance.Spec.Chromium.Enabled {
		volumes = append(volumes,
			corev1.Volume{
				Name: "chromium-tmp",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			corev1.Volume{
				Name: "chromium-shm",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium:    corev1.StorageMediumMemory,
						SizeLimit: resource.NewQuantity(1024*1024*1024, resource.BinarySI), // 1Gi
					},
				},
			},
		)
	}

	// CA bundle volume
	if cab := instance.Spec.Security.CABundle; cab != nil {
		if cab.ConfigMapName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "ca-bundle",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cab.ConfigMapName,
						},
						DefaultMode: &defaultMode,
					},
				},
			})
		} else if cab.SecretName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "ca-bundle",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  cab.SecretName,
						DefaultMode: &defaultMode,
					},
				},
			})
		}
	}

	// Custom sidecar volumes
	volumes = append(volumes, instance.Spec.SidecarVolumes...)

	// Extra volumes (available to main container via ExtraVolumeMounts)
	volumes = append(volumes, instance.Spec.ExtraVolumes...)

	return volumes
}

// buildResourceRequirements creates resource requirements for the main container
func buildResourceRequirements(instance *openclawv1alpha1.OpenClawInstance) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	// Requests
	cpuReq := instance.Spec.Resources.Requests.CPU
	if cpuReq == "" {
		cpuReq = "500m"
	}
	req.Requests[corev1.ResourceCPU] = resource.MustParse(cpuReq)

	memReq := instance.Spec.Resources.Requests.Memory
	if memReq == "" {
		memReq = "1Gi"
	}
	req.Requests[corev1.ResourceMemory] = resource.MustParse(memReq)

	// Limits
	cpuLim := instance.Spec.Resources.Limits.CPU
	if cpuLim == "" {
		cpuLim = "2000m"
	}
	req.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLim)

	memLim := instance.Spec.Resources.Limits.Memory
	if memLim == "" {
		memLim = "4Gi"
	}
	req.Limits[corev1.ResourceMemory] = resource.MustParse(memLim)

	return req
}

// buildChromiumResourceRequirements creates resource requirements for the Chromium container
func buildChromiumResourceRequirements(instance *openclawv1alpha1.OpenClawInstance) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	// Requests
	cpuReq := instance.Spec.Chromium.Resources.Requests.CPU
	if cpuReq == "" {
		cpuReq = "250m"
	}
	req.Requests[corev1.ResourceCPU] = resource.MustParse(cpuReq)

	memReq := instance.Spec.Chromium.Resources.Requests.Memory
	if memReq == "" {
		memReq = "512Mi"
	}
	req.Requests[corev1.ResourceMemory] = resource.MustParse(memReq)

	// Limits
	cpuLim := instance.Spec.Chromium.Resources.Limits.CPU
	if cpuLim == "" {
		cpuLim = "1000m"
	}
	req.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLim)

	memLim := instance.Spec.Chromium.Resources.Limits.Memory
	if memLim == "" {
		memLim = "2Gi"
	}
	req.Limits[corev1.ResourceMemory] = resource.MustParse(memLim)

	return req
}

// buildLivenessProbe creates the liveness probe
func buildLivenessProbe(instance *openclawv1alpha1.OpenClawInstance) *corev1.Probe {
	spec := instance.Spec.Probes.Liveness
	if spec != nil && spec.Enabled != nil && !*spec.Enabled {
		return nil
	}

	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(GatewayPort),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	if spec != nil {
		if spec.InitialDelaySeconds != nil {
			probe.InitialDelaySeconds = *spec.InitialDelaySeconds
		}
		if spec.PeriodSeconds != nil {
			probe.PeriodSeconds = *spec.PeriodSeconds
		}
		if spec.TimeoutSeconds != nil {
			probe.TimeoutSeconds = *spec.TimeoutSeconds
		}
		if spec.FailureThreshold != nil {
			probe.FailureThreshold = *spec.FailureThreshold
		}
	}

	return probe
}

// buildReadinessProbe creates the readiness probe
func buildReadinessProbe(instance *openclawv1alpha1.OpenClawInstance) *corev1.Probe {
	spec := instance.Spec.Probes.Readiness
	if spec != nil && spec.Enabled != nil && !*spec.Enabled {
		return nil
	}

	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(GatewayPort),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	if spec != nil {
		if spec.InitialDelaySeconds != nil {
			probe.InitialDelaySeconds = *spec.InitialDelaySeconds
		}
		if spec.PeriodSeconds != nil {
			probe.PeriodSeconds = *spec.PeriodSeconds
		}
		if spec.TimeoutSeconds != nil {
			probe.TimeoutSeconds = *spec.TimeoutSeconds
		}
		if spec.FailureThreshold != nil {
			probe.FailureThreshold = *spec.FailureThreshold
		}
	}

	return probe
}

// buildStartupProbe creates the startup probe
func buildStartupProbe(instance *openclawv1alpha1.OpenClawInstance) *corev1.Probe {
	spec := instance.Spec.Probes.Startup
	if spec != nil && spec.Enabled != nil && !*spec.Enabled {
		return nil
	}

	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(GatewayPort),
			},
		},
		InitialDelaySeconds: 0,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		SuccessThreshold:    1,
		FailureThreshold:    30, // 30 * 5s = 150s startup time
	}

	if spec != nil {
		if spec.InitialDelaySeconds != nil {
			probe.InitialDelaySeconds = *spec.InitialDelaySeconds
		}
		if spec.PeriodSeconds != nil {
			probe.PeriodSeconds = *spec.PeriodSeconds
		}
		if spec.TimeoutSeconds != nil {
			probe.TimeoutSeconds = *spec.TimeoutSeconds
		}
		if spec.FailureThreshold != nil {
			probe.FailureThreshold = *spec.FailureThreshold
		}
	}

	return probe
}

// getPullPolicy returns the image pull policy with defaults
func getPullPolicy(instance *openclawv1alpha1.OpenClawInstance) corev1.PullPolicy {
	if instance.Spec.Image.PullPolicy != "" {
		return instance.Spec.Image.PullPolicy
	}
	return corev1.PullIfNotPresent
}

// calculateConfigHash computes a hash of the config, workspace, and skills for rollout detection.
// Changes to any of these trigger a pod restart.
func calculateConfigHash(instance *openclawv1alpha1.OpenClawInstance) string {
	h := sha256.New()
	configData, _ := json.Marshal(instance.Spec.Config)
	h.Write(configData)
	if instance.Spec.Workspace != nil {
		wsData, _ := json.Marshal(instance.Spec.Workspace)
		h.Write(wsData)
	}
	if len(instance.Spec.Skills) > 0 {
		skillsData, _ := json.Marshal(instance.Spec.Skills)
		h.Write(skillsData)
	}
	if len(instance.Spec.InitContainers) > 0 {
		icData, _ := json.Marshal(instance.Spec.InitContainers)
		h.Write(icData)
	}
	if instance.Spec.RuntimeDeps.Pnpm || instance.Spec.RuntimeDeps.Python {
		rdData, _ := json.Marshal(instance.Spec.RuntimeDeps)
		h.Write(rdData)
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}
