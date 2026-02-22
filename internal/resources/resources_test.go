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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// newTestInstance creates a minimal OpenClawInstance for testing.
func newTestInstance(name string) *openclawv1alpha1.OpenClawInstance {
	return &openclawv1alpha1.OpenClawInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
		},
		Spec: openclawv1alpha1.OpenClawInstanceSpec{},
	}
}

// ---------------------------------------------------------------------------
// common.go tests
// ---------------------------------------------------------------------------

func TestLabels(t *testing.T) {
	instance := newTestInstance("my-instance")
	labels := Labels(instance)

	expected := map[string]string{
		"app.kubernetes.io/name":       "openclaw",
		"app.kubernetes.io/instance":   "my-instance",
		"app.kubernetes.io/managed-by": "openclaw-operator",
	}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(labels))
	}
	for k, v := range expected {
		if labels[k] != v {
			t.Errorf("label %q: expected %q, got %q", k, v, labels[k])
		}
	}
}

func TestSelectorLabels(t *testing.T) {
	instance := newTestInstance("my-instance")
	labels := SelectorLabels(instance)

	expected := map[string]string{
		"app.kubernetes.io/name":     "openclaw",
		"app.kubernetes.io/instance": "my-instance",
	}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d selector labels, got %d", len(expected), len(labels))
	}
	for k, v := range expected {
		if labels[k] != v {
			t.Errorf("selector label %q: expected %q, got %q", k, v, labels[k])
		}
	}
}

func TestSelectorLabels_SubsetOfLabels(t *testing.T) {
	instance := newTestInstance("test")
	allLabels := Labels(instance)
	selectorLabels := SelectorLabels(instance)

	for k, v := range selectorLabels {
		if allLabels[k] != v {
			t.Errorf("selector label %q=%q is not present in full labels", k, v)
		}
	}

	if len(selectorLabels) >= len(allLabels) {
		t.Error("selector labels should be a strict subset of full labels")
	}
}

func TestGetImage(t *testing.T) {
	tests := []struct {
		name     string
		image    openclawv1alpha1.ImageSpec
		expected string
	}{
		{
			name:     "defaults",
			image:    openclawv1alpha1.ImageSpec{},
			expected: "ghcr.io/openclaw/openclaw:latest",
		},
		{
			name: "custom repo and tag",
			image: openclawv1alpha1.ImageSpec{
				Repository: "my-registry.io/openclaw",
				Tag:        "v1.2.3",
			},
			expected: "my-registry.io/openclaw:v1.2.3",
		},
		{
			name: "digest takes precedence over tag",
			image: openclawv1alpha1.ImageSpec{
				Repository: "my-registry.io/openclaw",
				Tag:        "v1.2.3",
				Digest:     "sha256:abc123",
			},
			expected: "my-registry.io/openclaw@sha256:abc123",
		},
		{
			name: "digest with default repo",
			image: openclawv1alpha1.ImageSpec{
				Digest: "sha256:def456",
			},
			expected: "ghcr.io/openclaw/openclaw@sha256:def456",
		},
		{
			name: "custom repo with default tag",
			image: openclawv1alpha1.ImageSpec{
				Repository: "custom.io/img",
			},
			expected: "custom.io/img:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := newTestInstance("test")
			instance.Spec.Image = tt.image

			got := GetImage(instance)
			if got != tt.expected {
				t.Errorf("GetImage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNameHelpers(t *testing.T) {
	instance := newTestInstance("foo")

	tests := []struct {
		name     string
		fn       func(*openclawv1alpha1.OpenClawInstance) string
		expected string
	}{
		{"StatefulSetName", StatefulSetName, "foo"},
		{"DeploymentName", DeploymentName, "foo"},
		{"ServiceName", ServiceName, "foo"},
		{"RoleName", RoleName, "foo"},
		{"RoleBindingName", RoleBindingName, "foo"},
		{"ConfigMapName", ConfigMapName, "foo-config"},
		{"PVCName", PVCName, "foo-data"},
		{"NetworkPolicyName", NetworkPolicyName, "foo"},
		{"PDBName", PDBName, "foo"},
		{"IngressName", IngressName, "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(instance)
			if got != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestServiceAccountName_Default(t *testing.T) {
	instance := newTestInstance("my-inst")
	if got := ServiceAccountName(instance); got != "my-inst" {
		t.Errorf("ServiceAccountName() = %q, want %q", got, "my-inst")
	}
}

func TestServiceAccountName_Custom(t *testing.T) {
	instance := newTestInstance("my-inst")
	instance.Spec.Security.RBAC.ServiceAccountName = "custom-sa"
	if got := ServiceAccountName(instance); got != "custom-sa" {
		t.Errorf("ServiceAccountName() = %q, want %q", got, "custom-sa")
	}
}

func TestPtr(t *testing.T) {
	intVal := Ptr(int32(42))
	if *intVal != 42 {
		t.Errorf("Ptr(42) = %d, want 42", *intVal)
	}

	boolVal := Ptr(true)
	if !*boolVal {
		t.Error("Ptr(true) should be true")
	}

	strVal := Ptr("hello")
	if *strVal != "hello" {
		t.Errorf("Ptr(hello) = %q, want %q", *strVal, "hello")
	}
}

// ---------------------------------------------------------------------------
// deployment.go tests
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_Defaults(t *testing.T) {
	instance := newTestInstance("test-deploy")
	sts := BuildStatefulSet(instance)

	// ObjectMeta
	if sts.Name != "test-deploy" {
		t.Errorf("statefulset name = %q, want %q", sts.Name, "test-deploy")
	}
	if sts.Namespace != "test-ns" {
		t.Errorf("statefulset namespace = %q, want %q", sts.Namespace, "test-ns")
	}

	// Labels present
	if sts.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("statefulset missing app.kubernetes.io/name label")
	}

	// Replicas
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 1 {
		t.Errorf("replicas = %v, want 1", sts.Spec.Replicas)
	}

	// StatefulSet-specific fields
	if sts.Spec.ServiceName != "test-deploy" {
		t.Errorf("serviceName = %q, want %q", sts.Spec.ServiceName, "test-deploy")
	}
	if sts.Spec.PodManagementPolicy != appsv1.ParallelPodManagement {
		t.Errorf("podManagementPolicy = %v, want Parallel", sts.Spec.PodManagementPolicy)
	}
	if sts.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		t.Errorf("updateStrategy = %v, want RollingUpdate", sts.Spec.UpdateStrategy.Type)
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy == nil {
		t.Fatal("persistentVolumeClaimRetentionPolicy should be set (prevents spec drift)")
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted != appsv1.RetainPersistentVolumeClaimRetentionPolicyType {
		t.Errorf("pvcRetentionPolicy.whenDeleted = %v, want Retain", sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
	}
	if sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled != appsv1.RetainPersistentVolumeClaimRetentionPolicyType {
		t.Errorf("pvcRetentionPolicy.whenScaled = %v, want Retain", sts.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
	}

	// Selector labels
	sel := sts.Spec.Selector.MatchLabels
	if sel["app.kubernetes.io/name"] != "openclaw" || sel["app.kubernetes.io/instance"] != "test-deploy" {
		t.Error("selector labels do not match expected values")
	}

	// Config hash annotation
	ann := sts.Spec.Template.Annotations
	if _, ok := ann["openclaw.rocks/config-hash"]; !ok {
		t.Error("config-hash annotation missing from pod template")
	}

	// Pod security context
	psc := sts.Spec.Template.Spec.SecurityContext
	if psc == nil {
		t.Fatal("pod security context is nil")
	}
	if psc.RunAsNonRoot == nil || !*psc.RunAsNonRoot {
		t.Error("pod security context: runAsNonRoot should be true")
	}
	if psc.RunAsUser == nil || *psc.RunAsUser != 1000 {
		t.Errorf("pod security context: runAsUser = %v, want 1000", psc.RunAsUser)
	}
	if psc.RunAsGroup == nil || *psc.RunAsGroup != 1000 {
		t.Errorf("pod security context: runAsGroup = %v, want 1000", psc.RunAsGroup)
	}
	if psc.FSGroup == nil || *psc.FSGroup != 1000 {
		t.Errorf("pod security context: fsGroup = %v, want 1000", psc.FSGroup)
	}
	if psc.SeccompProfile == nil || psc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("pod security context: seccomp profile should be RuntimeDefault")
	}

	// Containers
	containers := sts.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	main := containers[0]
	if main.Name != "openclaw" {
		t.Errorf("main container name = %q, want %q", main.Name, "openclaw")
	}
	if main.Image != "ghcr.io/openclaw/openclaw:latest" {
		t.Errorf("main container image = %q, want default image", main.Image)
	}
	if main.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("image pull policy = %v, want IfNotPresent", main.ImagePullPolicy)
	}

	// Container security context
	csc := main.SecurityContext
	if csc == nil {
		t.Fatal("container security context is nil")
	}
	if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation {
		t.Error("container security context: allowPrivilegeEscalation should be false")
	}
	if csc.RunAsNonRoot == nil || !*csc.RunAsNonRoot {
		t.Error("container security context: runAsNonRoot should be true")
	}
	if csc.Capabilities == nil || len(csc.Capabilities.Drop) != 1 || csc.Capabilities.Drop[0] != "ALL" {
		t.Error("container security context: capabilities should drop ALL")
	}
	if csc.SeccompProfile == nil || csc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("container security context: seccomp profile should be RuntimeDefault")
	}

	// HOME env var must be set to match the mount path
	if len(main.Env) < 1 || main.Env[0].Name != "HOME" || main.Env[0].Value != "/home/openclaw" {
		t.Error("HOME env var should be set to /home/openclaw")
	}

	// Ports
	if len(main.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(main.Ports))
	}
	assertContainerPort(t, main.Ports, "gateway", GatewayPort)
	assertContainerPort(t, main.Ports, "canvas", CanvasPort)

	// Default resources
	cpuReq := main.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "500m" {
		t.Errorf("cpu request = %v, want 500m", cpuReq.String())
	}
	memReq := main.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Errorf("memory request = %v, want 1Gi", memReq.String())
	}
	cpuLim := main.Resources.Limits[corev1.ResourceCPU]
	if cpuLim.String() != "2" {
		t.Errorf("cpu limit = %v, want 2 (2000m)", cpuLim.String())
	}
	memLim := main.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("4Gi")) != 0 {
		t.Errorf("memory limit = %v, want 4Gi", memLim.String())
	}

	// Probes
	if main.LivenessProbe == nil {
		t.Error("liveness probe should not be nil by default")
	}
	if main.ReadinessProbe == nil {
		t.Error("readiness probe should not be nil by default")
	}
	if main.StartupProbe == nil {
		t.Error("startup probe should not be nil by default")
	}

	// Liveness probe defaults
	if main.LivenessProbe.TCPSocket == nil {
		t.Fatal("liveness probe should use TCPSocket")
	}
	if main.LivenessProbe.TCPSocket.Port.IntValue() != GatewayPort {
		t.Errorf("liveness probe port = %d, want %d", main.LivenessProbe.TCPSocket.Port.IntValue(), GatewayPort)
	}
	if main.LivenessProbe.InitialDelaySeconds != 30 {
		t.Errorf("liveness probe initialDelaySeconds = %d, want 30", main.LivenessProbe.InitialDelaySeconds)
	}
	if main.LivenessProbe.PeriodSeconds != 10 {
		t.Errorf("liveness probe periodSeconds = %d, want 10", main.LivenessProbe.PeriodSeconds)
	}

	// Readiness probe defaults
	if main.ReadinessProbe.InitialDelaySeconds != 5 {
		t.Errorf("readiness probe initialDelaySeconds = %d, want 5", main.ReadinessProbe.InitialDelaySeconds)
	}

	// Startup probe defaults
	if main.StartupProbe.FailureThreshold != 30 {
		t.Errorf("startup probe failureThreshold = %d, want 30", main.StartupProbe.FailureThreshold)
	}

	// Data volume mount
	assertVolumeMount(t, main.VolumeMounts, "data", "/home/openclaw/.openclaw")

	// Volumes - default persistence is enabled, so data volume should be PVC
	volumes := sts.Spec.Template.Spec.Volumes
	dataVol := findVolume(volumes, "data")
	if dataVol == nil {
		t.Fatal("data volume not found")
	}
	if dataVol.PersistentVolumeClaim == nil {
		t.Error("data volume should use PVC by default")
	}
	if dataVol.PersistentVolumeClaim.ClaimName != "test-deploy-data" {
		t.Errorf("PVC claim name = %q, want %q", dataVol.PersistentVolumeClaim.ClaimName, "test-deploy-data")
	}
}

func TestBuildStatefulSet_WithChromium(t *testing.T) {
	instance := newTestInstance("chromium-test")
	instance.Spec.Chromium.Enabled = true

	sts := BuildStatefulSet(instance)
	containers := sts.Spec.Template.Spec.Containers

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers with chromium enabled, got %d", len(containers))
	}

	// Find chromium container
	var chromium *corev1.Container
	for i := range containers {
		if containers[i].Name == "chromium" {
			chromium = &containers[i]
			break
		}
	}
	if chromium == nil {
		t.Fatal("chromium container not found")
	}

	// Main container should have CHROMIUM_URL env var
	mainContainer := containers[0]
	foundChromiumURL := false
	for _, env := range mainContainer.Env {
		if env.Name == "CHROMIUM_URL" {
			foundChromiumURL = true
			if env.Value != "http://localhost:3000" {
				t.Errorf("CHROMIUM_URL = %q, want %q", env.Value, "http://localhost:3000")
			}
			break
		}
	}
	if !foundChromiumURL {
		t.Error("main container should have CHROMIUM_URL env var when chromium is enabled")
	}

	// Chromium image defaults
	if chromium.Image != "ghcr.io/browserless/chromium:latest" {
		t.Errorf("chromium image = %q, want default", chromium.Image)
	}

	// Chromium port
	if len(chromium.Ports) != 1 {
		t.Fatalf("chromium container should have 1 port, got %d", len(chromium.Ports))
	}
	if chromium.Ports[0].ContainerPort != ChromiumPort {
		t.Errorf("chromium port = %d, want %d", chromium.Ports[0].ContainerPort, ChromiumPort)
	}
	if chromium.Ports[0].Name != "cdp" {
		t.Errorf("chromium port name = %q, want %q", chromium.Ports[0].Name, "cdp")
	}

	// Chromium security context
	csc := chromium.SecurityContext
	if csc == nil {
		t.Fatal("chromium security context is nil")
	}
	if csc.ReadOnlyRootFilesystem == nil || *csc.ReadOnlyRootFilesystem {
		t.Error("chromium: readOnlyRootFilesystem should be false (Chromium needs writable dirs)")
	}
	if csc.RunAsUser == nil || *csc.RunAsUser != 999 {
		t.Errorf("chromium: runAsUser = %v, want 999 (browserless blessuser)", csc.RunAsUser)
	}

	// Chromium resource defaults
	cpuReq := chromium.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "250m" {
		t.Errorf("chromium cpu request = %v, want 250m", cpuReq.String())
	}
	memReq := chromium.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("512Mi")) != 0 {
		t.Errorf("chromium memory request = %v, want 512Mi", memReq.String())
	}

	// Chromium volume mounts
	assertVolumeMount(t, chromium.VolumeMounts, "chromium-tmp", "/tmp")
	assertVolumeMount(t, chromium.VolumeMounts, "chromium-shm", "/dev/shm")

	// Volumes - check chromium-specific volumes exist
	volumes := sts.Spec.Template.Spec.Volumes
	tmpVol := findVolume(volumes, "chromium-tmp")
	if tmpVol == nil {
		t.Fatal("chromium-tmp volume not found")
	}
	if tmpVol.EmptyDir == nil {
		t.Error("chromium-tmp should be emptyDir")
	}

	shmVol := findVolume(volumes, "chromium-shm")
	if shmVol == nil {
		t.Fatal("chromium-shm volume not found")
	}
	if shmVol.EmptyDir == nil {
		t.Fatal("chromium-shm should be emptyDir")
	}
	if shmVol.EmptyDir.Medium != corev1.StorageMediumMemory {
		t.Errorf("chromium-shm medium = %v, want Memory", shmVol.EmptyDir.Medium)
	}
	expectedShmSize := resource.NewQuantity(1024*1024*1024, resource.BinarySI) // 1Gi
	if shmVol.EmptyDir.SizeLimit == nil {
		t.Fatal("chromium-shm sizeLimit is nil")
	}
	if shmVol.EmptyDir.SizeLimit.Cmp(*expectedShmSize) != 0 {
		t.Errorf("chromium-shm sizeLimit = %v, want 1Gi", shmVol.EmptyDir.SizeLimit.String())
	}
}

func TestBuildStatefulSet_CustomResources(t *testing.T) {
	instance := newTestInstance("res-test")
	instance.Spec.Resources = openclawv1alpha1.ResourcesSpec{
		Requests: openclawv1alpha1.ResourceList{
			CPU:    "1",
			Memory: "2Gi",
		},
		Limits: openclawv1alpha1.ResourceList{
			CPU:    "4",
			Memory: "8Gi",
		},
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	cpuReq := main.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "1" {
		t.Errorf("cpu request = %v, want 1", cpuReq.String())
	}
	memReq := main.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("2Gi")) != 0 {
		t.Errorf("memory request = %v, want 2Gi", memReq.String())
	}
	cpuLim := main.Resources.Limits[corev1.ResourceCPU]
	if cpuLim.String() != "4" {
		t.Errorf("cpu limit = %v, want 4", cpuLim.String())
	}
	memLim := main.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("8Gi")) != 0 {
		t.Errorf("memory limit = %v, want 8Gi", memLim.String())
	}
}

func TestBuildStatefulSet_ImageDigest(t *testing.T) {
	instance := newTestInstance("digest-test")
	instance.Spec.Image = openclawv1alpha1.ImageSpec{
		Repository: "my-registry.io/openclaw",
		Tag:        "v1.0.0",
		Digest:     "sha256:abcdef1234567890",
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	expected := "my-registry.io/openclaw@sha256:abcdef1234567890"
	if main.Image != expected {
		t.Errorf("image = %q, want %q", main.Image, expected)
	}
}

func TestBuildStatefulSet_ProbesDisabled(t *testing.T) {
	instance := newTestInstance("probes-disabled")
	instance.Spec.Probes = openclawv1alpha1.ProbesSpec{
		Liveness: &openclawv1alpha1.ProbeSpec{
			Enabled: Ptr(false),
		},
		Readiness: &openclawv1alpha1.ProbeSpec{
			Enabled: Ptr(false),
		},
		Startup: &openclawv1alpha1.ProbeSpec{
			Enabled: Ptr(false),
		},
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.LivenessProbe != nil {
		t.Error("liveness probe should be nil when disabled")
	}
	if main.ReadinessProbe != nil {
		t.Error("readiness probe should be nil when disabled")
	}
	if main.StartupProbe != nil {
		t.Error("startup probe should be nil when disabled")
	}
}

func TestBuildStatefulSet_CustomProbeValues(t *testing.T) {
	instance := newTestInstance("probes-custom")
	instance.Spec.Probes = openclawv1alpha1.ProbesSpec{
		Liveness: &openclawv1alpha1.ProbeSpec{
			InitialDelaySeconds: Ptr(int32(60)),
			PeriodSeconds:       Ptr(int32(20)),
			TimeoutSeconds:      Ptr(int32(10)),
			FailureThreshold:    Ptr(int32(5)),
		},
	}

	sts := BuildStatefulSet(instance)
	probe := sts.Spec.Template.Spec.Containers[0].LivenessProbe

	if probe == nil {
		t.Fatal("liveness probe should not be nil")
	}
	if probe.InitialDelaySeconds != 60 {
		t.Errorf("liveness initialDelaySeconds = %d, want 60", probe.InitialDelaySeconds)
	}
	if probe.PeriodSeconds != 20 {
		t.Errorf("liveness periodSeconds = %d, want 20", probe.PeriodSeconds)
	}
	if probe.TimeoutSeconds != 10 {
		t.Errorf("liveness timeoutSeconds = %d, want 10", probe.TimeoutSeconds)
	}
	if probe.FailureThreshold != 5 {
		t.Errorf("liveness failureThreshold = %d, want 5", probe.FailureThreshold)
	}
}

func TestBuildStatefulSet_PersistenceDisabled(t *testing.T) {
	instance := newTestInstance("no-pvc")
	instance.Spec.Storage.Persistence.Enabled = Ptr(false)

	sts := BuildStatefulSet(instance)
	volumes := sts.Spec.Template.Spec.Volumes

	dataVol := findVolume(volumes, "data")
	if dataVol == nil {
		t.Fatal("data volume not found")
	}
	if dataVol.EmptyDir == nil {
		t.Error("data volume should be emptyDir when persistence is disabled")
	}
	if dataVol.PersistentVolumeClaim != nil {
		t.Error("data volume should not use PVC when persistence is disabled")
	}
}

func TestBuildStatefulSet_ExistingClaim(t *testing.T) {
	instance := newTestInstance("existing-pvc")
	instance.Spec.Storage.Persistence.ExistingClaim = "my-existing-pvc"

	sts := BuildStatefulSet(instance)
	volumes := sts.Spec.Template.Spec.Volumes

	dataVol := findVolume(volumes, "data")
	if dataVol == nil {
		t.Fatal("data volume not found")
	}
	if dataVol.PersistentVolumeClaim == nil {
		t.Fatal("data volume should use PVC")
	}
	if dataVol.PersistentVolumeClaim.ClaimName != "my-existing-pvc" {
		t.Errorf("PVC claim name = %q, want %q", dataVol.PersistentVolumeClaim.ClaimName, "my-existing-pvc")
	}
}

func TestBuildStatefulSet_ConfigVolume_RawConfig(t *testing.T) {
	instance := newTestInstance("raw-cfg")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"key":"value"}`),
		},
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	// Main container should have config volume mounted read-only at /operator-config
	// for the postStart lifecycle hook to restore config on container restarts.
	assertVolumeMount(t, main.VolumeMounts, "config", "/operator-config")
	for _, vm := range main.VolumeMounts {
		if vm.Name == "config" && !vm.ReadOnly {
			t.Error("config volume mount in main container should be read-only")
		}
	}

	// Init container should copy config from ConfigMap to data volume
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(initContainers))
	}
	initC := initContainers[0]
	if initC.Name != "init-config" {
		t.Errorf("init container name = %q, want %q", initC.Name, "init-config")
	}
	assertVolumeMount(t, initC.VolumeMounts, "data", "/data")
	assertVolumeMount(t, initC.VolumeMounts, "config", "/config")

	// Init container must set SeccompProfile to prevent spec drift
	if initC.SecurityContext == nil {
		t.Fatal("init-config security context is nil")
	}
	if initC.SecurityContext.SeccompProfile == nil || initC.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("init-config container: seccomp profile should be RuntimeDefault (prevents spec drift)")
	}

	// Should have config volume pointing to managed configmap
	volumes := sts.Spec.Template.Spec.Volumes
	cfgVol := findVolume(volumes, "config")
	if cfgVol == nil {
		t.Fatal("config volume not found")
	}
	if cfgVol.ConfigMap == nil {
		t.Fatal("config volume should use ConfigMap")
	}
	if cfgVol.ConfigMap.Name != "raw-cfg-config" {
		t.Errorf("config volume configmap name = %q, want %q", cfgVol.ConfigMap.Name, "raw-cfg-config")
	}
}

func TestBuildStatefulSet_ConfigVolume_ConfigMapRef(t *testing.T) {
	instance := newTestInstance("ref-cfg")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "external-config",
		Key:  "my-config.json",
	}

	sts := BuildStatefulSet(instance)

	// Init container should copy openclaw.json (operator-managed key) from ConfigMap to data volume.
	// The controller reads the external CM and writes enriched content into the
	// operator-managed CM under "openclaw.json", so the init container always uses that key.
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(initContainers))
	}
	initC := initContainers[0]
	assertVolumeMount(t, initC.VolumeMounts, "data", "/data")
	assertVolumeMount(t, initC.VolumeMounts, "config", "/config")

	// Verify the command copies openclaw.json (not the custom key)
	expectedCmd := "cp /config/'openclaw.json' /data/openclaw.json"
	if len(initC.Command) != 3 || initC.Command[2] != expectedCmd {
		t.Errorf("init container command = %v, want sh -c %q", initC.Command, expectedCmd)
	}

	// Volume should reference the operator-managed ConfigMap (not the external one)
	volumes := sts.Spec.Template.Spec.Volumes
	cfgVol := findVolume(volumes, "config")
	if cfgVol == nil {
		t.Fatal("config volume not found")
	}
	if cfgVol.ConfigMap.Name != "ref-cfg-config" {
		t.Errorf("config volume configmap name = %q, want %q", cfgVol.ConfigMap.Name, "ref-cfg-config")
	}
}

func TestBuildStatefulSet_ConfigMapRef_DefaultKey(t *testing.T) {
	instance := newTestInstance("ref-default-key")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "external-config",
		// Key not set - operator-managed CM always uses "openclaw.json"
	}

	sts := BuildStatefulSet(instance)

	// Init container should use "openclaw.json" (operator-managed key)
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) != 1 {
		t.Fatalf("expected 1 init container, got %d", len(initContainers))
	}
	expectedCmd := "cp /config/'openclaw.json' /data/openclaw.json"
	if initContainers[0].Command[2] != expectedCmd {
		t.Errorf("init container command = %q, want %q", initContainers[0].Command[2], expectedCmd)
	}

	// Volume should reference the operator-managed ConfigMap
	cfgVol := findVolume(sts.Spec.Template.Spec.Volumes, "config")
	if cfgVol == nil {
		t.Fatal("config volume not found")
	}
	if cfgVol.ConfigMap.Name != "ref-default-key-config" {
		t.Errorf("config volume configmap name = %q, want %q", cfgVol.ConfigMap.Name, "ref-default-key-config")
	}
}

func TestBuildStatefulSet_VanillaDeployment_HasInitContainer(t *testing.T) {
	instance := newTestInstance("no-config")
	// No config set at all — vanilla deployment

	sts := BuildStatefulSet(instance)

	// Vanilla deployments should still get an init container (gateway.bind=lan)
	if len(sts.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("expected 1 init container for vanilla deployment, got %d", len(sts.Spec.Template.Spec.InitContainers))
	}
	if sts.Spec.Template.Spec.InitContainers[0].Name != "init-config" {
		t.Errorf("init container name = %q, want %q", sts.Spec.Template.Spec.InitContainers[0].Name, "init-config")
	}

	// Verify config volume is present
	configVol := findVolume(sts.Spec.Template.Spec.Volumes, "config")
	if configVol == nil {
		t.Fatal("config volume not found for vanilla deployment")
	}
	if configVol.ConfigMap == nil {
		t.Fatal("config volume should use ConfigMap")
	}
	if configVol.ConfigMap.Name != "no-config-config" {
		t.Errorf("config volume ConfigMap name = %q, want %q", configVol.ConfigMap.Name, "no-config-config")
	}
}

// ---------------------------------------------------------------------------
// PostStart lifecycle hook tests (config restore on container restart)
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_PostStart_OverwriteMode(t *testing.T) {
	instance := newTestInstance("poststart-overwrite")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.Lifecycle == nil || main.Lifecycle.PostStart == nil {
		t.Fatal("expected postStart lifecycle hook on main container")
	}

	cmd := main.Lifecycle.PostStart.Exec.Command
	if len(cmd) != 3 || cmd[0] != "sh" || cmd[1] != "-c" {
		t.Fatalf("expected sh -c command, got %v", cmd)
	}
	expected := "cp /operator-config/openclaw.json /home/openclaw/.openclaw/openclaw.json"
	if cmd[2] != expected {
		t.Errorf("postStart command = %q, want %q", cmd[2], expected)
	}
}

func TestBuildStatefulSet_PostStart_MergeMode(t *testing.T) {
	instance := newTestInstance("poststart-merge")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = ConfigMergeModeMerge

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.Lifecycle == nil || main.Lifecycle.PostStart == nil {
		t.Fatal("expected postStart lifecycle hook for merge mode")
	}

	cmd := main.Lifecycle.PostStart.Exec.Command
	if len(cmd) != 3 || cmd[0] != "sh" || cmd[1] != "-c" {
		t.Fatalf("expected sh -c command, got %v", cmd)
	}

	// Merge mode should use a Node.js deep-merge script
	if !strings.Contains(cmd[2], "node -e") {
		t.Errorf("merge mode postStart should use node, got %q", cmd[2])
	}
	if !strings.Contains(cmd[2], "/operator-config/openclaw.json") {
		t.Errorf("merge mode postStart should reference operator-config path, got %q", cmd[2])
	}
}

func TestBuildStatefulSet_PostStart_JSON5Mode_NoHook(t *testing.T) {
	instance := newTestInstance("poststart-json5")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.Format = ConfigFormatJSON5

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	// JSON5 mode can't use postStart (needs npx, too slow)
	if main.Lifecycle != nil && main.Lifecycle.PostStart != nil {
		t.Error("JSON5 mode should not have a postStart hook (npx too slow)")
	}
}

func TestBuildStatefulSet_PostStart_ConfigMapRef(t *testing.T) {
	instance := newTestInstance("poststart-ref")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "external-config",
		Key:  "my-config.json",
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.Lifecycle == nil || main.Lifecycle.PostStart == nil {
		t.Fatal("expected postStart lifecycle hook with configMapRef")
	}

	// PostStart should use openclaw.json (operator-managed key), not the
	// external CM's custom key, because the operator-managed CM always
	// stores enriched config under "openclaw.json".
	cmd := main.Lifecycle.PostStart.Exec.Command[2]
	expected := "cp /operator-config/openclaw.json /home/openclaw/.openclaw/openclaw.json"
	if cmd != expected {
		t.Errorf("postStart command = %q, want %q", cmd, expected)
	}
}

func TestBuildStatefulSet_PostStart_VanillaDeployment(t *testing.T) {
	instance := newTestInstance("poststart-vanilla")
	// No config set at all - vanilla deployment still gets postStart
	// because operator always creates a ConfigMap with gateway.bind=lan

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.Lifecycle == nil || main.Lifecycle.PostStart == nil {
		t.Fatal("vanilla deployment should still have postStart hook")
	}

	cmd := main.Lifecycle.PostStart.Exec.Command[2]
	if !strings.Contains(cmd, "cp /operator-config/openclaw.json") {
		t.Errorf("vanilla postStart should copy config, got %q", cmd)
	}
}

func TestBuildStatefulSet_ServiceAccountName(t *testing.T) {
	instance := newTestInstance("sa-test")
	sts := BuildStatefulSet(instance)
	if sts.Spec.Template.Spec.ServiceAccountName != "sa-test" {
		t.Errorf("serviceAccountName = %q, want %q", sts.Spec.Template.Spec.ServiceAccountName, "sa-test")
	}
}

func TestBuildStatefulSet_AutomountServiceAccountTokenDisabled(t *testing.T) {
	instance := newTestInstance("automount-test")
	sts := BuildStatefulSet(instance)
	token := sts.Spec.Template.Spec.AutomountServiceAccountToken
	if token == nil || *token != false {
		t.Errorf("AutomountServiceAccountToken = %v, want false", token)
	}
}

func TestBuildStatefulSet_ImagePullSecrets(t *testing.T) {
	instance := newTestInstance("pull-secrets")
	instance.Spec.Image.PullSecrets = []corev1.LocalObjectReference{
		{Name: "my-secret"},
		{Name: "other-secret"},
	}

	sts := BuildStatefulSet(instance)
	secrets := sts.Spec.Template.Spec.ImagePullSecrets
	if len(secrets) != 2 {
		t.Fatalf("expected 2 pull secrets, got %d", len(secrets))
	}
	if secrets[0].Name != "my-secret" {
		t.Errorf("first pull secret = %q, want %q", secrets[0].Name, "my-secret")
	}
	if secrets[1].Name != "other-secret" {
		t.Errorf("second pull secret = %q, want %q", secrets[1].Name, "other-secret")
	}
}

func TestBuildStatefulSet_ChromiumCustomImage(t *testing.T) {
	instance := newTestInstance("chromium-custom")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Chromium.Image = openclawv1alpha1.ChromiumImageSpec{
		Repository: "my-registry.io/chromium",
		Tag:        "v120",
	}

	sts := BuildStatefulSet(instance)
	var chromium *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "chromium" {
			chromium = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if chromium == nil {
		t.Fatal("chromium container not found")
	}
	if chromium.Image != "my-registry.io/chromium:v120" {
		t.Errorf("chromium image = %q, want %q", chromium.Image, "my-registry.io/chromium:v120")
	}
}

func TestBuildStatefulSet_ChromiumDigest(t *testing.T) {
	instance := newTestInstance("chromium-digest")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Chromium.Image = openclawv1alpha1.ChromiumImageSpec{
		Repository: "my-registry.io/chromium",
		Tag:        "v120",
		Digest:     "sha256:chromiumhash",
	}

	sts := BuildStatefulSet(instance)
	var chromium *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "chromium" {
			chromium = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if chromium == nil {
		t.Fatal("chromium container not found")
	}
	expected := "my-registry.io/chromium@sha256:chromiumhash"
	if chromium.Image != expected {
		t.Errorf("chromium image = %q, want %q", chromium.Image, expected)
	}
}

func TestBuildStatefulSet_NodeSelectorAndTolerations(t *testing.T) {
	instance := newTestInstance("scheduling")
	instance.Spec.Availability.NodeSelector = map[string]string{
		"node-type": "gpu",
	}
	instance.Spec.Availability.Tolerations = []corev1.Toleration{
		{
			Key:      "gpu",
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}

	sts := BuildStatefulSet(instance)
	podSpec := sts.Spec.Template.Spec

	if podSpec.NodeSelector["node-type"] != "gpu" {
		t.Error("nodeSelector not applied")
	}
	if len(podSpec.Tolerations) != 1 || podSpec.Tolerations[0].Key != "gpu" {
		t.Error("tolerations not applied")
	}
}

func TestBuildStatefulSet_EnvAndEnvFrom(t *testing.T) {
	instance := newTestInstance("env-test")
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "MY_VAR", Value: "my-value"},
	}
	instance.Spec.EnvFrom = []corev1.EnvFromSource{
		{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-keys"},
			},
		},
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if len(main.Env) != 3 || main.Env[0].Name != "HOME" || main.Env[1].Name != "OPENCLAW_DISABLE_BONJOUR" || main.Env[2].Name != "MY_VAR" {
		t.Errorf("env vars should include HOME, OPENCLAW_DISABLE_BONJOUR, then user-defined vars, got %v", envNames(main.Env))
	}
	if len(main.EnvFrom) != 1 || main.EnvFrom[0].SecretRef.Name != "api-keys" {
		t.Error("envFrom not passed through")
	}
}

// ---------------------------------------------------------------------------
// service.go tests
// ---------------------------------------------------------------------------

func TestBuildService_Default(t *testing.T) {
	instance := newTestInstance("svc-test")
	svc := BuildService(instance)

	if svc.Name != "svc-test" {
		t.Errorf("service name = %q, want %q", svc.Name, "svc-test")
	}
	if svc.Namespace != "test-ns" {
		t.Errorf("service namespace = %q, want %q", svc.Namespace, "test-ns")
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("service type = %v, want ClusterIP", svc.Spec.Type)
	}

	// Labels
	if svc.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("service missing app label")
	}

	// Selector
	sel := svc.Spec.Selector
	if sel["app.kubernetes.io/name"] != "openclaw" || sel["app.kubernetes.io/instance"] != "svc-test" {
		t.Error("service selector does not match expected values")
	}

	// Ports - should have gateway and canvas
	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(svc.Spec.Ports))
	}

	assertServicePort(t, svc.Spec.Ports, "gateway", int32(GatewayPort))
	assertServicePort(t, svc.Spec.Ports, "canvas", int32(CanvasPort))
}

func TestBuildService_WithChromium(t *testing.T) {
	instance := newTestInstance("svc-chromium")
	instance.Spec.Chromium.Enabled = true

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 3 {
		t.Fatalf("expected 3 ports with chromium, got %d", len(svc.Spec.Ports))
	}

	assertServicePort(t, svc.Spec.Ports, "gateway", int32(GatewayPort))
	assertServicePort(t, svc.Spec.Ports, "canvas", int32(CanvasPort))
	assertServicePort(t, svc.Spec.Ports, "chromium", int32(ChromiumPort))
}

func TestBuildService_LoadBalancer(t *testing.T) {
	instance := newTestInstance("svc-lb")
	instance.Spec.Networking.Service.Type = corev1.ServiceTypeLoadBalancer

	svc := BuildService(instance)

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Errorf("service type = %v, want LoadBalancer", svc.Spec.Type)
	}
}

func TestBuildService_NodePort(t *testing.T) {
	instance := newTestInstance("svc-np")
	instance.Spec.Networking.Service.Type = corev1.ServiceTypeNodePort

	svc := BuildService(instance)

	if svc.Spec.Type != corev1.ServiceTypeNodePort {
		t.Errorf("service type = %v, want NodePort", svc.Spec.Type)
	}
}

func TestBuildService_CustomAnnotations(t *testing.T) {
	instance := newTestInstance("svc-ann")
	instance.Spec.Networking.Service.Annotations = map[string]string{
		"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
	}

	svc := BuildService(instance)

	if svc.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] != "nlb" {
		t.Error("service annotations not applied")
	}
}

func TestBuildService_CustomPorts(t *testing.T) {
	instance := newTestInstance("svc-custom-ports")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name: "http",
			Port: 3978,
		},
	}

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	assertServicePort(t, svc.Spec.Ports, "http", 3978)
}

func TestBuildService_CustomPortsWithTargetPort(t *testing.T) {
	instance := newTestInstance("svc-custom-tp")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name:       "http",
			Port:       80,
			TargetPort: Ptr(int32(3978)),
		},
	}

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != 80 {
		t.Errorf("port = %d, want 80", svc.Spec.Ports[0].Port)
	}
	if svc.Spec.Ports[0].TargetPort.IntValue() != 3978 {
		t.Errorf("targetPort = %d, want 3978", svc.Spec.Ports[0].TargetPort.IntValue())
	}
}

func TestBuildService_CustomPortsMultiple(t *testing.T) {
	instance := newTestInstance("svc-multi-ports")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name: "http",
			Port: 3978,
		},
		{
			Name:       "grpc",
			Port:       50051,
			TargetPort: Ptr(int32(50051)),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(svc.Spec.Ports))
	}
	assertServicePort(t, svc.Spec.Ports, "http", 3978)
	assertServicePort(t, svc.Spec.Ports, "grpc", 50051)
}

func TestBuildService_CustomPortsOverrideDefaults(t *testing.T) {
	instance := newTestInstance("svc-override")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name: "http",
			Port: 8080,
		},
	}
	instance.Spec.Chromium.Enabled = true

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("custom ports should replace defaults including chromium; got %d ports", len(svc.Spec.Ports))
	}
	assertServicePort(t, svc.Spec.Ports, "http", 8080)
}

func TestBuildService_CustomPortsDefaultProtocol(t *testing.T) {
	instance := newTestInstance("svc-proto")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name: "http",
			Port: 8080,
		},
	}

	svc := BuildService(instance)

	if svc.Spec.Ports[0].Protocol != corev1.ProtocolTCP {
		t.Errorf("protocol = %v, want TCP", svc.Spec.Ports[0].Protocol)
	}
}

func TestBuildService_CustomPortsTargetPortDefaultsToPort(t *testing.T) {
	instance := newTestInstance("svc-tp-default")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{
			Name: "http",
			Port: 3978,
		},
	}

	svc := BuildService(instance)

	if svc.Spec.Ports[0].TargetPort.IntValue() != 3978 {
		t.Errorf("targetPort = %d, want 3978 (same as port)", svc.Spec.Ports[0].TargetPort.IntValue())
	}
}

// ---------------------------------------------------------------------------
// networkpolicy.go tests
// ---------------------------------------------------------------------------

func TestBuildNetworkPolicy_Default(t *testing.T) {
	instance := newTestInstance("np-test")
	np := BuildNetworkPolicy(instance)

	if np.Name != "np-test" {
		t.Errorf("network policy name = %q, want %q", np.Name, "np-test")
	}
	if np.Namespace != "test-ns" {
		t.Errorf("network policy namespace = %q, want %q", np.Namespace, "test-ns")
	}

	// Labels
	if np.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("network policy missing app label")
	}

	// Pod selector
	sel := np.Spec.PodSelector.MatchLabels
	if sel["app.kubernetes.io/name"] != "openclaw" || sel["app.kubernetes.io/instance"] != "np-test" {
		t.Error("pod selector does not match expected values")
	}

	// Policy types
	if len(np.Spec.PolicyTypes) != 2 {
		t.Fatalf("expected 2 policy types, got %d", len(np.Spec.PolicyTypes))
	}

	// Ingress rules - by default, allow from same namespace
	if len(np.Spec.Ingress) < 1 {
		t.Fatal("expected at least 1 ingress rule")
	}
	firstIngress := np.Spec.Ingress[0]
	if len(firstIngress.From) != 1 {
		t.Fatalf("expected 1 peer in first ingress rule, got %d", len(firstIngress.From))
	}
	nsSel := firstIngress.From[0].NamespaceSelector
	if nsSel == nil {
		t.Fatal("first ingress rule should have namespace selector")
	}
	if nsSel.MatchLabels["kubernetes.io/metadata.name"] != "test-ns" {
		t.Errorf("ingress namespace selector = %v, want test-ns", nsSel.MatchLabels)
	}

	// Ingress ports - gateway and canvas
	if len(firstIngress.Ports) != 2 {
		t.Fatalf("expected 2 ingress ports, got %d", len(firstIngress.Ports))
	}
	assertNPPort(t, firstIngress.Ports, GatewayPort)
	assertNPPort(t, firstIngress.Ports, CanvasPort)

	// Egress rules - DNS (UDP+TCP 53) and HTTPS (443)
	if len(np.Spec.Egress) < 2 {
		t.Fatalf("expected at least 2 egress rules (DNS + HTTPS), got %d", len(np.Spec.Egress))
	}

	// First egress: DNS
	dnsRule := np.Spec.Egress[0]
	if len(dnsRule.Ports) != 2 {
		t.Fatalf("DNS egress rule should have 2 ports (UDP+TCP), got %d", len(dnsRule.Ports))
	}
	foundUDP53 := false
	foundTCP53 := false
	for _, p := range dnsRule.Ports {
		if p.Port != nil && p.Port.IntValue() == 53 {
			if p.Protocol != nil && *p.Protocol == corev1.ProtocolUDP {
				foundUDP53 = true
			}
			if p.Protocol != nil && *p.Protocol == corev1.ProtocolTCP {
				foundTCP53 = true
			}
		}
	}
	if !foundUDP53 {
		t.Error("DNS egress rule missing UDP port 53")
	}
	if !foundTCP53 {
		t.Error("DNS egress rule missing TCP port 53")
	}

	// Second egress: HTTPS
	httpsRule := np.Spec.Egress[1]
	if len(httpsRule.Ports) != 1 {
		t.Fatalf("HTTPS egress rule should have 1 port, got %d", len(httpsRule.Ports))
	}
	if httpsRule.Ports[0].Port == nil || httpsRule.Ports[0].Port.IntValue() != 443 {
		t.Error("HTTPS egress rule should allow port 443")
	}
}

func TestBuildNetworkPolicy_CustomServicePorts(t *testing.T) {
	instance := newTestInstance("np-custom-ports")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{Name: "http", Port: 3978},
		{Name: "grpc", Port: 50051, Protocol: corev1.ProtocolTCP},
	}

	np := BuildNetworkPolicy(instance)

	// Same-namespace ingress rule should use custom ports
	firstIngress := np.Spec.Ingress[0]
	if len(firstIngress.Ports) != 2 {
		t.Fatalf("expected 2 ingress ports for custom service ports, got %d", len(firstIngress.Ports))
	}
	assertNPPort(t, firstIngress.Ports, 3978)
	assertNPPort(t, firstIngress.Ports, 50051)
}

func TestBuildNetworkPolicy_CustomServicePortsWithTargetPort(t *testing.T) {
	instance := newTestInstance("np-custom-tp")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{Name: "http", Port: 80, TargetPort: Ptr(int32(3978))},
	}

	np := BuildNetworkPolicy(instance)

	firstIngress := np.Spec.Ingress[0]
	if len(firstIngress.Ports) != 1 {
		t.Fatalf("expected 1 ingress port, got %d", len(firstIngress.Ports))
	}
	// NetworkPolicy should use targetPort (container port), not service port
	assertNPPort(t, firstIngress.Ports, 3978)
}

func TestBuildNetworkPolicy_CustomPortsApplyToAllRules(t *testing.T) {
	instance := newTestInstance("np-custom-all")
	instance.Spec.Networking.Service.Ports = []openclawv1alpha1.ServicePortSpec{
		{Name: "http", Port: 8080},
	}
	instance.Spec.Security.NetworkPolicy.AllowedIngressNamespaces = []string{"monitoring"}
	instance.Spec.Security.NetworkPolicy.AllowedIngressCIDRs = []string{"10.0.0.0/8"}

	np := BuildNetworkPolicy(instance)

	// 3 rules: same-ns, monitoring ns, CIDR
	if len(np.Spec.Ingress) != 3 {
		t.Fatalf("expected 3 ingress rules, got %d", len(np.Spec.Ingress))
	}
	for i, rule := range np.Spec.Ingress {
		if len(rule.Ports) != 1 {
			t.Fatalf("rule %d: expected 1 port, got %d", i, len(rule.Ports))
		}
		assertNPPort(t, rule.Ports, 8080)
	}
}

func TestBuildNetworkPolicy_CustomCIDRs(t *testing.T) {
	instance := newTestInstance("np-cidrs")
	instance.Spec.Security.NetworkPolicy.AllowedIngressCIDRs = []string{
		"10.0.0.0/8",
		"192.168.1.0/24",
	}
	instance.Spec.Security.NetworkPolicy.AllowedEgressCIDRs = []string{
		"172.16.0.0/12",
	}

	np := BuildNetworkPolicy(instance)

	// Should have 3 ingress rules: same-ns + 2 CIDRs
	if len(np.Spec.Ingress) != 3 {
		t.Fatalf("expected 3 ingress rules, got %d", len(np.Spec.Ingress))
	}

	// Verify CIDR ingress rules
	cidrRule1 := np.Spec.Ingress[1]
	if cidrRule1.From[0].IPBlock == nil {
		t.Fatal("second ingress rule should have IPBlock")
	}
	if cidrRule1.From[0].IPBlock.CIDR != "10.0.0.0/8" {
		t.Errorf("first CIDR = %q, want %q", cidrRule1.From[0].IPBlock.CIDR, "10.0.0.0/8")
	}

	cidrRule2 := np.Spec.Ingress[2]
	if cidrRule2.From[0].IPBlock.CIDR != "192.168.1.0/24" {
		t.Errorf("second CIDR = %q, want %q", cidrRule2.From[0].IPBlock.CIDR, "192.168.1.0/24")
	}

	// Egress should include the CIDR rule (DNS + HTTPS + 1 custom)
	if len(np.Spec.Egress) != 3 {
		t.Fatalf("expected 3 egress rules, got %d", len(np.Spec.Egress))
	}
	egressCIDR := np.Spec.Egress[2]
	if len(egressCIDR.To) != 1 || egressCIDR.To[0].IPBlock == nil {
		t.Fatal("third egress rule should have IPBlock")
	}
	if egressCIDR.To[0].IPBlock.CIDR != "172.16.0.0/12" {
		t.Errorf("egress CIDR = %q, want %q", egressCIDR.To[0].IPBlock.CIDR, "172.16.0.0/12")
	}
}

func TestBuildNetworkPolicy_DNSDisabled(t *testing.T) {
	instance := newTestInstance("np-no-dns")
	instance.Spec.Security.NetworkPolicy.AllowDNS = Ptr(false)

	np := BuildNetworkPolicy(instance)

	// Without DNS, only HTTPS egress rule
	if len(np.Spec.Egress) != 1 {
		t.Fatalf("expected 1 egress rule (HTTPS only), got %d", len(np.Spec.Egress))
	}

	// The single rule should be HTTPS (443)
	httpsRule := np.Spec.Egress[0]
	if len(httpsRule.Ports) != 1 || httpsRule.Ports[0].Port.IntValue() != 443 {
		t.Error("single egress rule should be HTTPS port 443")
	}
}

func TestBuildNetworkPolicy_AllowedNamespaces(t *testing.T) {
	instance := newTestInstance("np-ns")
	instance.Spec.Security.NetworkPolicy.AllowedIngressNamespaces = []string{
		"ingress-nginx",
		"monitoring",
	}

	np := BuildNetworkPolicy(instance)

	// Should have 3 ingress rules: same-ns + 2 allowed namespaces
	if len(np.Spec.Ingress) != 3 {
		t.Fatalf("expected 3 ingress rules, got %d", len(np.Spec.Ingress))
	}

	nsRule1 := np.Spec.Ingress[1]
	if nsRule1.From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "ingress-nginx" {
		t.Error("second ingress rule should allow ingress-nginx namespace")
	}
	nsRule2 := np.Spec.Ingress[2]
	if nsRule2.From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "monitoring" {
		t.Error("third ingress rule should allow monitoring namespace")
	}
}

func TestBuildNetworkPolicy_AdditionalEgress(t *testing.T) {
	instance := newTestInstance("np-extra-egress")
	instance.Spec.Security.NetworkPolicy.AdditionalEgress = []networkingv1.NetworkPolicyEgressRule{
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "bifrost",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: Ptr(corev1.ProtocolTCP),
					Port:     Ptr(intstr.FromInt(8080)),
				},
			},
		},
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "10.96.0.0/16",
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: Ptr(corev1.ProtocolTCP),
					Port:     Ptr(intstr.FromInt(9090)),
				},
			},
		},
	}

	np := BuildNetworkPolicy(instance)

	// Default rules: DNS (index 0) + HTTPS (index 1) = 2, plus 2 additional = 4
	if len(np.Spec.Egress) != 4 {
		t.Fatalf("expected 4 egress rules (DNS + HTTPS + 2 additional), got %d", len(np.Spec.Egress))
	}

	// First two rules should be the defaults (DNS + HTTPS)
	dnsRule := np.Spec.Egress[0]
	if len(dnsRule.Ports) != 2 || dnsRule.Ports[0].Port.IntValue() != 53 {
		t.Error("first egress rule should be DNS (port 53)")
	}
	httpsRule := np.Spec.Egress[1]
	if len(httpsRule.Ports) != 1 || httpsRule.Ports[0].Port.IntValue() != 443 {
		t.Error("second egress rule should be HTTPS (port 443)")
	}

	// Third rule: bifrost namespace on port 8080
	bifrostRule := np.Spec.Egress[2]
	if len(bifrostRule.To) != 1 || bifrostRule.To[0].NamespaceSelector == nil {
		t.Fatal("third egress rule should have a namespace selector")
	}
	if bifrostRule.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "bifrost" {
		t.Error("third egress rule should target bifrost namespace")
	}
	if len(bifrostRule.Ports) != 1 || bifrostRule.Ports[0].Port.IntValue() != 8080 {
		t.Error("third egress rule should allow port 8080")
	}

	// Fourth rule: CIDR 10.96.0.0/16 on port 9090
	cidrRule := np.Spec.Egress[3]
	if len(cidrRule.To) != 1 || cidrRule.To[0].IPBlock == nil {
		t.Fatal("fourth egress rule should have an IPBlock")
	}
	if cidrRule.To[0].IPBlock.CIDR != "10.96.0.0/16" {
		t.Errorf("fourth egress CIDR = %q, want %q", cidrRule.To[0].IPBlock.CIDR, "10.96.0.0/16")
	}
	if len(cidrRule.Ports) != 1 || cidrRule.Ports[0].Port.IntValue() != 9090 {
		t.Error("fourth egress rule should allow port 9090")
	}
}

// ---------------------------------------------------------------------------
// rbac.go tests
// ---------------------------------------------------------------------------

func TestBuildServiceAccount(t *testing.T) {
	instance := newTestInstance("sa-test")
	sa := BuildServiceAccount(instance)

	if sa.Name != "sa-test" {
		t.Errorf("service account name = %q, want %q", sa.Name, "sa-test")
	}
	if sa.Namespace != "test-ns" {
		t.Errorf("service account namespace = %q, want %q", sa.Namespace, "test-ns")
	}
	if sa.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("service account missing app label")
	}
	if sa.Labels["app.kubernetes.io/instance"] != "sa-test" {
		t.Error("service account missing instance label")
	}
	if sa.Labels["app.kubernetes.io/managed-by"] != "openclaw-operator" {
		t.Error("service account missing managed-by label")
	}
}

func TestBuildServiceAccount_AutomountDisabled(t *testing.T) {
	instance := newTestInstance("sa-automount")
	sa := BuildServiceAccount(instance)
	if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken != false {
		t.Errorf("AutomountServiceAccountToken = %v, want false", sa.AutomountServiceAccountToken)
	}
}

func TestBuildServiceAccount_CustomName(t *testing.T) {
	instance := newTestInstance("sa-custom")
	instance.Spec.Security.RBAC.ServiceAccountName = "my-custom-sa"

	sa := BuildServiceAccount(instance)

	if sa.Name != "my-custom-sa" {
		t.Errorf("service account name = %q, want %q", sa.Name, "my-custom-sa")
	}
}

func TestBuildRole_Default(t *testing.T) {
	instance := newTestInstance("role-test")
	role := BuildRole(instance)

	if role.Name != "role-test" {
		t.Errorf("role name = %q, want %q", role.Name, "role-test")
	}
	if role.Namespace != "test-ns" {
		t.Errorf("role namespace = %q, want %q", role.Namespace, "test-ns")
	}

	// Should have exactly 1 rule (configmap read)
	if len(role.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(role.Rules))
	}

	rule := role.Rules[0]
	if len(rule.APIGroups) != 1 || rule.APIGroups[0] != "" {
		t.Error("base rule should have core API group")
	}
	if len(rule.Resources) != 1 || rule.Resources[0] != "configmaps" {
		t.Errorf("base rule resources = %v, want [configmaps]", rule.Resources)
	}
	if len(rule.ResourceNames) != 1 || rule.ResourceNames[0] != "role-test-config" {
		t.Errorf("base rule resourceNames = %v, want [role-test-config]", rule.ResourceNames)
	}
	if len(rule.Verbs) != 2 {
		t.Fatalf("expected 2 verbs, got %d", len(rule.Verbs))
	}
	expectedVerbs := map[string]bool{"get": true, "watch": true}
	for _, v := range rule.Verbs {
		if !expectedVerbs[v] {
			t.Errorf("unexpected verb %q", v)
		}
	}
}

func TestBuildRole_AdditionalRules(t *testing.T) {
	instance := newTestInstance("role-extra")
	instance.Spec.Security.RBAC.AdditionalRules = []openclawv1alpha1.RBACRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
			Verbs:     []string{"get"},
		},
	}

	role := BuildRole(instance)

	// 1 base rule + 2 additional rules
	if len(role.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(role.Rules))
	}

	// Verify additional rules
	secondRule := role.Rules[1]
	if secondRule.Resources[0] != "secrets" {
		t.Errorf("second rule resources = %v, want [secrets]", secondRule.Resources)
	}
	thirdRule := role.Rules[2]
	if thirdRule.APIGroups[0] != "apps" || thirdRule.Resources[0] != "deployments" {
		t.Error("third rule does not match expected values")
	}
}

func TestBuildRoleBinding(t *testing.T) {
	instance := newTestInstance("rb-test")
	rb := BuildRoleBinding(instance)

	if rb.Name != "rb-test" {
		t.Errorf("role binding name = %q, want %q", rb.Name, "rb-test")
	}
	if rb.Namespace != "test-ns" {
		t.Errorf("role binding namespace = %q, want %q", rb.Namespace, "test-ns")
	}

	// RoleRef
	if rb.RoleRef.Kind != "Role" {
		t.Errorf("roleRef kind = %q, want %q", rb.RoleRef.Kind, "Role")
	}
	if rb.RoleRef.Name != "rb-test" {
		t.Errorf("roleRef name = %q, want %q", rb.RoleRef.Name, "rb-test")
	}
	if rb.RoleRef.APIGroup != "rbac.authorization.k8s.io" {
		t.Errorf("roleRef apiGroup = %q, want %q", rb.RoleRef.APIGroup, "rbac.authorization.k8s.io")
	}

	// Subjects
	if len(rb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(rb.Subjects))
	}
	subj := rb.Subjects[0]
	if subj.Kind != "ServiceAccount" {
		t.Errorf("subject kind = %q, want ServiceAccount", subj.Kind)
	}
	if subj.Name != "rb-test" {
		t.Errorf("subject name = %q, want %q", subj.Name, "rb-test")
	}
	if subj.Namespace != "test-ns" {
		t.Errorf("subject namespace = %q, want %q", subj.Namespace, "test-ns")
	}
}

func TestBuildRoleBinding_CustomServiceAccount(t *testing.T) {
	instance := newTestInstance("rb-custom-sa")
	instance.Spec.Security.RBAC.ServiceAccountName = "my-sa"

	rb := BuildRoleBinding(instance)

	// Subject should use the custom SA name
	if rb.Subjects[0].Name != "my-sa" {
		t.Errorf("subject name = %q, want %q", rb.Subjects[0].Name, "my-sa")
	}
	// RoleRef should still use instance name
	if rb.RoleRef.Name != "rb-custom-sa" {
		t.Errorf("roleRef name = %q, want %q", rb.RoleRef.Name, "rb-custom-sa")
	}
}

// ---------------------------------------------------------------------------
// configmap.go tests
// ---------------------------------------------------------------------------

func TestBuildConfigMap_Default(t *testing.T) {
	instance := newTestInstance("cm-test")
	cm := BuildConfigMap(instance, "")

	if cm.Name != "cm-test-config" {
		t.Errorf("configmap name = %q, want %q", cm.Name, "cm-test-config")
	}
	if cm.Namespace != "test-ns" {
		t.Errorf("configmap namespace = %q, want %q", cm.Namespace, "test-ns")
	}
	if cm.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("configmap missing app label")
	}

	// Default config should have gateway.bind=lan injected
	content, ok := cm.Data["openclaw.json"]
	if !ok {
		t.Fatal("configmap missing openclaw.json key")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse default config: %v", err)
	}
	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key in default config")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

func TestBuildConfigMap_RawConfig(t *testing.T) {
	instance := newTestInstance("cm-raw")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"mcpServers":{"test":{"url":"http://localhost:3000"}}}`),
		},
	}

	cm := BuildConfigMap(instance, "")

	content, ok := cm.Data["openclaw.json"]
	if !ok {
		t.Fatal("configmap missing openclaw.json key")
	}

	// The builder pretty-prints JSON, so check it contains the expected keys
	if content == "{}" {
		t.Error("config content should not be empty with raw config")
	}

	// Verify the content is valid JSON and contains expected data
	if content == "" {
		t.Error("config content should not be empty")
	}
}

func TestBuildConfigMap_InvalidJSON_RawPreserved(t *testing.T) {
	instance := newTestInstance("cm-invalid")
	// If raw JSON is technically valid but the builder tries to pretty-print,
	// verify it handles valid JSON correctly and gateway.bind is injected
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"key":"value"}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("key = %v, want %q", parsed["key"], "value")
	}
	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key after enrichment")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

// ---------------------------------------------------------------------------
// BuildConfigMapFromBytes tests
// ---------------------------------------------------------------------------

func TestBuildConfigMapFromBytes_EnrichesExternalConfig(t *testing.T) {
	instance := newTestInstance("from-bytes")
	externalConfig := []byte(`{"mcpServers":{"test":{"url":"http://localhost"}}}`)

	cm := BuildConfigMapFromBytes(instance, externalConfig, "my-gateway-token")

	content := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify user config is preserved
	if _, ok := parsed["mcpServers"]; !ok {
		t.Error("mcpServers should be preserved from external config")
	}

	// Verify gateway auth was injected
	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key after enrichment")
	}
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway.auth key after enrichment")
	}
	if auth["mode"] != "token" {
		t.Errorf("gateway.auth.mode = %v, want %q", auth["mode"], "token")
	}
	if auth["token"] != "my-gateway-token" {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], "my-gateway-token")
	}

	// Verify gateway.bind was injected
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

func TestBuildConfigMapFromBytes_PreservesUserConfig(t *testing.T) {
	instance := newTestInstance("from-bytes-preserve")
	externalConfig := []byte(`{
		"mcpServers": {"fetch": {"url": "http://localhost:3000"}},
		"globalShortcut": "Ctrl+Space",
		"selectedProvider": "anthropic"
	}`)

	cm := BuildConfigMapFromBytes(instance, externalConfig, "tok")

	content := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if parsed["globalShortcut"] != "Ctrl+Space" {
		t.Errorf("globalShortcut = %v, want %q", parsed["globalShortcut"], "Ctrl+Space")
	}
	if parsed["selectedProvider"] != "anthropic" {
		t.Errorf("selectedProvider = %v, want %q", parsed["selectedProvider"], "anthropic")
	}
	if _, ok := parsed["mcpServers"]; !ok {
		t.Error("mcpServers should be preserved from external config")
	}
}

func TestBuildConfigMapFromBytes_EmptyBytes(t *testing.T) {
	instance := newTestInstance("from-bytes-empty")

	cm := BuildConfigMapFromBytes(instance, nil, "tok")

	content := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Should still have gateway enrichment
	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key even with empty base config")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

func TestBuildConfigMapFromBytes_JSON5Passthrough(t *testing.T) {
	instance := newTestInstance("from-bytes-json5")
	// JSON5 content can't be parsed as JSON, so enrichment returns it unchanged
	json5Content := []byte(`{mcpServers: {test: {url: "http://localhost"}}}`)

	cm := BuildConfigMapFromBytes(instance, json5Content, "tok")

	// JSON5 content should pass through unchanged (enrichment can't parse it)
	content := cm.Data["openclaw.json"]
	if content != string(json5Content) {
		t.Errorf("JSON5 content should pass through unchanged, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// enrichConfigWithGatewayBind tests
// ---------------------------------------------------------------------------

func TestEnrichConfigWithGatewayBind(t *testing.T) {
	input := []byte(`{}`)
	out, err := enrichConfigWithGatewayBind(input)
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}

	gw, ok := cfg["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

func TestEnrichConfigWithGatewayBind_PreservesUserBind(t *testing.T) {
	input := []byte(`{"gateway":{"bind":"0.0.0.0"}}`)
	out, err := enrichConfigWithGatewayBind(input)
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}

	gw := cfg["gateway"].(map[string]interface{})
	if gw["bind"] != "0.0.0.0" {
		t.Errorf("gateway.bind = %v, want %q (user override)", gw["bind"], "0.0.0.0")
	}
}

func TestEnrichConfigWithGatewayBind_PreservesOtherFields(t *testing.T) {
	input := []byte(`{"gateway":{"auth":{"mode":"token","token":"secret"}}}`)
	out, err := enrichConfigWithGatewayBind(input)
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatal(err)
	}

	gw := cfg["gateway"].(map[string]interface{})
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("gateway.auth should be preserved")
	}
	if auth["token"] != "secret" {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], "secret")
	}
}

func TestEnrichConfigWithGatewayBind_InvalidJSON(t *testing.T) {
	input := []byte(`not valid json`)
	out, err := enrichConfigWithGatewayBind(input)
	if err != nil {
		t.Fatal("should not error on invalid JSON")
	}

	if !bytes.Equal(out, input) {
		t.Errorf("invalid JSON should be returned unchanged")
	}
}

func TestBuildConfigMap_RawConfig_GatewayBindInjected(t *testing.T) {
	instance := newTestInstance("cm-bind-inject")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"mcpServers":{"test":{"url":"http://localhost"}}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key injected into raw config")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
	// Original data preserved
	if _, ok := parsed["mcpServers"]; !ok {
		t.Error("mcpServers should be preserved from raw config")
	}
}

func TestBuildConfigMap_RawConfig_UserBindPreserved(t *testing.T) {
	instance := newTestInstance("cm-bind-preserve")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"gateway":{"bind":"0.0.0.0"}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	if gw["bind"] != "0.0.0.0" {
		t.Errorf("gateway.bind = %v, want %q (user override should win)", gw["bind"], "0.0.0.0")
	}
}

// ---------------------------------------------------------------------------
// pvc.go tests
// ---------------------------------------------------------------------------

func TestBuildPVC_Default(t *testing.T) {
	instance := newTestInstance("pvc-test")
	pvc := BuildPVC(instance)

	if pvc.Name != "pvc-test-data" {
		t.Errorf("pvc name = %q, want %q", pvc.Name, "pvc-test-data")
	}
	if pvc.Namespace != "test-ns" {
		t.Errorf("pvc namespace = %q, want %q", pvc.Namespace, "test-ns")
	}
	if pvc.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("pvc missing app label")
	}

	// Backup annotation
	if pvc.Annotations["openclaw.rocks/backup-enabled"] != "true" {
		t.Error("pvc missing backup-enabled annotation")
	}

	// Default size - 10Gi
	storageReq := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if storageReq.Cmp(resource.MustParse("10Gi")) != 0 {
		t.Errorf("storage size = %v, want 10Gi", storageReq.String())
	}

	// Default access mode - ReadWriteOnce
	if len(pvc.Spec.AccessModes) != 1 {
		t.Fatalf("expected 1 access mode, got %d", len(pvc.Spec.AccessModes))
	}
	if pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce {
		t.Errorf("access mode = %v, want ReadWriteOnce", pvc.Spec.AccessModes[0])
	}

	// No storage class by default
	if pvc.Spec.StorageClassName != nil {
		t.Errorf("storageClassName should be nil by default, got %v", *pvc.Spec.StorageClassName)
	}
}

func TestBuildPVC_CustomSize(t *testing.T) {
	instance := newTestInstance("pvc-custom")
	instance.Spec.Storage.Persistence.Size = "50Gi"
	scName := "fast-ssd"
	instance.Spec.Storage.Persistence.StorageClass = &scName

	pvc := BuildPVC(instance)

	storageReq := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if storageReq.Cmp(resource.MustParse("50Gi")) != 0 {
		t.Errorf("storage size = %v, want 50Gi", storageReq.String())
	}

	if pvc.Spec.StorageClassName == nil {
		t.Fatal("storageClassName should not be nil")
	}
	if *pvc.Spec.StorageClassName != "fast-ssd" {
		t.Errorf("storageClassName = %q, want %q", *pvc.Spec.StorageClassName, "fast-ssd")
	}
}

func TestBuildPVC_CustomAccessModes(t *testing.T) {
	instance := newTestInstance("pvc-modes")
	instance.Spec.Storage.Persistence.AccessModes = []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany,
	}

	pvc := BuildPVC(instance)

	if len(pvc.Spec.AccessModes) != 1 {
		t.Fatalf("expected 1 access mode, got %d", len(pvc.Spec.AccessModes))
	}
	if pvc.Spec.AccessModes[0] != corev1.ReadWriteMany {
		t.Errorf("access mode = %v, want ReadWriteMany", pvc.Spec.AccessModes[0])
	}
}

// ---------------------------------------------------------------------------
// pdb.go tests
// ---------------------------------------------------------------------------

func TestBuildPDB_Default(t *testing.T) {
	instance := newTestInstance("pdb-test")
	pdb := BuildPDB(instance)

	if pdb.Name != "pdb-test" {
		t.Errorf("pdb name = %q, want %q", pdb.Name, "pdb-test")
	}
	if pdb.Namespace != "test-ns" {
		t.Errorf("pdb namespace = %q, want %q", pdb.Namespace, "test-ns")
	}
	if pdb.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("pdb missing app label")
	}

	// Selector
	sel := pdb.Spec.Selector.MatchLabels
	if sel["app.kubernetes.io/name"] != "openclaw" || sel["app.kubernetes.io/instance"] != "pdb-test" {
		t.Error("pdb selector does not match expected values")
	}

	// Default maxUnavailable = 1
	if pdb.Spec.MaxUnavailable == nil {
		t.Fatal("maxUnavailable should not be nil")
	}
	if pdb.Spec.MaxUnavailable.Type != intstr.Int {
		t.Error("maxUnavailable should be int type")
	}
	if pdb.Spec.MaxUnavailable.IntVal != 1 {
		t.Errorf("maxUnavailable = %d, want 1", pdb.Spec.MaxUnavailable.IntVal)
	}
}

func TestBuildPDB_Custom(t *testing.T) {
	instance := newTestInstance("pdb-custom")
	instance.Spec.Availability.PodDisruptionBudget = &openclawv1alpha1.PodDisruptionBudgetSpec{
		MaxUnavailable: Ptr(int32(0)),
	}

	pdb := BuildPDB(instance)

	if pdb.Spec.MaxUnavailable.IntVal != 0 {
		t.Errorf("maxUnavailable = %d, want 0", pdb.Spec.MaxUnavailable.IntVal)
	}
}

func TestBuildPDB_CustomValue(t *testing.T) {
	instance := newTestInstance("pdb-val")
	instance.Spec.Availability.PodDisruptionBudget = &openclawv1alpha1.PodDisruptionBudgetSpec{
		MaxUnavailable: Ptr(int32(2)),
	}

	pdb := BuildPDB(instance)

	if pdb.Spec.MaxUnavailable.IntVal != 2 {
		t.Errorf("maxUnavailable = %d, want 2", pdb.Spec.MaxUnavailable.IntVal)
	}
}

// ---------------------------------------------------------------------------
// ingress.go tests
// ---------------------------------------------------------------------------

func TestBuildIngress_Basic(t *testing.T) {
	instance := newTestInstance("ing-test")
	className := "nginx"
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: &className,
		Hosts: []openclawv1alpha1.IngressHost{
			{
				Host: "openclaw.example.com",
			},
		},
		TLS: []openclawv1alpha1.IngressTLS{
			{
				Hosts:      []string{"openclaw.example.com"},
				SecretName: "openclaw-tls",
			},
		},
	}

	ing := BuildIngress(instance)

	// ObjectMeta
	if ing.Name != "ing-test" {
		t.Errorf("ingress name = %q, want %q", ing.Name, "ing-test")
	}
	if ing.Namespace != "test-ns" {
		t.Errorf("ingress namespace = %q, want %q", ing.Namespace, "test-ns")
	}
	if ing.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("ingress missing app label")
	}

	// IngressClassName
	if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "nginx" {
		t.Error("ingress className should be nginx")
	}

	// Rules
	if len(ing.Spec.Rules) != 1 {
		t.Fatalf("expected 1 ingress rule, got %d", len(ing.Spec.Rules))
	}
	rule := ing.Spec.Rules[0]
	if rule.Host != "openclaw.example.com" {
		t.Errorf("ingress host = %q, want %q", rule.Host, "openclaw.example.com")
	}
	if rule.HTTP == nil || len(rule.HTTP.Paths) != 1 {
		t.Fatal("expected 1 path in ingress rule")
	}
	path := rule.HTTP.Paths[0]
	if path.Path != "/" {
		t.Errorf("ingress path = %q, want %q", path.Path, "/")
	}
	if path.Backend.Service == nil {
		t.Fatal("ingress backend service is nil")
	}
	if path.Backend.Service.Name != "ing-test" {
		t.Errorf("ingress backend service name = %q, want %q", path.Backend.Service.Name, "ing-test")
	}
	if path.Backend.Service.Port.Number != int32(GatewayPort) {
		t.Errorf("ingress backend port = %d, want %d", path.Backend.Service.Port.Number, GatewayPort)
	}

	// TLS
	if len(ing.Spec.TLS) != 1 {
		t.Fatalf("expected 1 TLS config, got %d", len(ing.Spec.TLS))
	}
	tls := ing.Spec.TLS[0]
	if tls.SecretName != "openclaw-tls" {
		t.Errorf("TLS secretName = %q, want %q", tls.SecretName, "openclaw-tls")
	}
	if len(tls.Hosts) != 1 || tls.Hosts[0] != "openclaw.example.com" {
		t.Errorf("TLS hosts = %v, want [openclaw.example.com]", tls.Hosts)
	}
}

func TestBuildIngress_DefaultAnnotations(t *testing.T) {
	instance := newTestInstance("ing-ann")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	// No className = unknown provider = no provider-specific annotations
	if _, ok := ann["nginx.ingress.kubernetes.io/ssl-redirect"]; ok {
		t.Error("nginx annotations should not be emitted for unknown provider")
	}
	if _, ok := ann["traefik.ingress.kubernetes.io/router.entrypoints"]; ok {
		t.Error("traefik annotations should not be emitted for unknown provider")
	}
	if _, ok := ann["traefik.ingress.kubernetes.io/router.middlewares"]; ok {
		t.Error("router.middlewares annotation should never be emitted")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/proxy-read-timeout"]; ok {
		t.Error("nginx WebSocket annotations should not be emitted for unknown provider")
	}
}

func TestBuildIngress_SecurityDisabled(t *testing.T) {
	instance := newTestInstance("ing-nosec")
	className := "nginx"
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: &className,
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
		Security: openclawv1alpha1.IngressSecuritySpec{
			ForceHTTPS: Ptr(false),
			EnableHSTS: Ptr(false),
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	if _, ok := ann["nginx.ingress.kubernetes.io/ssl-redirect"]; ok {
		t.Error("ssl-redirect should not be set when ForceHTTPS is false")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/force-ssl-redirect"]; ok {
		t.Error("force-ssl-redirect should not be set when ForceHTTPS is false")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/configuration-snippet"]; ok {
		t.Error("HSTS configuration snippet should not be set when EnableHSTS is false")
	}
	if _, ok := ann["traefik.ingress.kubernetes.io/router.entrypoints"]; ok {
		t.Error("traefik annotations should not be emitted for nginx provider")
	}

	// WebSocket annotations should still be present for nginx provider
	if ann["nginx.ingress.kubernetes.io/proxy-read-timeout"] != "3600" {
		t.Error("proxy-read-timeout should still be 3600")
	}
}

func TestBuildIngress_RateLimiting(t *testing.T) {
	instance := newTestInstance("ing-rl")
	rps := int32(20)
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("nginx"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
		Security: openclawv1alpha1.IngressSecuritySpec{
			RateLimiting: &openclawv1alpha1.RateLimitingSpec{
				RequestsPerSecond: &rps,
			},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	if ann["nginx.ingress.kubernetes.io/limit-rps"] != "20" {
		t.Errorf("limit-rps = %q, want %q", ann["nginx.ingress.kubernetes.io/limit-rps"], "20")
	}
}

func TestBuildIngress_RateLimitingDefault(t *testing.T) {
	instance := newTestInstance("ing-rl-default")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("nginx"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
		Security: openclawv1alpha1.IngressSecuritySpec{
			RateLimiting: &openclawv1alpha1.RateLimitingSpec{
				// Enabled defaults to true, RPS defaults to 10
			},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	if ann["nginx.ingress.kubernetes.io/limit-rps"] != "10" {
		t.Errorf("limit-rps = %q, want %q", ann["nginx.ingress.kubernetes.io/limit-rps"], "10")
	}
}

func TestBuildIngress_RateLimitingDisabled(t *testing.T) {
	instance := newTestInstance("ing-rl-off")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("nginx"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
		Security: openclawv1alpha1.IngressSecuritySpec{
			RateLimiting: &openclawv1alpha1.RateLimitingSpec{
				Enabled: Ptr(false),
			},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	if _, ok := ann["nginx.ingress.kubernetes.io/limit-rps"]; ok {
		t.Error("limit-rps should not be set when rate limiting is disabled")
	}
}

func TestBuildIngress_CustomAnnotations(t *testing.T) {
	instance := newTestInstance("ing-custom-ann")
	className := "nginx"
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: &className,
		Annotations: map[string]string{
			"custom-key": "custom-value",
		},
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
	}

	ing := BuildIngress(instance)

	if ing.Annotations["custom-key"] != "custom-value" {
		t.Error("custom annotation not preserved")
	}
	// Provider annotations should coexist with custom annotations
	if ing.Annotations["nginx.ingress.kubernetes.io/proxy-http-version"] != "1.1" {
		t.Error("nginx annotations should coexist with custom annotations")
	}
}

func TestBuildIngress_MultipleHosts(t *testing.T) {
	instance := newTestInstance("ing-multi")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "a.example.com"},
			{Host: "b.example.com"},
		},
		TLS: []openclawv1alpha1.IngressTLS{
			{
				Hosts:      []string{"a.example.com", "b.example.com"},
				SecretName: "multi-tls",
			},
		},
	}

	ing := BuildIngress(instance)

	if len(ing.Spec.Rules) != 2 {
		t.Fatalf("expected 2 ingress rules, got %d", len(ing.Spec.Rules))
	}
	if ing.Spec.Rules[0].Host != "a.example.com" {
		t.Errorf("first host = %q, want %q", ing.Spec.Rules[0].Host, "a.example.com")
	}
	if ing.Spec.Rules[1].Host != "b.example.com" {
		t.Errorf("second host = %q, want %q", ing.Spec.Rules[1].Host, "b.example.com")
	}
	if len(ing.Spec.TLS) != 1 {
		t.Fatalf("expected 1 TLS entry, got %d", len(ing.Spec.TLS))
	}
	if len(ing.Spec.TLS[0].Hosts) != 2 {
		t.Errorf("TLS hosts count = %d, want 2", len(ing.Spec.TLS[0].Hosts))
	}
}

func TestBuildIngress_CustomPaths(t *testing.T) {
	instance := newTestInstance("ing-paths")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{
				Host: "test.example.com",
				Paths: []openclawv1alpha1.IngressPath{
					{Path: "/api", PathType: "Prefix"},
					{Path: "/health", PathType: "Exact"},
				},
			},
		},
	}

	ing := BuildIngress(instance)

	rule := ing.Spec.Rules[0]
	if len(rule.HTTP.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(rule.HTTP.Paths))
	}
	if rule.HTTP.Paths[0].Path != "/api" {
		t.Errorf("first path = %q, want %q", rule.HTTP.Paths[0].Path, "/api")
	}
	if rule.HTTP.Paths[1].Path != "/health" {
		t.Errorf("second path = %q, want %q", rule.HTTP.Paths[1].Path, "/health")
	}

	// Verify path types
	if rule.HTTP.Paths[0].PathType == nil {
		t.Fatal("first path pathType is nil")
	}
	// "Prefix" maps to PathTypePrefix
	// "Exact" maps to PathTypeExact

	if rule.HTTP.Paths[1].PathType == nil {
		t.Fatal("second path pathType is nil")
	}
}

func TestBuildIngress_NoHosts(t *testing.T) {
	instance := newTestInstance("ing-no-hosts")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		// No hosts
	}

	ing := BuildIngress(instance)

	if len(ing.Spec.Rules) != 0 {
		t.Errorf("expected 0 rules with no hosts, got %d", len(ing.Spec.Rules))
	}
}

func TestBuildIngress_CustomBackendPort(t *testing.T) {
	instance := newTestInstance("ing-port")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{
				Host: "aibot.example.com",
				Paths: []openclawv1alpha1.IngressPath{
					{Path: "/api/messages", PathType: "Prefix", Port: Ptr(int32(3978))},
				},
			},
		},
	}

	ing := BuildIngress(instance)

	rule := ing.Spec.Rules[0]
	if len(rule.HTTP.Paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(rule.HTTP.Paths))
	}
	backend := rule.HTTP.Paths[0].Backend.Service
	if backend.Port.Number != 3978 {
		t.Errorf("backend port = %d, want 3978", backend.Port.Number)
	}
}

func TestBuildIngress_DefaultBackendPort(t *testing.T) {
	instance := newTestInstance("ing-default-port")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{
				Host: "test.example.com",
				Paths: []openclawv1alpha1.IngressPath{
					{Path: "/", PathType: "Prefix"},
				},
			},
		},
	}

	ing := BuildIngress(instance)

	backend := ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service
	if backend.Port.Number != int32(GatewayPort) {
		t.Errorf("backend port = %d, want %d (GatewayPort)", backend.Port.Number, GatewayPort)
	}
}

func TestBuildIngress_MixedPorts(t *testing.T) {
	instance := newTestInstance("ing-mixed-ports")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts: []openclawv1alpha1.IngressHost{
			{
				Host: "app.example.com",
				Paths: []openclawv1alpha1.IngressPath{
					{Path: "/api", PathType: "Prefix", Port: Ptr(int32(3978))},
					{Path: "/ws", PathType: "Prefix"},
				},
			},
		},
	}

	ing := BuildIngress(instance)

	paths := ing.Spec.Rules[0].HTTP.Paths
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0].Backend.Service.Port.Number != 3978 {
		t.Errorf("first path backend port = %d, want 3978", paths[0].Backend.Service.Port.Number)
	}
	if paths[1].Backend.Service.Port.Number != int32(GatewayPort) {
		t.Errorf("second path backend port = %d, want %d (GatewayPort)", paths[1].Backend.Service.Port.Number, GatewayPort)
	}
}

// ---------------------------------------------------------------------------
// Provider detection tests
// ---------------------------------------------------------------------------

func TestDetectIngressProvider(t *testing.T) {
	tests := []struct {
		name     string
		class    *string
		expected IngressProvider
	}{
		{"nil className", nil, IngressProviderUnknown},
		{"nginx", Ptr("nginx"), IngressProviderNginx},
		{"NGINX uppercase", Ptr("NGINX"), IngressProviderNginx},
		{"nginx-internal", Ptr("nginx-internal"), IngressProviderNginx},
		{"traefik", Ptr("traefik"), IngressProviderTraefik},
		{"Traefik mixed case", Ptr("Traefik"), IngressProviderTraefik},
		{"traefik-external", Ptr("traefik-external"), IngressProviderTraefik},
		{"haproxy", Ptr("haproxy"), IngressProviderUnknown},
		{"empty string", Ptr(""), IngressProviderUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIngressProvider(tt.class)
			if got != tt.expected {
				t.Errorf("DetectIngressProvider(%v) = %q, want %q", tt.class, got, tt.expected)
			}
		})
	}
}

func TestBuildIngress_NginxProvider(t *testing.T) {
	instance := newTestInstance("ing-nginx")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("nginx"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	// nginx annotations should be present
	if ann["nginx.ingress.kubernetes.io/ssl-redirect"] != "true" {
		t.Error("nginx ssl-redirect should be present")
	}
	if ann["nginx.ingress.kubernetes.io/force-ssl-redirect"] != "true" {
		t.Error("nginx force-ssl-redirect should be present")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/configuration-snippet"]; !ok {
		t.Error("nginx HSTS snippet should be present")
	}
	if ann["nginx.ingress.kubernetes.io/proxy-read-timeout"] != "3600" {
		t.Error("nginx proxy-read-timeout should be present")
	}

	// traefik annotations should NOT be present
	if _, ok := ann["traefik.ingress.kubernetes.io/router.entrypoints"]; ok {
		t.Error("traefik annotation should not be emitted for nginx provider")
	}
	if _, ok := ann["traefik.ingress.kubernetes.io/router.middlewares"]; ok {
		t.Error("traefik middleware annotation should never be emitted")
	}
}

func TestBuildIngress_TraefikProvider(t *testing.T) {
	instance := newTestInstance("ing-traefik")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("traefik"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	// traefik annotation should be present
	if ann["traefik.ingress.kubernetes.io/router.entrypoints"] != "websecure" {
		t.Errorf("traefik router.entrypoints = %q, want %q", ann["traefik.ingress.kubernetes.io/router.entrypoints"], "websecure")
	}

	// nginx annotations should NOT be present
	if _, ok := ann["nginx.ingress.kubernetes.io/ssl-redirect"]; ok {
		t.Error("nginx ssl-redirect should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/force-ssl-redirect"]; ok {
		t.Error("nginx force-ssl-redirect should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/configuration-snippet"]; ok {
		t.Error("nginx HSTS snippet should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/proxy-read-timeout"]; ok {
		t.Error("nginx proxy-read-timeout should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/proxy-send-timeout"]; ok {
		t.Error("nginx proxy-send-timeout should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/proxy-http-version"]; ok {
		t.Error("nginx proxy-http-version should not be emitted for traefik provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/upstream-hash-by"]; ok {
		t.Error("nginx upstream-hash-by should not be emitted for traefik provider")
	}

	// Old middleware annotation must never appear
	if _, ok := ann["traefik.ingress.kubernetes.io/router.middlewares"]; ok {
		t.Error("traefik middleware annotation should never be emitted")
	}
}

func TestBuildIngress_UnknownProvider(t *testing.T) {
	instance := newTestInstance("ing-haproxy")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("haproxy"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	// No provider-specific annotations for unknown provider
	if _, ok := ann["nginx.ingress.kubernetes.io/ssl-redirect"]; ok {
		t.Error("nginx annotations should not be emitted for unknown provider")
	}
	if _, ok := ann["traefik.ingress.kubernetes.io/router.entrypoints"]; ok {
		t.Error("traefik annotations should not be emitted for unknown provider")
	}
	if _, ok := ann["nginx.ingress.kubernetes.io/proxy-read-timeout"]; ok {
		t.Error("nginx WebSocket annotations should not be emitted for unknown provider")
	}
}

func TestBuildIngress_TraefikNoRateLimiting(t *testing.T) {
	instance := newTestInstance("ing-traefik-rl")
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled:   true,
		ClassName: Ptr("traefik"),
		Hosts: []openclawv1alpha1.IngressHost{
			{Host: "test.example.com"},
		},
		Security: openclawv1alpha1.IngressSecuritySpec{
			RateLimiting: &openclawv1alpha1.RateLimitingSpec{
				// Enabled defaults to true
			},
		},
	}

	ing := BuildIngress(instance)
	ann := ing.Annotations

	// Rate limiting annotation should NOT be emitted for traefik (requires Middleware CRD)
	if _, ok := ann["nginx.ingress.kubernetes.io/limit-rps"]; ok {
		t.Error("rate limiting annotation should not be emitted for traefik provider")
	}
}

// ---------------------------------------------------------------------------
// Cross-cutting / integration-style tests
// ---------------------------------------------------------------------------

func TestAllBuilders_ConsistentLabels(t *testing.T) {
	instance := newTestInstance("label-check")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{}`),
		},
	}
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts:   []openclawv1alpha1.IngressHost{{Host: "test.example.com"}},
	}

	expectedLabels := Labels(instance)

	resources := []struct {
		name   string
		labels map[string]string
	}{
		{"Deployment", BuildStatefulSet(instance).Labels},
		{"Service", BuildService(instance).Labels},
		{"NetworkPolicy", BuildNetworkPolicy(instance).Labels},
		{"ServiceAccount", BuildServiceAccount(instance).Labels},
		{"Role", BuildRole(instance).Labels},
		{"RoleBinding", BuildRoleBinding(instance).Labels},
		{"ConfigMap", BuildConfigMap(instance, "").Labels},
		{"PVC", BuildPVC(instance).Labels},
		{"PDB", BuildPDB(instance).Labels},
		{"Ingress", BuildIngress(instance).Labels},
	}

	for _, r := range resources {
		t.Run(r.name, func(t *testing.T) {
			for k, v := range expectedLabels {
				if r.labels[k] != v {
					t.Errorf("%s: label %q = %q, want %q", r.name, k, r.labels[k], v)
				}
			}
		})
	}
}

func TestAllBuilders_ConsistentNamespace(t *testing.T) {
	instance := newTestInstance("ns-check")
	instance.Namespace = "production"
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{}`),
		},
	}
	instance.Spec.Networking.Ingress = openclawv1alpha1.IngressSpec{
		Enabled: true,
		Hosts:   []openclawv1alpha1.IngressHost{{Host: "test.example.com"}},
	}

	resources := []struct {
		name      string
		namespace string
	}{
		{"Deployment", BuildStatefulSet(instance).Namespace},
		{"Service", BuildService(instance).Namespace},
		{"NetworkPolicy", BuildNetworkPolicy(instance).Namespace},
		{"ServiceAccount", BuildServiceAccount(instance).Namespace},
		{"Role", BuildRole(instance).Namespace},
		{"RoleBinding", BuildRoleBinding(instance).Namespace},
		{"ConfigMap", BuildConfigMap(instance, "").Namespace},
		{"PVC", BuildPVC(instance).Namespace},
		{"PDB", BuildPDB(instance).Namespace},
		{"Ingress", BuildIngress(instance).Namespace},
	}

	for _, r := range resources {
		t.Run(r.name, func(t *testing.T) {
			if r.namespace != "production" {
				t.Errorf("%s: namespace = %q, want %q", r.name, r.namespace, "production")
			}
		})
	}
}

func TestBuildStatefulSet_ChromiumCustomResources(t *testing.T) {
	instance := newTestInstance("chromium-res")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Chromium.Resources = openclawv1alpha1.ResourcesSpec{
		Requests: openclawv1alpha1.ResourceList{
			CPU:    "500m",
			Memory: "1Gi",
		},
		Limits: openclawv1alpha1.ResourceList{
			CPU:    "2",
			Memory: "4Gi",
		},
	}

	sts := BuildStatefulSet(instance)
	var chromium *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "chromium" {
			chromium = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if chromium == nil {
		t.Fatal("chromium container not found")
	}

	cpuReq := chromium.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "500m" {
		t.Errorf("chromium cpu request = %v, want 500m", cpuReq.String())
	}
	memReq := chromium.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Errorf("chromium memory request = %v, want 1Gi", memReq.String())
	}
	cpuLim := chromium.Resources.Limits[corev1.ResourceCPU]
	if cpuLim.String() != "2" {
		t.Errorf("chromium cpu limit = %v, want 2", cpuLim.String())
	}
	memLim := chromium.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("4Gi")) != 0 {
		t.Errorf("chromium memory limit = %v, want 4Gi", memLim.String())
	}
}

func TestBuildStatefulSet_CustomPodSecurityContext(t *testing.T) {
	instance := newTestInstance("custom-psc")
	instance.Spec.Security.PodSecurityContext = &openclawv1alpha1.PodSecurityContextSpec{
		RunAsUser:  Ptr(int64(2000)),
		RunAsGroup: Ptr(int64(3000)),
		FSGroup:    Ptr(int64(4000)),
	}

	sts := BuildStatefulSet(instance)
	psc := sts.Spec.Template.Spec.SecurityContext

	if *psc.RunAsUser != 2000 {
		t.Errorf("runAsUser = %d, want 2000", *psc.RunAsUser)
	}
	if *psc.RunAsGroup != 3000 {
		t.Errorf("runAsGroup = %d, want 3000", *psc.RunAsGroup)
	}
	if *psc.FSGroup != 4000 {
		t.Errorf("fsGroup = %d, want 4000", *psc.FSGroup)
	}
	// runAsNonRoot should still be true (default)
	if !*psc.RunAsNonRoot {
		t.Error("runAsNonRoot should still be true")
	}
}

func TestBuildStatefulSet_CustomPullPolicy(t *testing.T) {
	instance := newTestInstance("pull-policy")
	instance.Spec.Image.PullPolicy = corev1.PullAlways

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("pullPolicy = %v, want Always", main.ImagePullPolicy)
	}
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func assertContainerPort(t *testing.T, ports []corev1.ContainerPort, name string, expectedPort int32) {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			if p.ContainerPort != expectedPort {
				t.Errorf("port %q = %d, want %d", name, p.ContainerPort, expectedPort)
			}
			if p.Protocol != corev1.ProtocolTCP {
				t.Errorf("port %q protocol = %v, want TCP", name, p.Protocol)
			}
			return
		}
	}
	t.Errorf("port %q not found", name)
}

func assertServicePort(t *testing.T, ports []corev1.ServicePort, name string, expectedPort int32) {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			if p.Port != expectedPort {
				t.Errorf("service port %q = %d, want %d", name, p.Port, expectedPort)
			}
			if p.TargetPort.IntValue() != int(expectedPort) {
				t.Errorf("service target port %q = %d, want %d", name, p.TargetPort.IntValue(), expectedPort)
			}
			if p.Protocol != corev1.ProtocolTCP {
				t.Errorf("service port %q protocol = %v, want TCP", name, p.Protocol)
			}
			return
		}
	}
	t.Errorf("service port %q not found", name)
}

func assertNPPort(t *testing.T, ports []networkingv1.NetworkPolicyPort, expectedPort int) {
	t.Helper()
	for _, p := range ports {
		if p.Port != nil && p.Port.IntValue() == expectedPort {
			return
		}
	}
	t.Errorf("network policy port %d not found", expectedPort)
}

func assertVolumeMount(t *testing.T, mounts []corev1.VolumeMount, name, expectedPath string) {
	t.Helper()
	for _, m := range mounts {
		if m.Name == name {
			if m.MountPath != expectedPath {
				t.Errorf("volume mount %q path = %q, want %q", name, m.MountPath, expectedPath)
			}
			return
		}
	}
	t.Errorf("volume mount %q not found", name)
}

func envNames(envs []corev1.EnvVar) []string {
	names := make([]string, len(envs))
	for i, e := range envs {
		names[i] = e.Name
	}
	return names
}

func findVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == name {
			return &volumes[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Kubernetes default field tests (regression for issue #28 — reconcile loop)
// ---------------------------------------------------------------------------

// TestBuildStatefulSet_KubernetesDefaults verifies that the StatefulSet builder
// explicitly sets all fields that Kubernetes would default on the server side.
// If any of these are missing, controllerutil.CreateOrUpdate sees a diff on
// every reconcile, causing an endless update loop.
func TestBuildStatefulSet_KubernetesDefaults(t *testing.T) {
	instance := newTestInstance("k8s-defaults")
	sts := BuildStatefulSet(instance)

	// StatefulSetSpec defaults
	if sts.Spec.RevisionHistoryLimit == nil || *sts.Spec.RevisionHistoryLimit != 10 {
		t.Errorf("RevisionHistoryLimit = %v, want 10", sts.Spec.RevisionHistoryLimit)
	}
	if sts.Spec.ServiceName != "k8s-defaults" {
		t.Errorf("ServiceName = %q, want %q", sts.Spec.ServiceName, "k8s-defaults")
	}
	if sts.Spec.PodManagementPolicy != appsv1.ParallelPodManagement {
		t.Errorf("PodManagementPolicy = %v, want Parallel", sts.Spec.PodManagementPolicy)
	}
	if sts.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		t.Errorf("UpdateStrategy = %v, want RollingUpdate", sts.Spec.UpdateStrategy.Type)
	}

	// PodSpec defaults
	podSpec := sts.Spec.Template.Spec
	if podSpec.RestartPolicy != corev1.RestartPolicyAlways {
		t.Errorf("RestartPolicy = %v, want Always", podSpec.RestartPolicy)
	}
	if podSpec.DNSPolicy != corev1.DNSClusterFirst {
		t.Errorf("DNSPolicy = %v, want ClusterFirst", podSpec.DNSPolicy)
	}
	if podSpec.SchedulerName != corev1.DefaultSchedulerName {
		t.Errorf("SchedulerName = %v, want %v", podSpec.SchedulerName, corev1.DefaultSchedulerName)
	}
	if podSpec.TerminationGracePeriodSeconds == nil || *podSpec.TerminationGracePeriodSeconds != 30 {
		t.Errorf("TerminationGracePeriodSeconds = %v, want 30", podSpec.TerminationGracePeriodSeconds)
	}

	// Container defaults
	main := sts.Spec.Template.Spec.Containers[0]
	if main.TerminationMessagePath != corev1.TerminationMessagePathDefault {
		t.Errorf("TerminationMessagePath = %q, want %q", main.TerminationMessagePath, corev1.TerminationMessagePathDefault)
	}
	if main.TerminationMessagePolicy != corev1.TerminationMessageReadFile {
		t.Errorf("TerminationMessagePolicy = %v, want File", main.TerminationMessagePolicy)
	}

	// Probe successThreshold defaults
	if main.LivenessProbe.SuccessThreshold != 1 {
		t.Errorf("LivenessProbe.SuccessThreshold = %d, want 1", main.LivenessProbe.SuccessThreshold)
	}
	if main.ReadinessProbe.SuccessThreshold != 1 {
		t.Errorf("ReadinessProbe.SuccessThreshold = %d, want 1", main.ReadinessProbe.SuccessThreshold)
	}
	if main.StartupProbe.SuccessThreshold != 1 {
		t.Errorf("StartupProbe.SuccessThreshold = %d, want 1", main.StartupProbe.SuccessThreshold)
	}
}

// TestBuildStatefulSet_InitContainerDefaults verifies init containers include
// Kubernetes default fields to avoid reconcile-loop drift.
func TestBuildStatefulSet_InitContainerDefaults(t *testing.T) {
	instance := newTestInstance("init-defaults")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	sts := BuildStatefulSet(instance)
	if len(sts.Spec.Template.Spec.InitContainers) == 0 {
		t.Fatal("expected init container when raw config is set")
	}

	init := sts.Spec.Template.Spec.InitContainers[0]
	if init.TerminationMessagePath != corev1.TerminationMessagePathDefault {
		t.Errorf("init container TerminationMessagePath = %q, want %q", init.TerminationMessagePath, corev1.TerminationMessagePathDefault)
	}
	if init.TerminationMessagePolicy != corev1.TerminationMessageReadFile {
		t.Errorf("init container TerminationMessagePolicy = %v, want File", init.TerminationMessagePolicy)
	}
	if init.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("init container ImagePullPolicy = %v, want IfNotPresent", init.ImagePullPolicy)
	}
}

// TestBuildStatefulSet_ChromiumContainerDefaults verifies the chromium sidecar
// includes Kubernetes default fields.
func TestBuildStatefulSet_ChromiumContainerDefaults(t *testing.T) {
	instance := newTestInstance("chromium-defaults")
	instance.Spec.Chromium.Enabled = true

	sts := BuildStatefulSet(instance)
	if len(sts.Spec.Template.Spec.Containers) < 2 {
		t.Fatal("expected chromium sidecar container")
	}

	chromium := sts.Spec.Template.Spec.Containers[1]
	if chromium.TerminationMessagePath != corev1.TerminationMessagePathDefault {
		t.Errorf("chromium TerminationMessagePath = %q, want %q", chromium.TerminationMessagePath, corev1.TerminationMessagePathDefault)
	}
	if chromium.TerminationMessagePolicy != corev1.TerminationMessageReadFile {
		t.Errorf("chromium TerminationMessagePolicy = %v, want File", chromium.TerminationMessagePolicy)
	}
}

// TestBuildStatefulSet_ConfigMapDefaultMode verifies the ConfigMap volume
// explicitly sets DefaultMode to match the Kubernetes default (0644).
func TestBuildStatefulSet_ConfigMapDefaultMode(t *testing.T) {
	instance := newTestInstance("cm-default-mode")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	sts := BuildStatefulSet(instance)
	configVol := findVolume(sts.Spec.Template.Spec.Volumes, "config")
	if configVol == nil {
		t.Fatal("config volume not found")
	}
	if configVol.ConfigMap == nil {
		t.Fatal("config volume should use ConfigMap")
	}
	if configVol.ConfigMap.DefaultMode == nil || *configVol.ConfigMap.DefaultMode != 0o644 {
		t.Errorf("ConfigMap DefaultMode = %v, want 0o644", configVol.ConfigMap.DefaultMode)
	}
}

// TestBuildService_KubernetesDefaults verifies Service builder includes
// Kubernetes default fields.
func TestBuildService_KubernetesDefaults(t *testing.T) {
	instance := newTestInstance("svc-defaults")
	svc := BuildService(instance)

	if svc.Spec.SessionAffinity != corev1.ServiceAffinityNone {
		t.Errorf("SessionAffinity = %v, want None", svc.Spec.SessionAffinity)
	}
}

// TestBuildStatefulSet_Idempotent verifies calling BuildStatefulSet twice with
// the same input produces identical specs (no random maps, no pointer aliasing
// issues). This is essential for CreateOrUpdate comparisons to work.
func TestBuildStatefulSet_Idempotent(t *testing.T) {
	instance := newTestInstance("idempotent")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"key":"val"}`)},
	}
	instance.Spec.Chromium.Enabled = true

	dep1 := BuildStatefulSet(instance)
	dep2 := BuildStatefulSet(instance)

	b1, _ := json.Marshal(dep1.Spec)
	b2, _ := json.Marshal(dep2.Spec)

	if !bytes.Equal(b1, b2) {
		t.Error("BuildStatefulSet is not idempotent — two calls with the same input produce different specs")
	}
}

// ---------------------------------------------------------------------------
// workspace_configmap.go tests
// ---------------------------------------------------------------------------

func TestBuildWorkspaceConfigMap_Nil(t *testing.T) {
	instance := newTestInstance("ws-nil")
	instance.Spec.Workspace = nil

	cm := BuildWorkspaceConfigMap(instance)
	if cm != nil {
		t.Fatal("expected nil ConfigMap when workspace is nil")
	}
}

func TestBuildWorkspaceConfigMap_EmptyFiles(t *testing.T) {
	instance := newTestInstance("ws-empty")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialDirectories: []string{"memory"},
	}

	cm := BuildWorkspaceConfigMap(instance)
	if cm != nil {
		t.Fatal("expected nil ConfigMap when initialFiles is empty")
	}
}

func TestBuildWorkspaceConfigMap_WithFiles(t *testing.T) {
	instance := newTestInstance("ws-files")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"SOUL.md":   "# Personality\nBe helpful.",
			"AGENTS.md": "# Agents config",
		},
	}

	cm := BuildWorkspaceConfigMap(instance)
	if cm == nil {
		t.Fatal("expected non-nil ConfigMap when files are set")
	}
	if cm.Name != "ws-files-workspace" {
		t.Errorf("ConfigMap name = %q, want %q", cm.Name, "ws-files-workspace")
	}
	if cm.Namespace != "test-ns" {
		t.Errorf("ConfigMap namespace = %q, want %q", cm.Namespace, "test-ns")
	}
	if len(cm.Data) != 2 {
		t.Fatalf("expected 2 data entries, got %d", len(cm.Data))
	}
	if cm.Data["SOUL.md"] != "# Personality\nBe helpful." {
		t.Errorf("SOUL.md content mismatch")
	}
	if cm.Data["AGENTS.md"] != "# Agents config" {
		t.Errorf("AGENTS.md content mismatch")
	}
}

func TestWorkspaceConfigMapName(t *testing.T) {
	instance := newTestInstance("foo")
	if got := WorkspaceConfigMapName(instance); got != "foo-workspace" {
		t.Errorf("WorkspaceConfigMapName() = %q, want %q", got, "foo-workspace")
	}
}

// ---------------------------------------------------------------------------
// BuildInitScript tests
// ---------------------------------------------------------------------------

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"it's", "'it'\\''s'"},
		{"no quotes", "'no quotes'"},
		{"a'b'c", "'a'\\''b'\\''c'"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildInitScript_ConfigOnly(t *testing.T) {
	instance := newTestInstance("init-config-only")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	script := BuildInitScript(instance)
	if script != "cp /config/'openclaw.json' /data/openclaw.json" {
		t.Errorf("unexpected script:\n%s", script)
	}
}

func TestBuildInitScript_WorkspaceOnly(t *testing.T) {
	instance := newTestInstance("init-ws-only")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"SOUL.md": "content",
		},
		InitialDirectories: []string{"memory"},
	}

	script := BuildInitScript(instance)
	expected := "cp /config/'openclaw.json' /data/openclaw.json\nmkdir -p /data/workspace/'memory'\nmkdir -p /data/workspace\n[ -f /data/workspace/'SOUL.md' ] || cp /workspace-init/'SOUL.md' /data/workspace/'SOUL.md'"
	if script != expected {
		t.Errorf("unexpected script:\ngot:  %q\nwant: %q", script, expected)
	}
}

func TestBuildInitScript_Both(t *testing.T) {
	instance := newTestInstance("init-both")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"SOUL.md":   "soul",
			"AGENTS.md": "agents",
		},
		InitialDirectories: []string{"memory", "tools"},
	}

	script := BuildInitScript(instance)

	// Verify all expected lines are present (sorted order)
	lines := strings.Split(script, "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d:\n%s", len(lines), script)
	}
	if lines[0] != "cp /config/'openclaw.json' /data/openclaw.json" {
		t.Errorf("line 0: %q", lines[0])
	}
	if lines[1] != "mkdir -p /data/workspace/'memory'" {
		t.Errorf("line 1: %q", lines[1])
	}
	if lines[2] != "mkdir -p /data/workspace/'tools'" {
		t.Errorf("line 2: %q", lines[2])
	}
	if lines[3] != "mkdir -p /data/workspace" {
		t.Errorf("line 3: %q", lines[3])
	}
	if lines[4] != "[ -f /data/workspace/'AGENTS.md' ] || cp /workspace-init/'AGENTS.md' /data/workspace/'AGENTS.md'" {
		t.Errorf("line 4: %q", lines[4])
	}
	if lines[5] != "[ -f /data/workspace/'SOUL.md' ] || cp /workspace-init/'SOUL.md' /data/workspace/'SOUL.md'" {
		t.Errorf("line 5: %q", lines[5])
	}
}

func TestBuildInitScript_DirsOnly(t *testing.T) {
	instance := newTestInstance("init-dirs-only")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialDirectories: []string{"memory", "tools/scripts"},
	}

	script := BuildInitScript(instance)
	expected := "cp /config/'openclaw.json' /data/openclaw.json\nmkdir -p /data/workspace/'memory'\nmkdir -p /data/workspace/'tools/scripts'"
	if script != expected {
		t.Errorf("unexpected script:\ngot:  %q\nwant: %q", script, expected)
	}
}

func TestBuildInitScript_ShellQuotesSpecialChars(t *testing.T) {
	instance := newTestInstance("init-special")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"it's a file.md": "content",
		},
	}

	script := BuildInitScript(instance)
	expected := "cp /config/'openclaw.json' /data/openclaw.json\nmkdir -p /data/workspace\n[ -f /data/workspace/'it'\\''s a file.md' ] || cp /workspace-init/'it'\\''s a file.md' /data/workspace/'it'\\''s a file.md'"
	if script != expected {
		t.Errorf("unexpected script:\ngot:  %q\nwant: %q", script, expected)
	}
}

func TestBuildInitScript_FilesOnly_MkdirWorkspace(t *testing.T) {
	// Regression test: files without directories must still mkdir /data/workspace
	// so that cp doesn't fail on first run with emptyDir.
	instance := newTestInstance("init-files-only")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"README.md": "hello",
		},
	}

	script := BuildInitScript(instance)
	if !strings.Contains(script, "mkdir -p /data/workspace\n") {
		t.Errorf("script should contain mkdir -p /data/workspace, got:\n%s", script)
	}
}

func TestBuildInitScript_VanillaDeployment(t *testing.T) {
	instance := newTestInstance("init-empty")
	script := BuildInitScript(instance)
	// Vanilla deployments now get a config copy (gateway.bind=lan)
	expected := "cp /config/'openclaw.json' /data/openclaw.json"
	if script != expected {
		t.Errorf("unexpected script:\ngot:  %q\nwant: %q", script, expected)
	}
}

// ---------------------------------------------------------------------------
// Config hash includes workspace
// ---------------------------------------------------------------------------

func TestConfigHash_ChangesWithWorkspace(t *testing.T) {
	instance := newTestInstance("hash-ws")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	dep1 := BuildStatefulSet(instance)
	hash1 := dep1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{"SOUL.md": "hello"},
	}

	dep2 := BuildStatefulSet(instance)
	hash2 := dep2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when workspace is added")
	}
}

func TestConfigHash_ChangesWithFileContent(t *testing.T) {
	instance := newTestInstance("hash-content")
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{"SOUL.md": "v1"},
	}

	dep1 := BuildStatefulSet(instance)
	hash1 := dep1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.Workspace.InitialFiles["SOUL.md"] = "v2"

	dep2 := BuildStatefulSet(instance)
	hash2 := dep2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when workspace file content changes")
	}
}

// ---------------------------------------------------------------------------
// Workspace volume and volume mount tests
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_WorkspaceVolume(t *testing.T) {
	instance := newTestInstance("ws-vol")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{"SOUL.md": "hello"},
	}

	sts := BuildStatefulSet(instance)

	// Verify workspace-init volume exists
	wsVol := findVolume(sts.Spec.Template.Spec.Volumes, "workspace-init")
	if wsVol == nil {
		t.Fatal("workspace-init volume not found")
	}
	if wsVol.ConfigMap == nil {
		t.Fatal("workspace-init volume should use ConfigMap")
	}
	if wsVol.ConfigMap.Name != "ws-vol-workspace" {
		t.Errorf("workspace-init ConfigMap name = %q, want %q", wsVol.ConfigMap.Name, "ws-vol-workspace")
	}

	// Verify init container has workspace-init mount
	init := sts.Spec.Template.Spec.InitContainers[0]
	assertVolumeMount(t, init.VolumeMounts, "workspace-init", "/workspace-init")
}

func TestBuildStatefulSet_NoWorkspaceVolume(t *testing.T) {
	instance := newTestInstance("no-ws-vol")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}

	sts := BuildStatefulSet(instance)

	// No workspace-init volume
	wsVol := findVolume(sts.Spec.Template.Spec.Volumes, "workspace-init")
	if wsVol != nil {
		t.Error("workspace-init volume should not exist without workspace files")
	}
}

func TestBuildStatefulSet_WorkspaceDirsOnly_NoVolume(t *testing.T) {
	instance := newTestInstance("ws-dirs-no-vol")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialDirectories: []string{"memory"},
	}

	sts := BuildStatefulSet(instance)

	// Dirs only — no workspace-init volume needed (no files to mount)
	wsVol := findVolume(sts.Spec.Template.Spec.Volumes, "workspace-init")
	if wsVol != nil {
		t.Error("workspace-init volume should not exist with only directories")
	}

	// But init container should still exist (for mkdir commands)
	if len(sts.Spec.Template.Spec.InitContainers) == 0 {
		t.Fatal("expected init container for workspace directories")
	}
}

func TestBuildStatefulSet_Idempotent_WithWorkspace(t *testing.T) {
	instance := newTestInstance("idempotent-ws")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"key":"val"}`)},
	}
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles:       map[string]string{"SOUL.md": "hello", "AGENTS.md": "agents"},
		InitialDirectories: []string{"memory", "tools"},
	}

	dep1 := BuildStatefulSet(instance)
	dep2 := BuildStatefulSet(instance)

	b1, _ := json.Marshal(dep1.Spec)
	b2, _ := json.Marshal(dep2.Spec)

	if !bytes.Equal(b1, b2) {
		t.Error("BuildStatefulSet with workspace is not idempotent")
	}
}

// ---------------------------------------------------------------------------
// Feature 1: ReadOnlyRootFilesystem tests
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_ReadOnlyRootFilesystem_Default(t *testing.T) {
	instance := newTestInstance("rorfs-default")
	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	csc := main.SecurityContext
	if csc.ReadOnlyRootFilesystem == nil || !*csc.ReadOnlyRootFilesystem {
		t.Error("readOnlyRootFilesystem should default to true")
	}
}

func TestBuildStatefulSet_ReadOnlyRootFilesystem_ExplicitFalse(t *testing.T) {
	instance := newTestInstance("rorfs-false")
	instance.Spec.Security.ContainerSecurityContext = &openclawv1alpha1.ContainerSecurityContextSpec{
		ReadOnlyRootFilesystem: Ptr(false),
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	if main.SecurityContext.ReadOnlyRootFilesystem == nil || *main.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("readOnlyRootFilesystem should be false when explicitly overridden")
	}
}

func TestBuildStatefulSet_TmpVolumeAndMount(t *testing.T) {
	instance := newTestInstance("tmp-vol")
	sts := BuildStatefulSet(instance)

	// Check /tmp volume mount on main container
	main := sts.Spec.Template.Spec.Containers[0]
	assertVolumeMount(t, main.VolumeMounts, "tmp", "/tmp")

	// Check tmp volume exists as emptyDir
	tmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "tmp")
	if tmpVol == nil {
		t.Fatal("tmp volume not found")
	}
	if tmpVol.EmptyDir == nil {
		t.Error("tmp volume should be emptyDir")
	}
}

// ---------------------------------------------------------------------------
// Feature 2: Config merge mode tests
// ---------------------------------------------------------------------------

func TestBuildInitScript_OverwriteMode(t *testing.T) {
	instance := newTestInstance("init-overwrite")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = "overwrite"

	script := BuildInitScript(instance)
	if !strings.Contains(script, "cp /config/") {
		t.Errorf("overwrite mode should use cp, got: %q", script)
	}
	if strings.Contains(script, "node -e") {
		t.Error("overwrite mode should not use node deep merge")
	}
}

func TestBuildInitScript_MergeMode(t *testing.T) {
	instance := newTestInstance("init-merge")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = ConfigMergeModeMerge

	script := BuildInitScript(instance)
	// Merge now uses Node.js deep merge instead of jq (jq image is distroless, no shell)
	if !strings.Contains(script, "node -e") {
		t.Errorf("merge mode should use node deep merge, got: %q", script)
	}
	if !strings.Contains(script, "fs.existsSync") {
		t.Errorf("merge mode should check for existing file via fs.existsSync, got: %q", script)
	}
	if !strings.Contains(script, "/tmp/merged.json") {
		t.Errorf("merge mode should write to /tmp/merged.json atomically, got: %q", script)
	}
	// Regression: renameSync fails across mount boundaries (EXDEV) - #120
	if strings.Contains(script, "renameSync") {
		t.Errorf("merge mode must not use renameSync (fails cross-device between /tmp and /data), got: %q", script)
	}
	if !strings.Contains(script, "copyFileSync") {
		t.Errorf("merge mode should use copyFileSync to move merged config to /data, got: %q", script)
	}
}

func TestBuildStatefulSet_MergeMode_OpenClawImage(t *testing.T) {
	instance := newTestInstance("merge-oci")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = ConfigMergeModeMerge

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) == 0 {
		t.Fatal("expected init container for merge mode")
	}

	initC := initContainers[0]
	wantImage := GetImage(instance)
	if initC.Image != wantImage {
		t.Errorf("merge mode init container image = %q, want %q", initC.Image, wantImage)
	}

	// Should have init-tmp mount
	assertVolumeMount(t, initC.VolumeMounts, "init-tmp", "/tmp")

	// Should have HOME and NPM_CONFIG_CACHE env vars
	var hasHome, hasNpm bool
	for _, e := range initC.Env {
		if e.Name == "HOME" && e.Value == "/tmp" {
			hasHome = true
		}
		if e.Name == "NPM_CONFIG_CACHE" && e.Value == "/tmp/.npm" {
			hasNpm = true
		}
	}
	if !hasHome {
		t.Error("merge mode init container should have HOME=/tmp")
	}
	if !hasNpm {
		t.Error("merge mode init container should have NPM_CONFIG_CACHE=/tmp/.npm")
	}

	// Should have writable rootfs (Node.js may need it)
	if initC.SecurityContext.ReadOnlyRootFilesystem == nil || *initC.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("merge mode init container should have writable rootfs")
	}
}

func TestBuildStatefulSet_OverwriteMode_BusyboxImage(t *testing.T) {
	instance := newTestInstance("overwrite-bb")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = "overwrite"

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) == 0 {
		t.Fatal("expected init container for overwrite mode")
	}

	initC := initContainers[0]
	if initC.Image != "busybox:1.37" {
		t.Errorf("overwrite mode init container image = %q, want busybox:1.37", initC.Image)
	}
}

func TestBuildStatefulSet_MergeMode_InitTmpVolume(t *testing.T) {
	instance := newTestInstance("merge-vol")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = ConfigMergeModeMerge

	sts := BuildStatefulSet(instance)
	initTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "init-tmp")
	if initTmpVol == nil {
		t.Fatal("init-tmp volume not found in merge mode")
	}
	if initTmpVol.EmptyDir == nil {
		t.Error("init-tmp volume should be emptyDir")
	}
}

func TestBuildStatefulSet_OverwriteMode_NoInitTmpVolume(t *testing.T) {
	instance := newTestInstance("overwrite-vol")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Config.MergeMode = "overwrite"

	sts := BuildStatefulSet(instance)
	initTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "init-tmp")
	if initTmpVol != nil {
		t.Error("init-tmp volume should not exist in overwrite mode")
	}
}

func TestBuildInitScript_MergeMode_NoConfig(t *testing.T) {
	instance := newTestInstance("merge-no-cfg")
	instance.Spec.Config.MergeMode = ConfigMergeModeMerge
	// No raw config set — but operator always creates a ConfigMap (gateway.bind)

	script := BuildInitScript(instance)
	// Should produce a merge script since configMapKey() now always returns "openclaw.json"
	if !strings.Contains(script, "node -e") {
		t.Errorf("merge mode should produce node deep merge script, got: %q", script)
	}
}

// ---------------------------------------------------------------------------
// Feature 3: Declarative skill installation tests
// ---------------------------------------------------------------------------

func TestParseSkillEntry_ClawHub(t *testing.T) {
	got := parseSkillEntry("@anthropic/mcp-server-fetch")
	want := "npx -y clawhub install '@anthropic/mcp-server-fetch'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseSkillEntry_Npm(t *testing.T) {
	got := parseSkillEntry("npm:@openclaw/matrix")
	want := "cd /home/openclaw/.openclaw && npm install '@openclaw/matrix'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseSkillEntry_NpmUnscoped(t *testing.T) {
	got := parseSkillEntry("npm:some-package")
	want := "cd /home/openclaw/.openclaw && npm install 'some-package'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildSkillsScript_NoSkills(t *testing.T) {
	instance := newTestInstance("no-skills")
	script := BuildSkillsScript(instance)
	if script != "" {
		t.Errorf("expected empty script, got: %q", script)
	}
}

func TestBuildSkillsScript_WithSkills(t *testing.T) {
	instance := newTestInstance("with-skills")
	instance.Spec.Skills = []string{"@anthropic/mcp-server-fetch", "@github/copilot-skill"}

	script := BuildSkillsScript(instance)

	// Skills should be sorted
	lines := strings.Split(script, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), script)
	}
	if lines[0] != "npx -y clawhub install '@anthropic/mcp-server-fetch'" {
		t.Errorf("line 0: %q", lines[0])
	}
	if lines[1] != "npx -y clawhub install '@github/copilot-skill'" {
		t.Errorf("line 1: %q", lines[1])
	}
}

func TestBuildSkillsScript_MixedPrefixes(t *testing.T) {
	instance := newTestInstance("mixed-skills")
	instance.Spec.Skills = []string{
		"npm:@openclaw/matrix",
		"@anthropic/mcp-server-fetch",
		"npm:some-tool",
	}

	script := BuildSkillsScript(instance)

	lines := strings.Split(script, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), script)
	}
	// Sorted: @anthropic/... < npm:@openclaw/... < npm:some-tool
	if lines[0] != "npx -y clawhub install '@anthropic/mcp-server-fetch'" {
		t.Errorf("line 0: %q", lines[0])
	}
	if lines[1] != "cd /home/openclaw/.openclaw && npm install '@openclaw/matrix'" {
		t.Errorf("line 1: %q", lines[1])
	}
	if lines[2] != "cd /home/openclaw/.openclaw && npm install 'some-tool'" {
		t.Errorf("line 2: %q", lines[2])
	}
}

func TestBuildStatefulSet_NoSkills_NoInitSkillsContainer(t *testing.T) {
	instance := newTestInstance("no-skills-sts")

	sts := BuildStatefulSet(instance)
	for _, c := range sts.Spec.Template.Spec.InitContainers {
		if c.Name == "init-skills" {
			t.Error("init-skills container should not exist without skills")
		}
	}
}

func TestBuildStatefulSet_WithSkills_InitSkillsContainer(t *testing.T) {
	instance := newTestInstance("skills-sts")
	instance.Spec.Skills = []string{"@anthropic/mcp-server-fetch"}

	sts := BuildStatefulSet(instance)

	var skillsContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-skills" {
			skillsContainer = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if skillsContainer == nil {
		t.Fatal("init-skills container not found")
	}

	// Should use same image as main container
	expectedImage := GetImage(instance)
	if skillsContainer.Image != expectedImage {
		t.Errorf("init-skills image = %q, want %q", skillsContainer.Image, expectedImage)
	}

	// Should have HOME and NPM_CONFIG_CACHE env vars
	envMap := map[string]string{}
	for _, e := range skillsContainer.Env {
		envMap[e.Name] = e.Value
	}
	if envMap["HOME"] != "/tmp" {
		t.Errorf("init-skills HOME = %q, want /tmp", envMap["HOME"])
	}
	if envMap["NPM_CONFIG_CACHE"] != "/tmp/.npm" {
		t.Errorf("init-skills NPM_CONFIG_CACHE = %q, want /tmp/.npm", envMap["NPM_CONFIG_CACHE"])
	}

	// Should have data and skills-tmp mounts
	assertVolumeMount(t, skillsContainer.VolumeMounts, "data", "/home/openclaw/.openclaw")
	assertVolumeMount(t, skillsContainer.VolumeMounts, "skills-tmp", "/tmp")

	// Security context should be restricted
	sc := skillsContainer.SecurityContext
	if sc == nil {
		t.Fatal("init-skills security context is nil")
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		t.Error("init-skills: allowPrivilegeEscalation should be false")
	}
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		t.Error("init-skills: runAsNonRoot should be true")
	}

	// NPM_CONFIG_IGNORE_SCRIPTS must be set to mitigate supply chain attacks (#91)
	if envMap["NPM_CONFIG_IGNORE_SCRIPTS"] != "true" {
		t.Errorf("init-skills NPM_CONFIG_IGNORE_SCRIPTS = %q, want \"true\"", envMap["NPM_CONFIG_IGNORE_SCRIPTS"])
	}
}

func TestBuildStatefulSet_WithNpmSkill_InitSkillsScript(t *testing.T) {
	instance := newTestInstance("npm-skill-sts")
	instance.Spec.Skills = []string{"npm:@openclaw/matrix"}

	sts := BuildStatefulSet(instance)

	var skillsContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-skills" {
			skillsContainer = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if skillsContainer == nil {
		t.Fatal("init-skills container not found")
	}

	// Command should use npm install, not clawhub
	script := skillsContainer.Command[2]
	if !strings.Contains(script, "npm install") {
		t.Errorf("expected npm install in script, got: %q", script)
	}
	if strings.Contains(script, "clawhub") {
		t.Errorf("npm: prefixed skill should not use clawhub, got: %q", script)
	}
}

func TestBuildStatefulSet_WithSkills_SkillsTmpVolume(t *testing.T) {
	instance := newTestInstance("skills-vol")
	instance.Spec.Skills = []string{"some-skill"}

	sts := BuildStatefulSet(instance)
	skillsTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "skills-tmp")
	if skillsTmpVol == nil {
		t.Fatal("skills-tmp volume not found")
	}
	if skillsTmpVol.EmptyDir == nil {
		t.Error("skills-tmp volume should be emptyDir")
	}
}

func TestBuildStatefulSet_NoSkills_NoSkillsTmpVolume(t *testing.T) {
	instance := newTestInstance("no-skills-vol")

	sts := BuildStatefulSet(instance)
	skillsTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "skills-tmp")
	if skillsTmpVol != nil {
		t.Error("skills-tmp volume should not exist without skills")
	}
}

func TestConfigHash_ChangesWithSkills(t *testing.T) {
	instance := newTestInstance("hash-skills")

	dep1 := BuildStatefulSet(instance)
	hash1 := dep1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.Skills = []string{"new-skill"}

	dep2 := BuildStatefulSet(instance)
	hash2 := dep2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when skills are added")
	}
}

func TestBuildStatefulSet_SkillsOnly_HasBothInitContainers(t *testing.T) {
	instance := newTestInstance("skills-only")
	instance.Spec.Skills = []string{"some-skill"}
	// No raw config set — but operator always creates config (gateway.bind)

	sts := BuildStatefulSet(instance)

	// Should have init-config container (gateway.bind=lan config)
	foundConfig := false
	foundSkills := false
	for _, c := range sts.Spec.Template.Spec.InitContainers {
		if c.Name == "init-config" {
			foundConfig = true
		}
		if c.Name == "init-skills" {
			foundSkills = true
		}
	}
	if !foundConfig {
		t.Error("init-config container should exist (gateway.bind config)")
	}
	if !foundSkills {
		t.Error("init-skills container should exist with skills defined")
	}
}

// secret.go tests — gateway token Secret
// ---------------------------------------------------------------------------

func TestGatewayTokenSecretName(t *testing.T) {
	instance := newTestInstance("my-app")
	got := GatewayTokenSecretName(instance)
	if got != "my-app-gateway-token" {
		t.Errorf("GatewayTokenSecretName() = %q, want %q", got, "my-app-gateway-token")
	}
}

func TestBuildGatewayTokenSecret(t *testing.T) {
	instance := newTestInstance("my-app")
	token := "abcdef1234567890abcdef1234567890"

	secret := BuildGatewayTokenSecret(instance, token)

	if secret.Name != "my-app-gateway-token" {
		t.Errorf("secret name = %q, want %q", secret.Name, "my-app-gateway-token")
	}
	if secret.Namespace != "test-ns" {
		t.Errorf("secret namespace = %q, want %q", secret.Namespace, "test-ns")
	}
	if secret.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("secret missing app label")
	}
	if string(secret.Data[GatewayTokenSecretKey]) != token {
		t.Errorf("secret data[%q] = %q, want %q", GatewayTokenSecretKey, string(secret.Data[GatewayTokenSecretKey]), token)
	}
}

// ---------------------------------------------------------------------------
// configmap.go tests — gateway auth enrichment
// ---------------------------------------------------------------------------

func TestEnrichConfigWithGatewayAuth_InjectsToken(t *testing.T) {
	configJSON := []byte(`{"channels":{"slack":{"enabled":true}}}`)
	token := "my-test-token"

	result, err := enrichConfigWithGatewayAuth(configJSON, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key in config")
	}
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway.auth key in config")
	}
	if auth["mode"] != "token" {
		t.Errorf("gateway.auth.mode = %v, want %q", auth["mode"], "token")
	}
	if auth["token"] != token {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], token)
	}
}

func TestEnrichConfigWithGatewayAuth_PreservesUserToken(t *testing.T) {
	configJSON := []byte(`{"gateway":{"auth":{"mode":"token","token":"user-token-123"}}}`)
	token := "operator-generated-token"

	result, err := enrichConfigWithGatewayAuth(configJSON, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	auth := gw["auth"].(map[string]interface{})

	// User's token should be preserved, not overwritten
	if auth["token"] != "user-token-123" {
		t.Errorf("gateway.auth.token = %v, want %q (user's value should be preserved)", auth["token"], "user-token-123")
	}
}

func TestEnrichConfigWithGatewayAuth_EmptyConfig(t *testing.T) {
	configJSON := []byte(`{}`)
	token := "my-token"

	result, err := enrichConfigWithGatewayAuth(configJSON, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	auth := gw["auth"].(map[string]interface{})
	if auth["mode"] != "token" {
		t.Errorf("gateway.auth.mode = %v, want %q", auth["mode"], "token")
	}
	if auth["token"] != "my-token" {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], "my-token")
	}
}

func TestEnrichConfigWithGatewayAuth_InvalidJSON(t *testing.T) {
	configJSON := []byte(`not json`)
	token := "my-token"

	result, err := enrichConfigWithGatewayAuth(configJSON, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return unchanged
	if !bytes.Equal(result, configJSON) {
		t.Errorf("expected unchanged result for invalid JSON, got %s", string(result))
	}
}

func TestBuildConfigMap_WithGatewayToken(t *testing.T) {
	instance := newTestInstance("gw-test")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"channels":{"slack":{"enabled":true}}}`),
		},
	}
	token := "abc123"

	cm := BuildConfigMap(instance, token)

	configContent := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(configContent), &parsed); err != nil {
		t.Fatalf("failed to parse ConfigMap data: %v", err)
	}

	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key in ConfigMap config")
	}
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway.auth key in ConfigMap config")
	}
	if auth["token"] != token {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], token)
	}
}

func TestBuildConfigMap_WithGatewayToken_NoRawConfig(t *testing.T) {
	instance := newTestInstance("gw-noraw")
	// No raw config set
	token := "abc123"

	cm := BuildConfigMap(instance, token)

	configContent := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(configContent), &parsed); err != nil {
		t.Fatalf("failed to parse ConfigMap data: %v", err)
	}

	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key in ConfigMap config even with no raw config")
	}
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway.auth in ConfigMap config")
	}
	if auth["token"] != token {
		t.Errorf("gateway.auth.token = %v, want %q", auth["token"], token)
	}
	// bind=lan should also be present
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
}

func TestBuildConfigMap_EmptyGatewayToken(t *testing.T) {
	instance := newTestInstance("gw-empty")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"key":"value"}`),
		},
	}

	cm := BuildConfigMap(instance, "")

	configContent := cm.Data["openclaw.json"]
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(configContent), &parsed); err != nil {
		t.Fatalf("failed to parse ConfigMap data: %v", err)
	}

	// gateway key IS present (gateway.bind=lan), but no auth when token is empty
	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gateway key (bind is always injected)")
	}
	if gw["bind"] != "lan" {
		t.Errorf("gateway.bind = %v, want %q", gw["bind"], "lan")
	}
	if _, ok := gw["auth"]; ok {
		t.Error("gateway.auth should not be present when token is empty")
	}
}

// ---------------------------------------------------------------------------
// statefulset.go tests — gateway token env + Bonjour disable
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_DisableBonjour(t *testing.T) {
	instance := newTestInstance("bonjour-test")
	sts := BuildStatefulSet(instance)

	main := sts.Spec.Template.Spec.Containers[0]
	found := false
	for _, env := range main.Env {
		if env.Name == "OPENCLAW_DISABLE_BONJOUR" {
			found = true
			if env.Value != "1" {
				t.Errorf("OPENCLAW_DISABLE_BONJOUR = %q, want %q", env.Value, "1")
			}
			break
		}
	}
	if !found {
		t.Error("OPENCLAW_DISABLE_BONJOUR env var should always be present")
	}
}

func TestBuildStatefulSet_GatewayTokenEnv(t *testing.T) {
	instance := newTestInstance("gw-env-test")
	secretName := "gw-env-test-gateway-token"

	sts := BuildStatefulSet(instance, secretName)

	main := sts.Spec.Template.Spec.Containers[0]
	var gwEnv *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "OPENCLAW_GATEWAY_TOKEN" {
			gwEnv = &main.Env[i]
			break
		}
	}

	if gwEnv == nil {
		t.Fatal("OPENCLAW_GATEWAY_TOKEN env var not found")
	}
	if gwEnv.ValueFrom == nil || gwEnv.ValueFrom.SecretKeyRef == nil {
		t.Fatal("OPENCLAW_GATEWAY_TOKEN should use SecretKeyRef")
	}
	if gwEnv.ValueFrom.SecretKeyRef.Name != secretName {
		t.Errorf("secret name = %q, want %q", gwEnv.ValueFrom.SecretKeyRef.Name, secretName)
	}
	if gwEnv.ValueFrom.SecretKeyRef.Key != GatewayTokenSecretKey {
		t.Errorf("secret key = %q, want %q", gwEnv.ValueFrom.SecretKeyRef.Key, GatewayTokenSecretKey)
	}
}

func TestBuildStatefulSet_GatewayTokenEnv_UserOverride(t *testing.T) {
	instance := newTestInstance("gw-override")
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "OPENCLAW_GATEWAY_TOKEN", Value: "user-provided-token"},
	}
	secretName := "gw-override-gateway-token"

	sts := BuildStatefulSet(instance, secretName)

	main := sts.Spec.Template.Spec.Containers[0]
	// Count occurrences of OPENCLAW_GATEWAY_TOKEN
	count := 0
	for _, env := range main.Env {
		if env.Name == "OPENCLAW_GATEWAY_TOKEN" {
			count++
			// The one present should be the user's value, not a SecretKeyRef
			if env.Value != "user-provided-token" {
				t.Errorf("OPENCLAW_GATEWAY_TOKEN value = %q, want %q (user's value)", env.Value, "user-provided-token")
			}
			if env.ValueFrom != nil {
				t.Error("user's OPENCLAW_GATEWAY_TOKEN should not use SecretKeyRef")
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 OPENCLAW_GATEWAY_TOKEN env var, got %d", count)
	}
}

func TestBuildStatefulSet_ExistingSecret(t *testing.T) {
	instance := newTestInstance("existing-secret")
	instance.Spec.Gateway.ExistingSecret = "my-custom-secret"
	existingSecretName := "my-custom-secret"

	sts := BuildStatefulSet(instance, existingSecretName)

	main := sts.Spec.Template.Spec.Containers[0]
	var gwEnv *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "OPENCLAW_GATEWAY_TOKEN" {
			gwEnv = &main.Env[i]
			break
		}
	}

	if gwEnv == nil {
		t.Fatal("OPENCLAW_GATEWAY_TOKEN env var not found")
	}
	if gwEnv.ValueFrom == nil || gwEnv.ValueFrom.SecretKeyRef == nil {
		t.Fatal("OPENCLAW_GATEWAY_TOKEN should use SecretKeyRef")
	}
	if gwEnv.ValueFrom.SecretKeyRef.Name != existingSecretName {
		t.Errorf("secret name = %q, want %q", gwEnv.ValueFrom.SecretKeyRef.Name, existingSecretName)
	}
	if gwEnv.ValueFrom.SecretKeyRef.Key != GatewayTokenSecretKey {
		t.Errorf("secret key = %q, want %q", gwEnv.ValueFrom.SecretKeyRef.Key, GatewayTokenSecretKey)
	}
}

func TestBuildStatefulSet_ExistingSecret_UserOverride(t *testing.T) {
	instance := newTestInstance("existing-secret-override")
	instance.Spec.Gateway.ExistingSecret = "my-custom-secret"
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "OPENCLAW_GATEWAY_TOKEN", Value: "user-provided-token"},
	}

	sts := BuildStatefulSet(instance, "my-custom-secret")

	main := sts.Spec.Template.Spec.Containers[0]
	count := 0
	for _, env := range main.Env {
		if env.Name == "OPENCLAW_GATEWAY_TOKEN" {
			count++
			if env.Value != "user-provided-token" {
				t.Errorf("OPENCLAW_GATEWAY_TOKEN value = %q, want %q", env.Value, "user-provided-token")
			}
			if env.ValueFrom != nil {
				t.Error("user's OPENCLAW_GATEWAY_TOKEN should not use SecretKeyRef")
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 OPENCLAW_GATEWAY_TOKEN env var, got %d", count)
	}
}

func TestBuildStatefulSet_NoGatewayTokenSecretName(t *testing.T) {
	instance := newTestInstance("no-gw")

	sts := BuildStatefulSet(instance)

	main := sts.Spec.Template.Spec.Containers[0]
	for _, env := range main.Env {
		if env.Name == "OPENCLAW_GATEWAY_TOKEN" {
			t.Error("OPENCLAW_GATEWAY_TOKEN should not be present when no secret name is provided")
		}
	}
}

// ---------------------------------------------------------------------------
// Feature: fsGroupChangePolicy
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_FSGroupChangePolicy_Default(t *testing.T) {
	instance := newTestInstance("fsgcp-default")
	sts := BuildStatefulSet(instance)
	psc := sts.Spec.Template.Spec.SecurityContext
	if psc.FSGroupChangePolicy != nil {
		t.Errorf("FSGroupChangePolicy should be nil by default, got %v", *psc.FSGroupChangePolicy)
	}
}

func TestBuildStatefulSet_FSGroupChangePolicy_OnRootMismatch(t *testing.T) {
	instance := newTestInstance("fsgcp-onroot")
	policy := corev1.FSGroupChangeOnRootMismatch
	instance.Spec.Security.PodSecurityContext = &openclawv1alpha1.PodSecurityContextSpec{
		FSGroupChangePolicy: &policy,
	}

	sts := BuildStatefulSet(instance)
	psc := sts.Spec.Template.Spec.SecurityContext
	if psc.FSGroupChangePolicy == nil {
		t.Fatal("FSGroupChangePolicy should not be nil")
	}
	if *psc.FSGroupChangePolicy != corev1.FSGroupChangeOnRootMismatch {
		t.Errorf("FSGroupChangePolicy = %v, want OnRootMismatch", *psc.FSGroupChangePolicy)
	}
}

func TestBuildStatefulSet_FSGroupChangePolicy_Always(t *testing.T) {
	instance := newTestInstance("fsgcp-always")
	policy := corev1.FSGroupChangeAlways
	instance.Spec.Security.PodSecurityContext = &openclawv1alpha1.PodSecurityContextSpec{
		FSGroupChangePolicy: &policy,
	}

	sts := BuildStatefulSet(instance)
	psc := sts.Spec.Template.Spec.SecurityContext
	if psc.FSGroupChangePolicy == nil {
		t.Fatal("FSGroupChangePolicy should not be nil")
	}
	if *psc.FSGroupChangePolicy != corev1.FSGroupChangeAlways {
		t.Errorf("FSGroupChangePolicy = %v, want Always", *psc.FSGroupChangePolicy)
	}
}

// ---------------------------------------------------------------------------
// Feature: SA annotations
// ---------------------------------------------------------------------------

func TestBuildServiceAccount_NoAnnotations(t *testing.T) {
	instance := newTestInstance("sa-no-ann")
	sa := BuildServiceAccount(instance)
	if len(sa.Annotations) > 0 {
		t.Errorf("expected nil/empty annotations, got %v", sa.Annotations)
	}
}

func TestBuildServiceAccount_WithAnnotations(t *testing.T) {
	instance := newTestInstance("sa-ann")
	instance.Spec.Security.RBAC.ServiceAccountAnnotations = map[string]string{
		"eks.amazonaws.com/role-arn":     "arn:aws:iam::123456789:role/my-role",
		"iam.gke.io/gcp-service-account": "my-sa@my-project.iam.gserviceaccount.com",
	}

	sa := BuildServiceAccount(instance)
	if len(sa.Annotations) != 2 {
		t.Fatalf("expected 2 annotations, got %d", len(sa.Annotations))
	}
	if sa.Annotations["eks.amazonaws.com/role-arn"] != "arn:aws:iam::123456789:role/my-role" {
		t.Error("IRSA annotation not found")
	}
	if sa.Annotations["iam.gke.io/gcp-service-account"] != "my-sa@my-project.iam.gserviceaccount.com" {
		t.Error("GKE WI annotation not found")
	}
}

func TestBuildServiceAccount_AnnotationsDoNotAffectLabels(t *testing.T) {
	instance := newTestInstance("sa-ann-labels")
	instance.Spec.Security.RBAC.ServiceAccountAnnotations = map[string]string{
		"test": "value",
	}

	sa := BuildServiceAccount(instance)
	if sa.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("labels should still be set when annotations are present")
	}
}

// ---------------------------------------------------------------------------
// Feature: Extra volumes/mounts
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_ExtraVolumes_None(t *testing.T) {
	instance := newTestInstance("no-extras")
	sts := BuildStatefulSet(instance)
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == "my-extra" {
			t.Error("should not have extra volumes when none configured")
		}
	}
}

func TestBuildStatefulSet_ExtraVolumes(t *testing.T) {
	instance := newTestInstance("extras")
	instance.Spec.ExtraVolumes = []corev1.Volume{
		{
			Name: "ssh-keys",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: "ssh-secret"},
			},
		},
	}
	instance.Spec.ExtraVolumeMounts = []corev1.VolumeMount{
		{Name: "ssh-keys", MountPath: "/home/openclaw/.ssh", ReadOnly: true},
	}

	sts := BuildStatefulSet(instance)

	// Check volume exists
	vol := findVolume(sts.Spec.Template.Spec.Volumes, "ssh-keys")
	if vol == nil {
		t.Fatal("extra volume 'ssh-keys' not found")
	}
	if vol.Secret == nil || vol.Secret.SecretName != "ssh-secret" {
		t.Error("extra volume should reference ssh-secret")
	}

	// Check mount exists on main container
	main := sts.Spec.Template.Spec.Containers[0]
	assertVolumeMount(t, main.VolumeMounts, "ssh-keys", "/home/openclaw/.ssh")
}

func TestBuildStatefulSet_ExtraVolumes_DontInterfereWithExisting(t *testing.T) {
	instance := newTestInstance("extras-coexist")
	instance.Spec.ExtraVolumes = []corev1.Volume{
		{
			Name: "custom-vol",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	sts := BuildStatefulSet(instance)
	volumes := sts.Spec.Template.Spec.Volumes

	// Existing volumes should still be present
	if findVolume(volumes, "data") == nil {
		t.Error("data volume should still exist")
	}
	if findVolume(volumes, "tmp") == nil {
		t.Error("tmp volume should still exist")
	}
	if findVolume(volumes, "custom-vol") == nil {
		t.Error("extra volume should be appended")
	}
}

// ---------------------------------------------------------------------------
// Feature: CA bundle injection
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_CABundle_Nil(t *testing.T) {
	instance := newTestInstance("no-ca")
	sts := BuildStatefulSet(instance)

	if findVolume(sts.Spec.Template.Spec.Volumes, "ca-bundle") != nil {
		t.Error("ca-bundle volume should not exist when CABundle is nil")
	}
}

func TestBuildStatefulSet_CABundle_ConfigMap(t *testing.T) {
	instance := newTestInstance("ca-cm")
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		ConfigMapName: "my-ca-bundle",
		Key:           "custom-ca.crt",
	}

	sts := BuildStatefulSet(instance)

	// Volume
	vol := findVolume(sts.Spec.Template.Spec.Volumes, "ca-bundle")
	if vol == nil {
		t.Fatal("ca-bundle volume not found")
	}
	if vol.ConfigMap == nil {
		t.Fatal("ca-bundle volume should use ConfigMap")
	}
	if vol.ConfigMap.Name != "my-ca-bundle" {
		t.Errorf("ca-bundle configmap = %q, want %q", vol.ConfigMap.Name, "my-ca-bundle")
	}

	// Main container mount + env
	main := sts.Spec.Template.Spec.Containers[0]
	assertVolumeMount(t, main.VolumeMounts, "ca-bundle", "/etc/ssl/certs/custom-ca-bundle.crt")

	foundEnv := false
	for _, env := range main.Env {
		if env.Name == "NODE_EXTRA_CA_CERTS" {
			foundEnv = true
			if env.Value != "/etc/ssl/certs/custom-ca-bundle.crt" {
				t.Errorf("NODE_EXTRA_CA_CERTS = %q, want /etc/ssl/certs/custom-ca-bundle.crt", env.Value)
			}
		}
	}
	if !foundEnv {
		t.Error("NODE_EXTRA_CA_CERTS env var not found on main container")
	}
}

func TestBuildStatefulSet_CABundle_Secret(t *testing.T) {
	instance := newTestInstance("ca-secret")
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		SecretName: "ca-secret",
	}

	sts := BuildStatefulSet(instance)

	vol := findVolume(sts.Spec.Template.Spec.Volumes, "ca-bundle")
	if vol == nil {
		t.Fatal("ca-bundle volume not found")
	}
	if vol.Secret == nil {
		t.Fatal("ca-bundle volume should use Secret")
	}
	if vol.Secret.SecretName != "ca-secret" {
		t.Errorf("ca-bundle secret = %q, want %q", vol.Secret.SecretName, "ca-secret")
	}
}

func TestBuildStatefulSet_CABundle_DefaultKey(t *testing.T) {
	instance := newTestInstance("ca-default-key")
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		ConfigMapName: "my-ca",
		// Key not set — should default to "ca-bundle.crt"
	}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	// Find the ca-bundle mount and check subPath
	for _, m := range main.VolumeMounts {
		if m.Name == "ca-bundle" {
			if m.SubPath != "ca-bundle.crt" {
				t.Errorf("subPath = %q, want %q", m.SubPath, "ca-bundle.crt")
			}
			return
		}
	}
	t.Error("ca-bundle volume mount not found")
}

func TestBuildStatefulSet_CABundle_WithChromium(t *testing.T) {
	instance := newTestInstance("ca-chromium")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		ConfigMapName: "my-ca",
		Key:           "ca.crt",
	}

	sts := BuildStatefulSet(instance)

	// Find chromium container
	var chromium *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "chromium" {
			chromium = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if chromium == nil {
		t.Fatal("chromium container not found")
	}

	// Check mount
	assertVolumeMount(t, chromium.VolumeMounts, "ca-bundle", "/etc/ssl/certs/custom-ca-bundle.crt")

	// Check env
	foundEnv := false
	for _, env := range chromium.Env {
		if env.Name == "NODE_EXTRA_CA_CERTS" {
			foundEnv = true
		}
	}
	if !foundEnv {
		t.Error("NODE_EXTRA_CA_CERTS env var not found on chromium container")
	}
}

func TestBuildStatefulSet_CABundle_InitSkills(t *testing.T) {
	instance := newTestInstance("ca-skills")
	instance.Spec.Skills = []string{"some-skill"}
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		ConfigMapName: "my-ca",
		Key:           "ca.crt",
	}

	sts := BuildStatefulSet(instance)

	// Find init-skills container
	var initSkills *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-skills" {
			initSkills = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if initSkills == nil {
		t.Fatal("init-skills container not found")
	}

	// Check mount
	assertVolumeMount(t, initSkills.VolumeMounts, "ca-bundle", "/etc/ssl/certs/custom-ca-bundle.crt")

	// Check env
	foundEnv := false
	for _, env := range initSkills.Env {
		if env.Name == "NODE_EXTRA_CA_CERTS" {
			foundEnv = true
		}
	}
	if !foundEnv {
		t.Error("NODE_EXTRA_CA_CERTS env var not found on init-skills container")
	}
}

// ---------------------------------------------------------------------------
// Feature: Custom init containers
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_NoCustomInitContainers(t *testing.T) {
	instance := newTestInstance("no-custom-init")
	sts := BuildStatefulSet(instance)
	for _, c := range sts.Spec.Template.Spec.InitContainers {
		if c.Name == "my-init" {
			t.Error("should not have custom init containers when none configured")
		}
	}
}

func TestBuildStatefulSet_CustomInitContainers(t *testing.T) {
	instance := newTestInstance("custom-init")
	instance.Spec.InitContainers = []corev1.Container{
		{
			Name:    "my-init",
			Image:   "busybox:1.37",
			Command: []string{"echo", "hello"},
		},
	}

	sts := BuildStatefulSet(instance)

	// Custom init container should be last
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) == 0 {
		t.Fatal("expected at least one init container")
	}
	last := initContainers[len(initContainers)-1]
	if last.Name != "my-init" {
		t.Errorf("last init container = %q, want %q", last.Name, "my-init")
	}
	if last.Image != "busybox:1.37" {
		t.Errorf("custom init container image = %q, want %q", last.Image, "busybox:1.37")
	}
}

func TestBuildStatefulSet_CustomInitContainers_AfterOperatorManaged(t *testing.T) {
	instance := newTestInstance("custom-init-order")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.Skills = []string{"some-skill"}
	instance.Spec.InitContainers = []corev1.Container{
		{Name: "user-init", Image: "busybox:1.37"},
	}

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	if len(initContainers) != 3 {
		t.Fatalf("expected 3 init containers, got %d", len(initContainers))
	}
	if initContainers[0].Name != "init-config" {
		t.Errorf("initContainers[0] = %q, want init-config", initContainers[0].Name)
	}
	if initContainers[1].Name != "init-skills" {
		t.Errorf("initContainers[1] = %q, want init-skills", initContainers[1].Name)
	}
	if initContainers[2].Name != "user-init" {
		t.Errorf("initContainers[2] = %q, want user-init", initContainers[2].Name)
	}
}

func TestConfigHash_ChangesWithInitContainers(t *testing.T) {
	instance := newTestInstance("hash-ic")

	dep1 := BuildStatefulSet(instance)
	hash1 := dep1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.InitContainers = []corev1.Container{
		{Name: "my-init", Image: "busybox:1.37"},
	}

	dep2 := BuildStatefulSet(instance)
	hash2 := dep2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when init containers are added")
	}
}

// ---------------------------------------------------------------------------
// Feature: JSON5 config support
// ---------------------------------------------------------------------------

func TestBuildInitScript_JSON5_Overwrite(t *testing.T) {
	instance := newTestInstance("json5-overwrite")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "my-config",
		Key:  "config.json5",
	}
	instance.Spec.Config.Format = ConfigFormatJSON5

	script := BuildInitScript(instance)
	if !strings.Contains(script, "npx -y json5") {
		t.Errorf("JSON5 overwrite should use npx json5, got: %q", script)
	}
	if !strings.Contains(script, "/tmp/converted.json") {
		t.Errorf("JSON5 overwrite should write to /tmp/converted.json, got: %q", script)
	}
}

func TestBuildStatefulSet_JSON5_UsesOpenClawImage(t *testing.T) {
	instance := newTestInstance("json5-image")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "my-config",
		Key:  "config.json5",
	}
	instance.Spec.Config.Format = ConfigFormatJSON5

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers
	if len(initContainers) == 0 {
		t.Fatal("expected init container for JSON5 mode")
	}

	initC := initContainers[0]
	expectedImage := GetImage(instance)
	if initC.Image != expectedImage {
		t.Errorf("JSON5 init container image = %q, want %q", initC.Image, expectedImage)
	}
}

func TestBuildStatefulSet_JSON5_InitTmpVolume(t *testing.T) {
	instance := newTestInstance("json5-vol")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "my-config",
	}
	instance.Spec.Config.Format = ConfigFormatJSON5

	sts := BuildStatefulSet(instance)

	// Should have init-tmp volume
	initTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "init-tmp")
	if initTmpVol == nil {
		t.Fatal("init-tmp volume not found in JSON5 mode")
	}

	// Should have init-tmp mount on init container
	initC := sts.Spec.Template.Spec.InitContainers[0]
	assertVolumeMount(t, initC.VolumeMounts, "init-tmp", "/tmp")
}

func TestBuildStatefulSet_JSON5_WritableRootFS(t *testing.T) {
	instance := newTestInstance("json5-writable")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "my-config",
	}
	instance.Spec.Config.Format = ConfigFormatJSON5

	sts := BuildStatefulSet(instance)
	initC := sts.Spec.Template.Spec.InitContainers[0]

	if initC.SecurityContext.ReadOnlyRootFilesystem == nil || *initC.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("JSON5 init container should have writable root filesystem for npx")
	}
}

func TestBuildInitScript_JSON_Overwrite_NoBusyboxRegression(t *testing.T) {
	instance := newTestInstance("json-overwrite")
	instance.Spec.Config.ConfigMapRef = &openclawv1alpha1.ConfigMapKeySelector{
		Name: "my-config",
	}
	instance.Spec.Config.Format = "json"

	script := BuildInitScript(instance)
	if strings.Contains(script, "npx") {
		t.Errorf("JSON overwrite should not use npx, got: %q", script)
	}
	if !strings.Contains(script, "cp /config/") {
		t.Errorf("JSON overwrite should use cp, got: %q", script)
	}
}

// ---------------------------------------------------------------------------
// Feature: Runtime dependency init containers (pnpm, Python/uv)
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_RuntimeDeps_Pnpm(t *testing.T) {
	instance := newTestInstance("pnpm")
	instance.Spec.RuntimeDeps.Pnpm = true

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	// Should have init-pnpm container
	var pnpmContainer *corev1.Container
	for i := range initContainers {
		if initContainers[i].Name == "init-pnpm" {
			pnpmContainer = &initContainers[i]
			break
		}
	}
	if pnpmContainer == nil {
		t.Fatal("init-pnpm container not found")
	}

	// Should use the OpenClaw image (has Node.js + corepack)
	if pnpmContainer.Image != GetImage(instance) {
		t.Errorf("init-pnpm image = %q, want %q", pnpmContainer.Image, GetImage(instance))
	}

	// Should mount data volume
	assertVolumeMount(t, pnpmContainer.VolumeMounts, "data", "/home/openclaw/.openclaw")
	// Should mount pnpm-tmp volume
	assertVolumeMount(t, pnpmContainer.VolumeMounts, "pnpm-tmp", "/tmp")

	// Script should check for existing install (idempotent)
	script := pnpmContainer.Command[2]
	if !strings.Contains(script, "already installed") {
		t.Error("pnpm init script should check for existing install")
	}
	if !strings.Contains(script, "corepack enable pnpm") {
		t.Error("pnpm init script should use corepack")
	}

	// Security context
	if pnpmContainer.SecurityContext.ReadOnlyRootFilesystem == nil || *pnpmContainer.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("init-pnpm should have writable root filesystem")
	}
	if pnpmContainer.SecurityContext.RunAsNonRoot == nil || !*pnpmContainer.SecurityContext.RunAsNonRoot {
		t.Error("init-pnpm should run as non-root")
	}

	// pnpm-tmp volume should exist
	pnpmTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "pnpm-tmp")
	if pnpmTmpVol == nil {
		t.Fatal("pnpm-tmp volume not found")
	}
	if pnpmTmpVol.EmptyDir == nil {
		t.Error("pnpm-tmp should be an emptyDir volume")
	}

	// PATH should be extended in main container
	mainContainer := sts.Spec.Template.Spec.Containers[0]
	var pathEnv *corev1.EnvVar
	for i := range mainContainer.Env {
		if mainContainer.Env[i].Name == "PATH" {
			pathEnv = &mainContainer.Env[i]
			break
		}
	}
	if pathEnv == nil {
		t.Fatal("PATH env var not found in main container")
	}
	if !strings.Contains(pathEnv.Value, RuntimeDepsLocalBin) {
		t.Errorf("PATH should contain %q, got %q", RuntimeDepsLocalBin, pathEnv.Value)
	}
}

func TestBuildStatefulSet_RuntimeDeps_Python(t *testing.T) {
	instance := newTestInstance("python")
	instance.Spec.RuntimeDeps.Python = true

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	// Should have init-python container
	var pythonContainer *corev1.Container
	for i := range initContainers {
		if initContainers[i].Name == "init-python" {
			pythonContainer = &initContainers[i]
			break
		}
	}
	if pythonContainer == nil {
		t.Fatal("init-python container not found")
	}

	// Should use the uv image
	if pythonContainer.Image != UvImage {
		t.Errorf("init-python image = %q, want %q", pythonContainer.Image, UvImage)
	}

	// Should mount data volume
	assertVolumeMount(t, pythonContainer.VolumeMounts, "data", "/home/openclaw/.openclaw")
	// Should mount python-tmp volume
	assertVolumeMount(t, pythonContainer.VolumeMounts, "python-tmp", "/tmp")

	// Script should check for existing install (idempotent)
	script := pythonContainer.Command[2]
	if !strings.Contains(script, "already installed") {
		t.Error("python init script should check for existing install")
	}
	if !strings.Contains(script, "uv python install 3.12") {
		t.Error("python init script should install Python 3.12")
	}
	if !strings.Contains(script, "cp /usr/local/bin/uv") {
		t.Error("python init script should copy uv binary")
	}

	// Security context
	if pythonContainer.SecurityContext.ReadOnlyRootFilesystem == nil || *pythonContainer.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("init-python should have writable root filesystem")
	}
	if pythonContainer.SecurityContext.RunAsNonRoot == nil || !*pythonContainer.SecurityContext.RunAsNonRoot {
		t.Error("init-python should run as non-root")
	}

	// python-tmp volume should exist
	pythonTmpVol := findVolume(sts.Spec.Template.Spec.Volumes, "python-tmp")
	if pythonTmpVol == nil {
		t.Fatal("python-tmp volume not found")
	}

	// PATH should be extended in main container
	mainContainer := sts.Spec.Template.Spec.Containers[0]
	var pathEnv *corev1.EnvVar
	for i := range mainContainer.Env {
		if mainContainer.Env[i].Name == "PATH" {
			pathEnv = &mainContainer.Env[i]
			break
		}
	}
	if pathEnv == nil {
		t.Fatal("PATH env var not found in main container")
	}
}

func TestBuildStatefulSet_RuntimeDeps_Both(t *testing.T) {
	instance := newTestInstance("both-deps")
	instance.Spec.RuntimeDeps.Pnpm = true
	instance.Spec.RuntimeDeps.Python = true

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	var hasPnpm, hasPython bool
	var pnpmIdx, pythonIdx int
	for i, c := range initContainers {
		if c.Name == "init-pnpm" {
			hasPnpm = true
			pnpmIdx = i
		}
		if c.Name == "init-python" {
			hasPython = true
			pythonIdx = i
		}
	}
	if !hasPnpm {
		t.Error("init-pnpm not found")
	}
	if !hasPython {
		t.Error("init-python not found")
	}
	if hasPnpm && hasPython && pnpmIdx >= pythonIdx {
		t.Error("init-pnpm should come before init-python")
	}

	// Both tmp volumes should exist
	if findVolume(sts.Spec.Template.Spec.Volumes, "pnpm-tmp") == nil {
		t.Error("pnpm-tmp volume not found")
	}
	if findVolume(sts.Spec.Template.Spec.Volumes, "python-tmp") == nil {
		t.Error("python-tmp volume not found")
	}
}

func TestBuildStatefulSet_RuntimeDeps_None(t *testing.T) {
	instance := newTestInstance("no-deps")
	// RuntimeDeps defaults to zero value (both false)

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	for _, c := range initContainers {
		if c.Name == "init-pnpm" || c.Name == "init-python" {
			t.Errorf("unexpected runtime dep init container: %s", c.Name)
		}
	}

	// No runtime dep tmp volumes
	if findVolume(sts.Spec.Template.Spec.Volumes, "pnpm-tmp") != nil {
		t.Error("pnpm-tmp volume should not exist")
	}
	if findVolume(sts.Spec.Template.Spec.Volumes, "python-tmp") != nil {
		t.Error("python-tmp volume should not exist")
	}

	// No PATH override
	mainContainer := sts.Spec.Template.Spec.Containers[0]
	for _, e := range mainContainer.Env {
		if e.Name == "PATH" {
			t.Error("PATH env var should not be set when no runtime deps")
		}
	}
}

func TestBuildStatefulSet_RuntimeDeps_InitContainerOrder(t *testing.T) {
	instance := newTestInstance("order")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{}`)},
	}
	instance.Spec.RuntimeDeps.Pnpm = true
	instance.Spec.RuntimeDeps.Python = true
	instance.Spec.Skills = []string{"some-skill"}
	instance.Spec.InitContainers = []corev1.Container{
		{Name: "user-init", Image: "busybox:1.37"},
	}

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	expected := []string{"init-config", "init-pnpm", "init-python", "init-skills", "user-init"}
	if len(initContainers) != len(expected) {
		t.Fatalf("expected %d init containers, got %d: %v", len(expected), len(initContainers),
			func() []string {
				names := make([]string, len(initContainers))
				for i, c := range initContainers {
					names[i] = c.Name
				}
				return names
			}())
	}
	for i, name := range expected {
		if initContainers[i].Name != name {
			t.Errorf("initContainers[%d] = %q, want %q", i, initContainers[i].Name, name)
		}
	}
}

func TestBuildStatefulSet_RuntimeDeps_Pnpm_CABundle(t *testing.T) {
	instance := newTestInstance("pnpm-ca")
	instance.Spec.RuntimeDeps.Pnpm = true
	instance.Spec.Security.CABundle = &openclawv1alpha1.CABundleSpec{
		ConfigMapName: "my-ca",
		Key:           "ca.crt",
	}

	sts := BuildStatefulSet(instance)
	var pnpmContainer *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-pnpm" {
			pnpmContainer = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if pnpmContainer == nil {
		t.Fatal("init-pnpm container not found")
	}

	// Should have CA bundle mount
	assertVolumeMount(t, pnpmContainer.VolumeMounts, "ca-bundle", "/etc/ssl/certs/custom-ca-bundle.crt")

	// Should have NODE_EXTRA_CA_CERTS env
	var hasCAEnv bool
	for _, e := range pnpmContainer.Env {
		if e.Name == "NODE_EXTRA_CA_CERTS" {
			hasCAEnv = true
			break
		}
	}
	if !hasCAEnv {
		t.Error("init-pnpm should have NODE_EXTRA_CA_CERTS when CA bundle is configured")
	}
}

func TestConfigHash_ChangesWithRuntimeDeps(t *testing.T) {
	instance := newTestInstance("hash-rd")

	sts1 := BuildStatefulSet(instance)
	hash1 := sts1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.RuntimeDeps.Pnpm = true

	sts2 := BuildStatefulSet(instance)
	hash2 := sts2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when runtime deps are enabled")
	}
}

// ---------------------------------------------------------------------------
// Tailscale integration tests
// ---------------------------------------------------------------------------

func TestBuildConfigMap_WithTailscale(t *testing.T) {
	instance := newTestInstance("ts-serve")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Mode = "serve"

	cm := BuildConfigMap(instance, "")
	content, ok := cm.Data["openclaw.json"]
	if !ok {
		t.Fatal("ConfigMap should have openclaw.json key")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw, ok := parsed["gateway"].(map[string]interface{})
	if !ok {
		t.Fatal("config should have gateway key")
	}

	ts, ok := gw["tailscale"].(map[string]interface{})
	if !ok {
		t.Fatal("gateway should have tailscale key")
	}

	if ts["mode"] != "serve" {
		t.Errorf("expected tailscale.mode=serve, got %v", ts["mode"])
	}
	if ts["resetOnExit"] != true {
		t.Errorf("expected tailscale.resetOnExit=true, got %v", ts["resetOnExit"])
	}
}

func TestBuildConfigMap_TailscaleFunnel(t *testing.T) {
	instance := newTestInstance("ts-funnel")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Mode = "funnel"

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	ts := gw["tailscale"].(map[string]interface{})

	if ts["mode"] != "funnel" {
		t.Errorf("expected tailscale.mode=funnel, got %v", ts["mode"])
	}
}

func TestBuildConfigMap_TailscaleAuthSSO(t *testing.T) {
	instance := newTestInstance("ts-sso")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.AuthSSO = true

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	auth, ok := gw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("gateway should have auth key when AuthSSO is enabled")
	}

	if auth["allowTailscale"] != true {
		t.Errorf("expected auth.allowTailscale=true, got %v", auth["allowTailscale"])
	}
}

func TestBuildConfigMap_TailscaleUserOverride(t *testing.T) {
	instance := newTestInstance("ts-override")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Mode = "serve"
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"gateway":{"tailscale":{"mode":"funnel","resetOnExit":false}}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	ts := gw["tailscale"].(map[string]interface{})

	if ts["mode"] != "funnel" {
		t.Errorf("user-set mode should be preserved, expected funnel, got %v", ts["mode"])
	}
	if ts["resetOnExit"] != false {
		t.Errorf("user-set resetOnExit should be preserved, expected false, got %v", ts["resetOnExit"])
	}
}

func TestBuildStatefulSet_TailscaleAuthKey(t *testing.T) {
	instance := newTestInstance("ts-authkey")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.AuthKeySecretRef = &corev1.LocalObjectReference{Name: "ts-auth"}

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	var tsAuthKey *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "TS_AUTHKEY" {
			tsAuthKey = &main.Env[i]
			break
		}
	}
	if tsAuthKey == nil {
		t.Fatal("TS_AUTHKEY env var should be present")
	}
	if tsAuthKey.ValueFrom == nil || tsAuthKey.ValueFrom.SecretKeyRef == nil {
		t.Fatal("TS_AUTHKEY should use SecretKeyRef")
	}
	if tsAuthKey.ValueFrom.SecretKeyRef.Name != "ts-auth" {
		t.Errorf("expected secret name ts-auth, got %s", tsAuthKey.ValueFrom.SecretKeyRef.Name)
	}
	if tsAuthKey.ValueFrom.SecretKeyRef.Key != "authkey" {
		t.Errorf("expected secret key authkey, got %s", tsAuthKey.ValueFrom.SecretKeyRef.Key)
	}
}

func TestBuildStatefulSet_TailscaleHostname(t *testing.T) {
	instance := newTestInstance("ts-hostname")
	instance.Spec.Tailscale.Enabled = true

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	var tsHostname *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "TS_HOSTNAME" {
			tsHostname = &main.Env[i]
			break
		}
	}
	if tsHostname == nil {
		t.Fatal("TS_HOSTNAME env var should be present")
	}
	if tsHostname.Value != "ts-hostname" {
		t.Errorf("expected TS_HOSTNAME=ts-hostname (instance name), got %s", tsHostname.Value)
	}
}

func TestBuildStatefulSet_TailscaleCustomHostname(t *testing.T) {
	instance := newTestInstance("ts-custom")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Hostname = "my-custom-host"

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	var tsHostname *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "TS_HOSTNAME" {
			tsHostname = &main.Env[i]
			break
		}
	}
	if tsHostname == nil {
		t.Fatal("TS_HOSTNAME env var should be present")
	}
	if tsHostname.Value != "my-custom-host" {
		t.Errorf("expected TS_HOSTNAME=my-custom-host, got %s", tsHostname.Value)
	}
}

func TestBuildStatefulSet_TailscaleCustomSecretKey(t *testing.T) {
	instance := newTestInstance("ts-customkey")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.AuthKeySecretRef = &corev1.LocalObjectReference{Name: "ts-secret"}
	instance.Spec.Tailscale.AuthKeySecretKey = "my-key"

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	var tsAuthKey *corev1.EnvVar
	for i := range main.Env {
		if main.Env[i].Name == "TS_AUTHKEY" {
			tsAuthKey = &main.Env[i]
			break
		}
	}
	if tsAuthKey == nil {
		t.Fatal("TS_AUTHKEY env var should be present")
	}
	if tsAuthKey.ValueFrom.SecretKeyRef.Key != "my-key" {
		t.Errorf("expected custom secret key my-key, got %s", tsAuthKey.ValueFrom.SecretKeyRef.Key)
	}
}

func TestBuildStatefulSet_TailscaleDisabled(t *testing.T) {
	instance := newTestInstance("ts-disabled")

	sts := BuildStatefulSet(instance)
	main := sts.Spec.Template.Spec.Containers[0]

	for _, env := range main.Env {
		if env.Name == "TS_AUTHKEY" || env.Name == "TS_HOSTNAME" {
			t.Errorf("unexpected Tailscale env var %s when Tailscale is disabled", env.Name)
		}
	}
}

func TestBuildNetworkPolicy_TailscaleEgress(t *testing.T) {
	instance := newTestInstance("ts-np")
	instance.Spec.Tailscale.Enabled = true

	np := BuildNetworkPolicy(instance)

	// Default egress: DNS (0), HTTPS (1), Tailscale STUN+WireGuard (2)
	if len(np.Spec.Egress) < 3 {
		t.Fatalf("expected at least 3 egress rules, got %d", len(np.Spec.Egress))
	}

	tsRule := np.Spec.Egress[2]
	if len(tsRule.Ports) != 2 {
		t.Fatalf("expected 2 Tailscale egress ports, got %d", len(tsRule.Ports))
	}

	foundSTUN := false
	foundWG := false
	for _, p := range tsRule.Ports {
		if p.Protocol != nil && *p.Protocol == corev1.ProtocolUDP && p.Port != nil {
			switch p.Port.IntValue() {
			case 3478:
				foundSTUN = true
			case 41641:
				foundWG = true
			}
		}
	}
	if !foundSTUN {
		t.Error("expected STUN egress rule (UDP 3478)")
	}
	if !foundWG {
		t.Error("expected WireGuard egress rule (UDP 41641)")
	}
}

func TestBuildNetworkPolicy_TailscaleDisabled(t *testing.T) {
	instance := newTestInstance("ts-np-off")

	np := BuildNetworkPolicy(instance)

	// Default egress: DNS (0), HTTPS (1) - no Tailscale rules
	if len(np.Spec.Egress) != 2 {
		t.Errorf("expected 2 egress rules when Tailscale is disabled, got %d", len(np.Spec.Egress))
	}

	for _, rule := range np.Spec.Egress {
		for _, p := range rule.Ports {
			if p.Protocol != nil && *p.Protocol == corev1.ProtocolUDP && p.Port != nil {
				if p.Port.IntValue() == 3478 || p.Port.IntValue() == 41641 {
					t.Errorf("unexpected Tailscale egress port %d when disabled", p.Port.IntValue())
				}
			}
		}
	}
}

func TestBuildStatefulSet_Idempotent_WithTailscale(t *testing.T) {
	instance := newTestInstance("idempotent-ts")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"key":"val"}`)},
	}
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Mode = "serve"
	instance.Spec.Tailscale.AuthKeySecretRef = &corev1.LocalObjectReference{Name: "ts-auth"}
	instance.Spec.Tailscale.Hostname = "my-ts-host"

	dep1 := BuildStatefulSet(instance)
	dep2 := BuildStatefulSet(instance)

	b1, _ := json.Marshal(dep1.Spec)
	b2, _ := json.Marshal(dep2.Spec)

	if !bytes.Equal(b1, b2) {
		t.Error("BuildStatefulSet with Tailscale is not idempotent")
	}
}

func TestConfigHash_ChangesWithTailscale(t *testing.T) {
	instance := newTestInstance("hash-ts")

	sts1 := BuildStatefulSet(instance)
	hash1 := sts1.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.Mode = "serve"

	sts2 := BuildStatefulSet(instance)
	hash2 := sts2.Spec.Template.Annotations["openclaw.rocks/config-hash"]

	if hash1 == hash2 {
		t.Error("config hash should change when Tailscale is enabled")
	}
}

func TestBuildConfigMap_TailscaleDefaultMode(t *testing.T) {
	instance := newTestInstance("ts-default-mode")
	instance.Spec.Tailscale.Enabled = true
	// Mode is empty - should default to "serve"

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	ts := gw["tailscale"].(map[string]interface{})

	if ts["mode"] != "serve" {
		t.Errorf("expected default mode=serve, got %v", ts["mode"])
	}
}

func TestBuildConfigMap_TailscaleNoAuthSSO(t *testing.T) {
	instance := newTestInstance("ts-no-sso")
	instance.Spec.Tailscale.Enabled = true
	instance.Spec.Tailscale.AuthSSO = false

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	gw := parsed["gateway"].(map[string]interface{})
	if auth, ok := gw["auth"].(map[string]interface{}); ok {
		if _, hasAllowTS := auth["allowTailscale"]; hasAllowTS {
			t.Error("auth.allowTailscale should not be set when AuthSSO is disabled")
		}
	}
}

// ---------------------------------------------------------------------------
// Browser config enrichment tests (Chromium sidecar)
// ---------------------------------------------------------------------------

func TestBuildConfigMap_ChromiumBrowserConfig(t *testing.T) {
	instance := newTestInstance("cr-browser")
	instance.Spec.Chromium.Enabled = true

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	browser, ok := parsed["browser"].(map[string]interface{})
	if !ok {
		t.Fatal("expected browser key in config when chromium is enabled")
	}

	if browser["defaultProfile"] != "default" {
		t.Errorf("browser.defaultProfile = %v, want %q", browser["defaultProfile"], "default")
	}

	profiles, ok := browser["profiles"].(map[string]interface{})
	if !ok {
		t.Fatal("expected browser.profiles key")
	}

	// Both "default" and "chrome" profiles must point at the sidecar.
	// LLMs often explicitly pass profile="chrome", so we redirect it.
	for _, name := range []string{"default", "chrome"} {
		p, ok := profiles[name].(map[string]interface{})
		if !ok {
			t.Fatalf("expected browser.profiles.%s key", name)
		}
		if p["cdpUrl"] != "http://localhost:3000" {
			t.Errorf("browser.profiles.%s.cdpUrl = %v, want %q", name, p["cdpUrl"], "http://localhost:3000")
		}
		if p["color"] != "#4285F4" {
			t.Errorf("browser.profiles.%s.color = %v, want %q", name, p["color"], "#4285F4")
		}
	}
}

func TestBuildConfigMap_ChromiumDisabled_NoBrowserConfig(t *testing.T) {
	instance := newTestInstance("cr-disabled")
	// Chromium not enabled (default)

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	if _, ok := parsed["browser"]; ok {
		t.Error("browser config should not be present when chromium is disabled")
	}
}

func TestBuildConfigMap_ChromiumUserOverrideDefaultProfile(t *testing.T) {
	instance := newTestInstance("cr-override-profile")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"browser":{"defaultProfile":"chrome"}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	browser := parsed["browser"].(map[string]interface{})
	if browser["defaultProfile"] != "chrome" {
		t.Errorf("user-set defaultProfile should be preserved, got %v", browser["defaultProfile"])
	}
}

func TestBuildConfigMap_ChromiumUserOverrideCdpUrl(t *testing.T) {
	instance := newTestInstance("cr-override-cdp")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"browser":{"profiles":{"default":{"cdpUrl":"ws://custom:1234"}}}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	browser := parsed["browser"].(map[string]interface{})
	profiles := browser["profiles"].(map[string]interface{})
	defaultProfile := profiles["default"].(map[string]interface{})

	if defaultProfile["cdpUrl"] != "ws://custom:1234" {
		t.Errorf("user-set cdpUrl should be preserved, got %v", defaultProfile["cdpUrl"])
	}
}

func TestBuildConfigMap_ChromiumUserOverrideCdpPort(t *testing.T) {
	instance := newTestInstance("cr-override-port")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{
			Raw: []byte(`{"browser":{"profiles":{"default":{"cdpPort":18800}}}}`),
		},
	}

	cm := BuildConfigMap(instance, "")
	content := cm.Data["openclaw.json"]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("failed to parse config JSON: %v", err)
	}

	browser := parsed["browser"].(map[string]interface{})
	profiles := browser["profiles"].(map[string]interface{})
	defaultProfile := profiles["default"].(map[string]interface{})

	// cdpUrl should NOT be injected when user set cdpPort
	if _, hasCdpURL := defaultProfile["cdpUrl"]; hasCdpURL {
		t.Error("cdpUrl should not be injected when user set cdpPort")
	}
	// cdpPort should be preserved
	if defaultProfile["cdpPort"] != float64(18800) {
		t.Errorf("user-set cdpPort should be preserved, got %v", defaultProfile["cdpPort"])
	}
}

// ---------------------------------------------------------------------------
// Ollama sidecar tests
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_OllamaEnabled(t *testing.T) {
	instance := newTestInstance("ollama-test")
	instance.Spec.Ollama.Enabled = true

	sts := BuildStatefulSet(instance)
	containers := sts.Spec.Template.Spec.Containers

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers (main + ollama), got %d", len(containers))
	}

	var ollama *corev1.Container
	for i := range containers {
		if containers[i].Name == "ollama" {
			ollama = &containers[i]
			break
		}
	}
	if ollama == nil {
		t.Fatal("ollama container not found")
	}

	// Main container should have OLLAMA_HOST env var
	mainContainer := containers[0]
	foundOllamaHost := false
	for _, env := range mainContainer.Env {
		if env.Name == "OLLAMA_HOST" {
			foundOllamaHost = true
			if env.Value != "http://localhost:11434" {
				t.Errorf("OLLAMA_HOST = %q, want %q", env.Value, "http://localhost:11434")
			}
			break
		}
	}
	if !foundOllamaHost {
		t.Error("main container should have OLLAMA_HOST env var when ollama is enabled")
	}

	// Ollama image defaults
	if ollama.Image != "ollama/ollama:latest" {
		t.Errorf("ollama image = %q, want default", ollama.Image)
	}

	// Ollama port
	if len(ollama.Ports) != 1 {
		t.Fatalf("ollama container should have 1 port, got %d", len(ollama.Ports))
	}
	if ollama.Ports[0].ContainerPort != OllamaPort {
		t.Errorf("ollama port = %d, want %d", ollama.Ports[0].ContainerPort, OllamaPort)
	}
	if ollama.Ports[0].Name != "ollama" {
		t.Errorf("ollama port name = %q, want %q", ollama.Ports[0].Name, "ollama")
	}

	// Ollama security context — runs as root
	osc := ollama.SecurityContext
	if osc == nil {
		t.Fatal("ollama security context is nil")
	}
	if osc.ReadOnlyRootFilesystem == nil || *osc.ReadOnlyRootFilesystem {
		t.Error("ollama: readOnlyRootFilesystem should be false")
	}
	if osc.RunAsUser == nil || *osc.RunAsUser != 0 {
		t.Errorf("ollama: runAsUser = %v, want 0 (root)", osc.RunAsUser)
	}
	if osc.RunAsNonRoot == nil || *osc.RunAsNonRoot {
		t.Error("ollama: runAsNonRoot should be false")
	}

	// Ollama resource defaults
	cpuReq := ollama.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "500m" {
		t.Errorf("ollama cpu request = %v, want 500m", cpuReq.String())
	}
	memReq := ollama.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Errorf("ollama memory request = %v, want 1Gi", memReq.String())
	}

	// Ollama volume mount
	assertVolumeMount(t, ollama.VolumeMounts, "ollama-models", "/root/.ollama")

	// Volumes — check ollama-models volume exists with emptyDir
	volumes := sts.Spec.Template.Spec.Volumes
	var ollamaVol *corev1.Volume
	for i := range volumes {
		if volumes[i].Name == "ollama-models" {
			ollamaVol = &volumes[i]
			break
		}
	}
	if ollamaVol == nil {
		t.Fatal("ollama-models volume not found")
	}
	if ollamaVol.EmptyDir == nil {
		t.Fatal("ollama-models volume should be emptyDir by default")
	}
	if ollamaVol.EmptyDir.SizeLimit == nil {
		t.Fatal("ollama-models emptyDir should have a sizeLimit")
	}
	if ollamaVol.EmptyDir.SizeLimit.Cmp(resource.MustParse("20Gi")) != 0 {
		t.Errorf("ollama-models sizeLimit = %v, want 20Gi", ollamaVol.EmptyDir.SizeLimit.String())
	}

	// No init container when no models specified
	for _, ic := range sts.Spec.Template.Spec.InitContainers {
		if ic.Name == "init-ollama" {
			t.Error("init-ollama should not be present when no models are specified")
		}
	}
}

func TestBuildStatefulSet_OllamaEnabled_WithModels(t *testing.T) {
	instance := newTestInstance("ollama-models")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Models = []string{"llama3.2", "nomic-embed-text"}

	sts := BuildStatefulSet(instance)
	initContainers := sts.Spec.Template.Spec.InitContainers

	var initOllama *corev1.Container
	for i := range initContainers {
		if initContainers[i].Name == "init-ollama" {
			initOllama = &initContainers[i]
			break
		}
	}
	if initOllama == nil {
		t.Fatal("init-ollama container not found")
	}

	// Verify command pulls both models
	cmd := initOllama.Command[2] // ["sh", "-c", "..."]
	if !strings.Contains(cmd, "ollama pull 'llama3.2'") {
		t.Errorf("init-ollama command should pull llama3.2, got: %s", cmd)
	}
	if !strings.Contains(cmd, "ollama pull 'nomic-embed-text'") {
		t.Errorf("init-ollama command should pull nomic-embed-text, got: %s", cmd)
	}
	if !strings.Contains(cmd, "ollama serve") {
		t.Errorf("init-ollama command should start ollama serve, got: %s", cmd)
	}
	if !strings.Contains(cmd, "kill %1 2>/dev/null; exit 0") {
		t.Errorf("init-ollama command should kill server, got: %s", cmd)
	}

	// Verify init container mounts ollama-models
	assertVolumeMount(t, initOllama.VolumeMounts, "ollama-models", "/root/.ollama")

	// Verify init container runs as root
	if initOllama.SecurityContext.RunAsUser == nil || *initOllama.SecurityContext.RunAsUser != 0 {
		t.Errorf("init-ollama: runAsUser = %v, want 0", initOllama.SecurityContext.RunAsUser)
	}
}

func TestBuildStatefulSet_OllamaEnabled_NoModels(t *testing.T) {
	instance := newTestInstance("ollama-no-models")
	instance.Spec.Ollama.Enabled = true

	sts := BuildStatefulSet(instance)

	// Sidecar should be present
	found := false
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "ollama" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ollama sidecar should be present")
	}

	// No init container
	for _, ic := range sts.Spec.Template.Spec.InitContainers {
		if ic.Name == "init-ollama" {
			t.Error("init-ollama should not be present when no models are specified")
		}
	}
}

func TestBuildStatefulSet_OllamaEnabled_GPU(t *testing.T) {
	instance := newTestInstance("ollama-gpu")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.GPU = Ptr(int32(2))

	sts := BuildStatefulSet(instance)

	var ollama *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "ollama" {
			ollama = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if ollama == nil {
		t.Fatal("ollama container not found")
	}

	gpuRes := corev1.ResourceName("nvidia.com/gpu")

	// Check requests
	gpuReq, ok := ollama.Resources.Requests[gpuRes]
	if !ok {
		t.Fatal("nvidia.com/gpu request not found")
	}
	if gpuReq.String() != "2" {
		t.Errorf("gpu request = %v, want 2", gpuReq.String())
	}

	// Check limits
	gpuLim, ok := ollama.Resources.Limits[gpuRes]
	if !ok {
		t.Fatal("nvidia.com/gpu limit not found")
	}
	if gpuLim.String() != "2" {
		t.Errorf("gpu limit = %v, want 2", gpuLim.String())
	}
}

func TestBuildStatefulSet_OllamaEnabled_ExistingClaim(t *testing.T) {
	instance := newTestInstance("ollama-pvc")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Storage.ExistingClaim = "my-model-pvc"

	sts := BuildStatefulSet(instance)
	volumes := sts.Spec.Template.Spec.Volumes

	var ollamaVol *corev1.Volume
	for i := range volumes {
		if volumes[i].Name == "ollama-models" {
			ollamaVol = &volumes[i]
			break
		}
	}
	if ollamaVol == nil {
		t.Fatal("ollama-models volume not found")
	}
	if ollamaVol.PersistentVolumeClaim == nil {
		t.Fatal("ollama-models should use PVC when existingClaim is set")
	}
	if ollamaVol.PersistentVolumeClaim.ClaimName != "my-model-pvc" {
		t.Errorf("PVC claim name = %q, want %q", ollamaVol.PersistentVolumeClaim.ClaimName, "my-model-pvc")
	}
}

func TestBuildStatefulSet_OllamaEnabled_CustomImage(t *testing.T) {
	instance := newTestInstance("ollama-custom-img")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Image = openclawv1alpha1.OllamaImageSpec{
		Repository: "my-registry.io/ollama",
		Tag:        "v0.3.0",
	}

	sts := BuildStatefulSet(instance)

	var ollama *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "ollama" {
			ollama = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if ollama == nil {
		t.Fatal("ollama container not found")
	}
	if ollama.Image != "my-registry.io/ollama:v0.3.0" {
		t.Errorf("ollama image = %q, want %q", ollama.Image, "my-registry.io/ollama:v0.3.0")
	}
}

func TestBuildStatefulSet_OllamaEnabled_CustomImageDigest(t *testing.T) {
	instance := newTestInstance("ollama-digest")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Image = openclawv1alpha1.OllamaImageSpec{
		Repository: "ollama/ollama",
		Tag:        "v0.3.0",
		Digest:     "sha256:ollamahash",
	}

	sts := BuildStatefulSet(instance)

	var ollama *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "ollama" {
			ollama = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if ollama == nil {
		t.Fatal("ollama container not found")
	}
	if ollama.Image != "ollama/ollama@sha256:ollamahash" {
		t.Errorf("ollama image = %q, want %q", ollama.Image, "ollama/ollama@sha256:ollamahash")
	}
}

func TestBuildStatefulSet_OllamaEnabled_CustomResources(t *testing.T) {
	instance := newTestInstance("ollama-res")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Resources = openclawv1alpha1.ResourcesSpec{
		Requests: openclawv1alpha1.ResourceList{
			CPU:    "1",
			Memory: "4Gi",
		},
		Limits: openclawv1alpha1.ResourceList{
			CPU:    "4",
			Memory: "16Gi",
		},
	}

	sts := BuildStatefulSet(instance)

	var ollama *corev1.Container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "ollama" {
			ollama = &sts.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if ollama == nil {
		t.Fatal("ollama container not found")
	}

	cpuReq := ollama.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "1" {
		t.Errorf("cpu request = %v, want 1", cpuReq.String())
	}
	memLim := ollama.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("16Gi")) != 0 {
		t.Errorf("memory limit = %v, want 16Gi", memLim.String())
	}
}

func TestBuildStatefulSet_OllamaAndChromiumEnabled(t *testing.T) {
	instance := newTestInstance("both-sidecars")
	instance.Spec.Chromium.Enabled = true
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Models = []string{"llama3.2"}

	sts := BuildStatefulSet(instance)
	containers := sts.Spec.Template.Spec.Containers

	if len(containers) != 3 {
		t.Fatalf("expected 3 containers (main + chromium + ollama), got %d", len(containers))
	}

	names := make(map[string]bool)
	for _, c := range containers {
		names[c.Name] = true
	}
	if !names["openclaw"] {
		t.Error("main container not found")
	}
	if !names["chromium"] {
		t.Error("chromium container not found")
	}
	if !names["ollama"] {
		t.Error("ollama container not found")
	}

	// Both env vars should be present on main container
	mainContainer := containers[0]
	foundChromiumURL := false
	foundOllamaHost := false
	for _, env := range mainContainer.Env {
		if env.Name == "CHROMIUM_URL" {
			foundChromiumURL = true
		}
		if env.Name == "OLLAMA_HOST" {
			foundOllamaHost = true
		}
	}
	if !foundChromiumURL {
		t.Error("main container should have CHROMIUM_URL")
	}
	if !foundOllamaHost {
		t.Error("main container should have OLLAMA_HOST")
	}

	// Both volumes should exist
	volumes := sts.Spec.Template.Spec.Volumes
	volumeNames := make(map[string]bool)
	for _, v := range volumes {
		volumeNames[v.Name] = true
	}
	if !volumeNames["chromium-tmp"] {
		t.Error("chromium-tmp volume not found")
	}
	if !volumeNames["chromium-shm"] {
		t.Error("chromium-shm volume not found")
	}
	if !volumeNames["ollama-models"] {
		t.Error("ollama-models volume not found")
	}

	// Init container for model pull
	foundInitOllama := false
	for _, ic := range sts.Spec.Template.Spec.InitContainers {
		if ic.Name == "init-ollama" {
			foundInitOllama = true
			break
		}
	}
	if !foundInitOllama {
		t.Error("init-ollama container not found when models are specified")
	}
}

func TestBuildStatefulSet_OllamaDisabled(t *testing.T) {
	instance := newTestInstance("no-ollama")

	sts := BuildStatefulSet(instance)

	// No ollama container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "ollama" {
			t.Error("ollama container should not be present when disabled")
		}
	}

	// No ollama-models volume
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == "ollama-models" {
			t.Error("ollama-models volume should not be present when disabled")
		}
	}

	// No OLLAMA_HOST env var
	mainContainer := sts.Spec.Template.Spec.Containers[0]
	for _, env := range mainContainer.Env {
		if env.Name == "OLLAMA_HOST" {
			t.Error("OLLAMA_HOST should not be set when ollama is disabled")
		}
	}
}

func TestBuildStatefulSet_OllamaEnabled_CustomStorageSize(t *testing.T) {
	instance := newTestInstance("ollama-storage")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Storage.SizeLimit = "50Gi"

	sts := BuildStatefulSet(instance)

	var ollamaVol *corev1.Volume
	for i := range sts.Spec.Template.Spec.Volumes {
		if sts.Spec.Template.Spec.Volumes[i].Name == "ollama-models" {
			ollamaVol = &sts.Spec.Template.Spec.Volumes[i]
			break
		}
	}
	if ollamaVol == nil {
		t.Fatal("ollama-models volume not found")
	}
	if ollamaVol.EmptyDir == nil {
		t.Fatal("expected emptyDir")
	}
	if ollamaVol.EmptyDir.SizeLimit.Cmp(resource.MustParse("50Gi")) != 0 {
		t.Errorf("sizeLimit = %v, want 50Gi", ollamaVol.EmptyDir.SizeLimit.String())
	}
}

func TestBuildStatefulSet_OllamaContainerDefaults(t *testing.T) {
	instance := newTestInstance("ollama-defaults")
	instance.Spec.Ollama.Enabled = true

	sts := BuildStatefulSet(instance)
	if len(sts.Spec.Template.Spec.Containers) < 2 {
		t.Fatal("expected ollama sidecar container")
	}

	var ollama corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "ollama" {
			ollama = c
			break
		}
	}

	// Verify Kubernetes default fields are set
	if ollama.TerminationMessagePath != corev1.TerminationMessagePathDefault {
		t.Errorf("TerminationMessagePath = %q, want default", ollama.TerminationMessagePath)
	}
	if ollama.TerminationMessagePolicy != corev1.TerminationMessageReadFile {
		t.Errorf("TerminationMessagePolicy = %v, want ReadFile", ollama.TerminationMessagePolicy)
	}
	if ollama.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("ImagePullPolicy = %v, want IfNotPresent", ollama.ImagePullPolicy)
	}

	// Seccomp profile
	if ollama.SecurityContext.SeccompProfile == nil {
		t.Fatal("seccomp profile should be set")
	}
	if ollama.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Errorf("seccomp type = %v, want RuntimeDefault", ollama.SecurityContext.SeccompProfile.Type)
	}
}

func TestBuildStatefulSet_OllamaEnabled_InitContainerUsesCustomImage(t *testing.T) {
	instance := newTestInstance("ollama-init-img")
	instance.Spec.Ollama.Enabled = true
	instance.Spec.Ollama.Models = []string{"llama3.2"}
	instance.Spec.Ollama.Image = openclawv1alpha1.OllamaImageSpec{
		Repository: "my-registry.io/ollama",
		Tag:        "v0.3.0",
	}

	sts := BuildStatefulSet(instance)

	var initOllama *corev1.Container
	for i := range sts.Spec.Template.Spec.InitContainers {
		if sts.Spec.Template.Spec.InitContainers[i].Name == "init-ollama" {
			initOllama = &sts.Spec.Template.Spec.InitContainers[i]
			break
		}
	}
	if initOllama == nil {
		t.Fatal("init-ollama container not found")
	}
	if initOllama.Image != "my-registry.io/ollama:v0.3.0" {
		t.Errorf("init-ollama image = %q, want %q", initOllama.Image, "my-registry.io/ollama:v0.3.0")
	}
}

// ---------------------------------------------------------------------------
// PrometheusRule tests
// ---------------------------------------------------------------------------

func TestPrometheusRuleName(t *testing.T) {
	instance := newTestInstance("my-instance")
	name := PrometheusRuleName(instance)
	if name != "my-instance-alerts" {
		t.Errorf("PrometheusRuleName = %q, want %q", name, "my-instance-alerts")
	}
}

func TestBuildPrometheusRule(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.PrometheusRule = &openclawv1alpha1.PrometheusRuleSpec{
		Enabled: Ptr(true),
	}

	pr := BuildPrometheusRule(instance)

	// Check GVK
	gvk := pr.GetObjectKind().GroupVersionKind()
	if gvk != PrometheusRuleGVK() {
		t.Errorf("GVK = %v, want %v", gvk, PrometheusRuleGVK())
	}

	// Check name
	if pr.GetName() != "my-instance-alerts" {
		t.Errorf("name = %q, want %q", pr.GetName(), "my-instance-alerts")
	}

	// Check namespace
	if pr.GetNamespace() != "test-ns" {
		t.Errorf("namespace = %q, want %q", pr.GetNamespace(), "test-ns")
	}

	// Check labels
	labels := pr.GetLabels()
	if labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("missing app.kubernetes.io/name label")
	}

	// Check spec.groups[0].rules has 7 alerts
	spec, ok := pr.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("missing spec")
	}
	groups, ok := spec["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatal("expected exactly 1 rule group")
	}
	group, ok := groups[0].(map[string]interface{})
	if !ok {
		t.Fatal("invalid group")
	}
	rules, ok := group["rules"].([]interface{})
	if !ok {
		t.Fatal("missing rules")
	}
	if len(rules) != 7 {
		t.Errorf("expected 7 alerts, got %d", len(rules))
	}

	// Check all alerts have runbook_url
	for i, r := range rules {
		rule, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("rule %d is not a map", i)
		}
		annotations, ok := rule["annotations"].(map[string]interface{})
		if !ok {
			t.Fatalf("rule %d missing annotations", i)
		}
		runbook, ok := annotations["runbook_url"].(string)
		if !ok || runbook == "" {
			t.Errorf("rule %d missing runbook_url", i)
		}
		if !strings.HasPrefix(runbook, "https://openclaw.rocks/docs/runbooks/") {
			t.Errorf("rule %d runbook_url = %q, expected default base URL", i, runbook)
		}
	}
}

// ---------------------------------------------------------------------------
// Self-Configure tests
// ---------------------------------------------------------------------------

func TestBuildRole_SelfConfigureEnabled(t *testing.T) {
	instance := newTestInstance("sc-role")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled:        true,
		AllowedActions: []openclawv1alpha1.SelfConfigAction{"skills", "config"},
	}

	role := BuildRole(instance)

	// 1 base rule (configmap) + 3 self-configure rules (instances, selfconfigs, secrets) + 0 additional
	if len(role.Rules) < 4 {
		t.Fatalf("expected at least 4 rules, got %d", len(role.Rules))
	}

	// Check for openclawinstances get rule
	foundInstances := false
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			if res == "openclawinstances" {
				foundInstances = true
				if len(rule.ResourceNames) != 1 || rule.ResourceNames[0] != "sc-role" {
					t.Errorf("instances rule resourceNames = %v, want [sc-role]", rule.ResourceNames)
				}
				if len(rule.Verbs) != 1 || rule.Verbs[0] != "get" {
					t.Errorf("instances rule verbs = %v, want [get]", rule.Verbs)
				}
			}
		}
	}
	if !foundInstances {
		t.Error("missing openclawinstances rule")
	}

	// Check for openclawselfconfigs rule
	foundSelfConfigs := false
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			if res != "openclawselfconfigs" {
				continue
			}
			foundSelfConfigs = true
			if len(rule.ResourceNames) > 0 {
				t.Error("selfconfigs rule should not have resourceNames (create cannot be scoped)")
			}
			verbSet := map[string]bool{}
			for _, v := range rule.Verbs {
				verbSet[v] = true
			}
			for _, expected := range []string{"create", "get", "list"} {
				if !verbSet[expected] {
					t.Errorf("selfconfigs rule missing verb %q", expected)
				}
			}
		}
	}
	if !foundSelfConfigs {
		t.Error("missing openclawselfconfigs rule")
	}

	// Check for secrets rule (gateway token secret should be included)
	foundSecrets := false
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			if res != "secrets" {
				continue
			}
			foundSecrets = true
			if len(rule.ResourceNames) == 0 {
				t.Error("secrets rule should have resourceNames for scoping")
			}
			// Should include the gateway token secret
			found := false
			for _, name := range rule.ResourceNames {
				if name == "sc-role-gateway-token" {
					found = true
				}
			}
			if !found {
				t.Errorf("secrets rule resourceNames %v should include sc-role-gateway-token", rule.ResourceNames)
			}
		}
	}
	if !foundSecrets {
		t.Error("missing secrets rule")
	}
}

func TestBuildRole_SelfConfigureDisabled(t *testing.T) {
	instance := newTestInstance("sc-disabled")

	role := BuildRole(instance)

	// Only the 1 base rule (configmap)
	if len(role.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(role.Rules))
	}

	// No openclaw.rocks rules
	for _, rule := range role.Rules {
		for _, ag := range rule.APIGroups {
			if ag == "openclaw.rocks" {
				t.Error("should not have openclaw.rocks rules when self-configure is disabled")
			}
		}
	}
}

func TestBuildRole_SelfConfigureWithEnvFromSecrets(t *testing.T) {
	instance := newTestInstance("sc-envfrom")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled:        true,
		AllowedActions: []openclawv1alpha1.SelfConfigAction{"skills"},
	}
	instance.Spec.EnvFrom = []corev1.EnvFromSource{
		{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-keys"},
			},
		},
	}

	role := BuildRole(instance)

	// Find secrets rule
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			if res == "secrets" {
				// Should include both gateway token and envfrom secret
				nameSet := map[string]bool{}
				for _, name := range rule.ResourceNames {
					nameSet[name] = true
				}
				if !nameSet["api-keys"] {
					t.Error("secrets rule should include envFrom secret 'api-keys'")
				}
				if !nameSet["sc-envfrom-gateway-token"] {
					t.Error("secrets rule should include gateway token secret")
				}
			}
		}
	}
}

func TestBuildServiceAccount_SelfConfigureTokenMount(t *testing.T) {
	instance := newTestInstance("sc-sa-token")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	sa := BuildServiceAccount(instance)

	if sa.AutomountServiceAccountToken == nil || !*sa.AutomountServiceAccountToken {
		t.Error("AutomountServiceAccountToken should be true when self-configure is enabled")
	}
}

func TestBuildServiceAccount_SelfConfigureDisabledTokenMount(t *testing.T) {
	instance := newTestInstance("sc-sa-notoken")

	sa := BuildServiceAccount(instance)

	if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken {
		t.Error("AutomountServiceAccountToken should be false when self-configure is disabled")
	}
}

func TestBuildStatefulSet_SelfConfigureEnvVars(t *testing.T) {
	instance := newTestInstance("sc-env")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	sts := BuildStatefulSet(instance)

	mainContainer := sts.Spec.Template.Spec.Containers[0]
	envMap := map[string]string{}
	for _, ev := range mainContainer.Env {
		envMap[ev.Name] = ev.Value
	}

	if envMap["OPENCLAW_INSTANCE_NAME"] != "sc-env" {
		t.Errorf("OPENCLAW_INSTANCE_NAME = %q, want %q", envMap["OPENCLAW_INSTANCE_NAME"], "sc-env")
	}
	if envMap["OPENCLAW_NAMESPACE"] != "test-ns" {
		t.Errorf("OPENCLAW_NAMESPACE = %q, want %q", envMap["OPENCLAW_NAMESPACE"], "test-ns")
	}
}

func TestBuildStatefulSet_SelfConfigureDisabledNoEnvVars(t *testing.T) {
	instance := newTestInstance("sc-noenv")

	sts := BuildStatefulSet(instance)

	mainContainer := sts.Spec.Template.Spec.Containers[0]
	for _, ev := range mainContainer.Env {
		if ev.Name == "OPENCLAW_INSTANCE_NAME" || ev.Name == "OPENCLAW_NAMESPACE" {
			t.Errorf("should not have %s when self-configure is disabled", ev.Name)
		}
	}
}

func TestBuildStatefulSet_SelfConfigureAutoMount(t *testing.T) {
	instance := newTestInstance("sc-automount")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	sts := BuildStatefulSet(instance)

	if sts.Spec.Template.Spec.AutomountServiceAccountToken == nil ||
		!*sts.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Error("pod AutomountServiceAccountToken should be true when self-configure is enabled")
	}
}

func TestBuildNetworkPolicy_SelfConfigureEgress(t *testing.T) {
	instance := newTestInstance("sc-netpol")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	np := BuildNetworkPolicy(instance)

	// Check for port 6443 egress rule
	found6443 := false
	for _, rule := range np.Spec.Egress {
		for _, port := range rule.Ports {
			if port.Port != nil && port.Port.IntValue() == 6443 {
				found6443 = true
			}
		}
	}
	if !found6443 {
		t.Error("NetworkPolicy should have egress rule for K8s API port 6443")
	}
}

func TestBuildNetworkPolicy_SelfConfigureDisabledNo6443(t *testing.T) {
	instance := newTestInstance("sc-netpol-off")

	np := BuildNetworkPolicy(instance)

	for _, rule := range np.Spec.Egress {
		for _, port := range rule.Ports {
			if port.Port != nil && port.Port.IntValue() == 6443 {
				t.Error("NetworkPolicy should NOT have port 6443 when self-configure is disabled")
			}
		}
	}
}

func TestBuildWorkspaceConfigMap_SelfConfigureSkillInjected(t *testing.T) {
	instance := newTestInstance("sc-ws")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	cm := BuildWorkspaceConfigMap(instance)

	if cm == nil {
		t.Fatal("BuildWorkspaceConfigMap returned nil when self-configure is enabled")
	}

	if _, ok := cm.Data["SELFCONFIG.md"]; !ok {
		t.Error("workspace ConfigMap missing SELFCONFIG.md")
	}
	if _, ok := cm.Data["selfconfig.sh"]; !ok {
		t.Error("workspace ConfigMap missing selfconfig.sh")
	}
}

func TestBuildWorkspaceConfigMap_SelfConfigureMergedWithUserFiles(t *testing.T) {
	instance := newTestInstance("sc-ws-merge")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"README.md": "# My Project",
			"notes.txt": "some notes",
		},
	}

	cm := BuildWorkspaceConfigMap(instance)

	if cm == nil {
		t.Fatal("BuildWorkspaceConfigMap returned nil")
	}

	// User files preserved
	if cm.Data["README.md"] != "# My Project" {
		t.Error("user file README.md not preserved")
	}
	if cm.Data["notes.txt"] != "some notes" {
		t.Error("user file notes.txt not preserved")
	}

	// Self-configure files injected
	if _, ok := cm.Data["SELFCONFIG.md"]; !ok {
		t.Error("missing SELFCONFIG.md")
	}
	if _, ok := cm.Data["selfconfig.sh"]; !ok {
		t.Error("missing selfconfig.sh")
	}
}

func TestBuildWorkspaceConfigMap_SelfConfigureDisabledNoFiles(t *testing.T) {
	instance := newTestInstance("sc-ws-off")

	cm := BuildWorkspaceConfigMap(instance)

	if cm != nil {
		t.Error("BuildWorkspaceConfigMap should return nil when no workspace files and self-configure disabled")
	}
}

func TestHasWorkspaceFiles_SelfConfigureEnabled(t *testing.T) {
	instance := newTestInstance("sc-hasws")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	if !hasWorkspaceFiles(instance) {
		t.Error("hasWorkspaceFiles should return true when self-configure is enabled")
	}
}

func TestBuildStatefulSet_SelfConfigureWorkspaceVolume(t *testing.T) {
	instance := newTestInstance("sc-ws-vol")
	instance.Spec.SelfConfigure = openclawv1alpha1.SelfConfigureSpec{
		Enabled: true,
	}

	sts := BuildStatefulSet(instance)

	// Should have workspace-init volume
	foundVol := false
	for _, vol := range sts.Spec.Template.Spec.Volumes {
		if vol.Name == "workspace-init" {
			foundVol = true
			if vol.ConfigMap == nil {
				t.Error("workspace-init volume should reference a ConfigMap")
			} else if vol.ConfigMap.Name != "sc-ws-vol-workspace" {
				t.Errorf("workspace-init volume ConfigMap name = %q, want %q", vol.ConfigMap.Name, "sc-ws-vol-workspace")
			}
		}
	}
	if !foundVol {
		t.Error("missing workspace-init volume when self-configure is enabled")
	}
}

func TestBuildPrometheusRule_CustomRunbookURL(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.PrometheusRule = &openclawv1alpha1.PrometheusRuleSpec{
		Enabled:        Ptr(true),
		RunbookBaseURL: "https://wiki.example.com/runbooks",
	}

	pr := BuildPrometheusRule(instance)

	spec := pr.Object["spec"].(map[string]interface{})
	groups := spec["groups"].([]interface{})
	group := groups[0].(map[string]interface{})
	rules := group["rules"].([]interface{})

	firstRule := rules[0].(map[string]interface{})
	annotations := firstRule["annotations"].(map[string]interface{})
	runbook := annotations["runbook_url"].(string)
	if !strings.HasPrefix(runbook, "https://wiki.example.com/runbooks/") {
		t.Errorf("runbook_url = %q, expected custom base URL", runbook)
	}
}

func TestBuildPrometheusRule_CustomLabels(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.PrometheusRule = &openclawv1alpha1.PrometheusRuleSpec{
		Enabled: Ptr(true),
		Labels: map[string]string{
			"release": "kube-prometheus-stack",
		},
	}

	pr := BuildPrometheusRule(instance)
	labels := pr.GetLabels()

	if labels["release"] != "kube-prometheus-stack" {
		t.Errorf("custom label release = %q, want %q", labels["release"], "kube-prometheus-stack")
	}
	// Standard labels should still be present
	if labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("missing standard label")
	}
}

// ---------------------------------------------------------------------------
// Grafana dashboard tests
// ---------------------------------------------------------------------------

func TestGrafanaDashboardOperatorName(t *testing.T) {
	instance := newTestInstance("my-instance")
	name := GrafanaDashboardOperatorName(instance)
	if name != "my-instance-dashboard-operator" {
		t.Errorf("GrafanaDashboardOperatorName = %q, want %q", name, "my-instance-dashboard-operator")
	}
}

func TestGrafanaDashboardInstanceName(t *testing.T) {
	instance := newTestInstance("my-instance")
	name := GrafanaDashboardInstanceName(instance)
	if name != "my-instance-dashboard-instance" {
		t.Errorf("GrafanaDashboardInstanceName = %q, want %q", name, "my-instance-dashboard-instance")
	}
}

func TestBuildGrafanaDashboardOperator(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.GrafanaDashboard = &openclawv1alpha1.GrafanaDashboardSpec{
		Enabled: Ptr(true),
	}

	cm := BuildGrafanaDashboardOperator(instance)

	// Check name
	if cm.Name != "my-instance-dashboard-operator" {
		t.Errorf("name = %q, want %q", cm.Name, "my-instance-dashboard-operator")
	}

	// Check grafana_dashboard label
	if cm.Labels["grafana_dashboard"] != "1" {
		t.Errorf("grafana_dashboard label = %q, want %q", cm.Labels["grafana_dashboard"], "1")
	}

	// Check grafana_folder annotation
	if cm.Annotations["grafana_folder"] != "OpenClaw" {
		t.Errorf("grafana_folder annotation = %q, want %q", cm.Annotations["grafana_folder"], "OpenClaw")
	}

	// Check data key exists and contains valid JSON
	dashJSON, ok := cm.Data["openclaw-operator.json"]
	if !ok {
		t.Fatal("missing openclaw-operator.json data key")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(dashJSON), &parsed); err != nil {
		t.Fatalf("invalid JSON in dashboard: %v", err)
	}

	// Check title
	if parsed["title"] != "OpenClaw Operator" {
		t.Errorf("dashboard title = %q, want %q", parsed["title"], "OpenClaw Operator")
	}

	// Check UID
	if parsed["uid"] != "openclaw-operator-overview" {
		t.Errorf("dashboard UID = %q, want %q", parsed["uid"], "openclaw-operator-overview")
	}

	// Check template variables exist
	templating, ok := parsed["templating"].(map[string]interface{})
	if !ok {
		t.Fatal("missing templating")
	}
	vars, ok := templating["list"].([]interface{})
	if !ok || len(vars) < 3 {
		t.Fatalf("expected at least 3 template variables, got %d", len(vars))
	}

	// Check panels exist (should be >10)
	panels, ok := parsed["panels"].([]interface{})
	if !ok {
		t.Fatal("missing panels")
	}
	if len(panels) < 10 {
		t.Errorf("expected at least 10 panels, got %d", len(panels))
	}
}

func TestBuildGrafanaDashboardInstance(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.GrafanaDashboard = &openclawv1alpha1.GrafanaDashboardSpec{
		Enabled: Ptr(true),
	}

	cm := BuildGrafanaDashboardInstance(instance)

	// Check name
	if cm.Name != "my-instance-dashboard-instance" {
		t.Errorf("name = %q, want %q", cm.Name, "my-instance-dashboard-instance")
	}

	// Check grafana_dashboard label
	if cm.Labels["grafana_dashboard"] != "1" {
		t.Errorf("grafana_dashboard label = %q, want %q", cm.Labels["grafana_dashboard"], "1")
	}

	// Check data key exists and contains valid JSON
	dashJSON, ok := cm.Data["openclaw-instance.json"]
	if !ok {
		t.Fatal("missing openclaw-instance.json data key")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(dashJSON), &parsed); err != nil {
		t.Fatalf("invalid JSON in dashboard: %v", err)
	}

	// Check title
	if parsed["title"] != "OpenClaw Instance" {
		t.Errorf("dashboard title = %q, want %q", parsed["title"], "OpenClaw Instance")
	}

	// Check UID
	if parsed["uid"] != "openclaw-instance-detail" {
		t.Errorf("dashboard UID = %q, want %q", parsed["uid"], "openclaw-instance-detail")
	}

	// Check panels exist (should be >10)
	panels, ok := parsed["panels"].([]interface{})
	if !ok {
		t.Fatal("missing panels")
	}
	if len(panels) < 10 {
		t.Errorf("expected at least 10 panels, got %d", len(panels))
	}
}

func TestBuildGrafanaDashboard_CustomLabelsAndFolder(t *testing.T) {
	instance := newTestInstance("my-instance")
	instance.Spec.Observability.Metrics.GrafanaDashboard = &openclawv1alpha1.GrafanaDashboardSpec{
		Enabled: Ptr(true),
		Labels: map[string]string{
			"custom-label": "custom-value",
		},
		Folder: "Infrastructure",
	}

	cm := BuildGrafanaDashboardOperator(instance)

	// Check custom label
	if cm.Labels["custom-label"] != "custom-value" {
		t.Errorf("custom label = %q, want %q", cm.Labels["custom-label"], "custom-value")
	}

	// Check custom folder
	if cm.Annotations["grafana_folder"] != "Infrastructure" {
		t.Errorf("grafana_folder = %q, want %q", cm.Annotations["grafana_folder"], "Infrastructure")
	}

	// Standard labels should still be present
	if cm.Labels["grafana_dashboard"] != "1" {
		t.Error("missing grafana_dashboard label")
	}
	if cm.Labels["app.kubernetes.io/name"] != "openclaw" {
		t.Error("missing standard label")
	}
}

func TestPrometheusRuleGVK(t *testing.T) {
	gvk := PrometheusRuleGVK()
	if gvk.Group != "monitoring.coreos.com" {
		t.Errorf("group = %q, want %q", gvk.Group, "monitoring.coreos.com")
	}
	if gvk.Version != "v1" {
		t.Errorf("version = %q, want %q", gvk.Version, "v1")
	}
	if gvk.Kind != "PrometheusRule" {
		t.Errorf("kind = %q, want %q", gvk.Kind, "PrometheusRule")
	}
}

// ---------------------------------------------------------------------------
// HPA tests
// ---------------------------------------------------------------------------

func TestHPAName(t *testing.T) {
	instance := newTestInstance("my-app")
	if got := HPAName(instance); got != "my-app" {
		t.Errorf("HPAName() = %q, want %q", got, "my-app")
	}
}

func TestIsHPAEnabled(t *testing.T) {
	tests := []struct {
		name     string
		as       *openclawv1alpha1.AutoScalingSpec
		expected bool
	}{
		{"nil spec", nil, false},
		{"nil enabled", &openclawv1alpha1.AutoScalingSpec{}, false},
		{"enabled false", &openclawv1alpha1.AutoScalingSpec{Enabled: Ptr(false)}, false},
		{"enabled true", &openclawv1alpha1.AutoScalingSpec{Enabled: Ptr(true)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := newTestInstance("test")
			instance.Spec.Availability.AutoScaling = tt.as
			if got := IsHPAEnabled(instance); got != tt.expected {
				t.Errorf("IsHPAEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildHPA_Defaults(t *testing.T) {
	instance := newTestInstance("my-app")
	instance.Spec.Availability.AutoScaling = &openclawv1alpha1.AutoScalingSpec{
		Enabled: Ptr(true),
	}

	hpa := BuildHPA(instance)

	if hpa.Name != "my-app" {
		t.Errorf("name = %q, want %q", hpa.Name, "my-app")
	}
	if hpa.Namespace != "test-ns" {
		t.Errorf("namespace = %q, want %q", hpa.Namespace, "test-ns")
	}
	if hpa.Spec.ScaleTargetRef.Kind != "StatefulSet" {
		t.Errorf("target kind = %q, want StatefulSet", hpa.Spec.ScaleTargetRef.Kind)
	}
	if hpa.Spec.ScaleTargetRef.Name != StatefulSetName(instance) {
		t.Errorf("target name = %q, want %q", hpa.Spec.ScaleTargetRef.Name, StatefulSetName(instance))
	}
	if *hpa.Spec.MinReplicas != 1 {
		t.Errorf("minReplicas = %d, want 1", *hpa.Spec.MinReplicas)
	}
	if hpa.Spec.MaxReplicas != 5 {
		t.Errorf("maxReplicas = %d, want 5", hpa.Spec.MaxReplicas)
	}
	if len(hpa.Spec.Metrics) != 1 {
		t.Fatalf("metrics count = %d, want 1", len(hpa.Spec.Metrics))
	}
	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 80 {
		t.Errorf("cpu target = %d, want 80", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}
}

func TestBuildHPA_CustomValues(t *testing.T) {
	instance := newTestInstance("my-app")
	instance.Spec.Availability.AutoScaling = &openclawv1alpha1.AutoScalingSpec{
		Enabled:              Ptr(true),
		MinReplicas:          Ptr(int32(2)),
		MaxReplicas:          Ptr(int32(10)),
		TargetCPUUtilization: Ptr(int32(60)),
	}

	hpa := BuildHPA(instance)

	if *hpa.Spec.MinReplicas != 2 {
		t.Errorf("minReplicas = %d, want 2", *hpa.Spec.MinReplicas)
	}
	if hpa.Spec.MaxReplicas != 10 {
		t.Errorf("maxReplicas = %d, want 10", hpa.Spec.MaxReplicas)
	}
	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 60 {
		t.Errorf("cpu target = %d, want 60", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}
}

func TestBuildHPA_WithMemoryMetric(t *testing.T) {
	instance := newTestInstance("my-app")
	instance.Spec.Availability.AutoScaling = &openclawv1alpha1.AutoScalingSpec{
		Enabled:                 Ptr(true),
		TargetMemoryUtilization: Ptr(int32(70)),
	}

	hpa := BuildHPA(instance)

	if len(hpa.Spec.Metrics) != 2 {
		t.Fatalf("metrics count = %d, want 2", len(hpa.Spec.Metrics))
	}
	memMetric := hpa.Spec.Metrics[1]
	if string(memMetric.Resource.Name) != "memory" {
		t.Errorf("second metric resource = %q, want memory", memMetric.Resource.Name)
	}
	if *memMetric.Resource.Target.AverageUtilization != 70 {
		t.Errorf("memory target = %d, want 70", *memMetric.Resource.Target.AverageUtilization)
	}
}

func TestStatefulSetReplicas_HPAEnabled(t *testing.T) {
	instance := newTestInstance("my-app")
	instance.Spec.Availability.AutoScaling = &openclawv1alpha1.AutoScalingSpec{
		Enabled: Ptr(true),
	}

	sts := BuildStatefulSet(instance)
	if sts.Spec.Replicas != nil {
		t.Errorf("replicas should be nil when HPA is enabled, got %d", *sts.Spec.Replicas)
	}
}

func TestStatefulSetReplicas_HPADisabled(t *testing.T) {
	instance := newTestInstance("my-app")

	sts := BuildStatefulSet(instance)
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 1 {
		t.Errorf("replicas should be 1 when HPA is disabled")
	}
}

// ---------------------------------------------------------------------------
// Web terminal sidecar tests
// ---------------------------------------------------------------------------

func TestBuildStatefulSet_WebTerminalEnabled(t *testing.T) {
	instance := newTestInstance("web-terminal-test")
	instance.Spec.WebTerminal.Enabled = true

	sts := BuildStatefulSet(instance)
	containers := sts.Spec.Template.Spec.Containers

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers (main + web-terminal), got %d", len(containers))
	}

	var wt *corev1.Container
	for i := range containers {
		if containers[i].Name == "web-terminal" {
			wt = &containers[i]
			break
		}
	}
	if wt == nil {
		t.Fatal("web-terminal container not found")
	}

	// Image defaults
	if wt.Image != "tsl0922/ttyd:latest" {
		t.Errorf("web-terminal image = %q, want default", wt.Image)
	}

	// Port
	if len(wt.Ports) != 1 {
		t.Fatalf("web-terminal container should have 1 port, got %d", len(wt.Ports))
	}
	if wt.Ports[0].ContainerPort != WebTerminalPort {
		t.Errorf("web-terminal port = %d, want %d", wt.Ports[0].ContainerPort, WebTerminalPort)
	}
	if wt.Ports[0].Name != "web-terminal" {
		t.Errorf("web-terminal port name = %q, want %q", wt.Ports[0].Name, "web-terminal")
	}

	// Security context
	sc := wt.SecurityContext
	if sc == nil {
		t.Fatal("web-terminal security context is nil")
	}
	if sc.ReadOnlyRootFilesystem == nil || *sc.ReadOnlyRootFilesystem {
		t.Error("web-terminal: readOnlyRootFilesystem should be false")
	}
	if sc.RunAsUser == nil || *sc.RunAsUser != 1000 {
		t.Errorf("web-terminal: runAsUser = %v, want 1000", sc.RunAsUser)
	}
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		t.Error("web-terminal: runAsNonRoot should be true")
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		t.Error("web-terminal: allowPrivilegeEscalation should be false")
	}

	// Resource defaults
	cpuReq := wt.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "50m" {
		t.Errorf("web-terminal cpu request = %v, want 50m", cpuReq.String())
	}
	memReq := wt.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("64Mi")) != 0 {
		t.Errorf("web-terminal memory request = %v, want 64Mi", memReq.String())
	}
	cpuLim := wt.Resources.Limits[corev1.ResourceCPU]
	if cpuLim.String() != "200m" {
		t.Errorf("web-terminal cpu limit = %v, want 200m", cpuLim.String())
	}
	memLim := wt.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("128Mi")) != 0 {
		t.Errorf("web-terminal memory limit = %v, want 128Mi", memLim.String())
	}

	// Volume mounts
	assertVolumeMount(t, wt.VolumeMounts, "data", "/home/openclaw/.openclaw")
	assertVolumeMount(t, wt.VolumeMounts, "web-terminal-tmp", "/tmp")

	// Data mount should NOT be read-only by default
	for _, m := range wt.VolumeMounts {
		if m.Name == "data" && m.ReadOnly {
			t.Error("data mount should not be read-only by default")
		}
	}

	// Volumes - check web-terminal-tmp volume exists
	volumes := sts.Spec.Template.Spec.Volumes
	var wtVol *corev1.Volume
	for i := range volumes {
		if volumes[i].Name == "web-terminal-tmp" {
			wtVol = &volumes[i]
			break
		}
	}
	if wtVol == nil {
		t.Fatal("web-terminal-tmp volume not found")
	}
	if wtVol.EmptyDir == nil {
		t.Fatal("web-terminal-tmp volume should be emptyDir")
	}

	// Command
	if len(wt.Command) != 3 {
		t.Fatalf("web-terminal command should have 3 elements, got %d", len(wt.Command))
	}
	if wt.Command[0] != "sh" || wt.Command[1] != "-c" {
		t.Errorf("web-terminal command prefix = %v, want [sh -c ...]", wt.Command[:2])
	}
	if !strings.Contains(wt.Command[2], "exec ttyd") {
		t.Errorf("web-terminal command should contain 'exec ttyd', got %q", wt.Command[2])
	}

	// No credential env vars by default
	if len(wt.Env) != 0 {
		t.Errorf("web-terminal should have 0 env vars by default, got %d", len(wt.Env))
	}
}

func TestBuildStatefulSet_WebTerminalCustomImage(t *testing.T) {
	instance := newTestInstance("wt-custom-img")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.Image = openclawv1alpha1.WebTerminalImageSpec{
		Repository: "my-registry.io/ttyd",
		Tag:        "v1.7.0",
	}

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	if wt.Image != "my-registry.io/ttyd:v1.7.0" {
		t.Errorf("web-terminal image = %q, want my-registry.io/ttyd:v1.7.0", wt.Image)
	}
}

func TestBuildStatefulSet_WebTerminalDigest(t *testing.T) {
	instance := newTestInstance("wt-digest")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.Image = openclawv1alpha1.WebTerminalImageSpec{
		Repository: "tsl0922/ttyd",
		Digest:     "sha256:abcdef1234567890",
	}

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	if wt.Image != "tsl0922/ttyd@sha256:abcdef1234567890" {
		t.Errorf("web-terminal image = %q, want tsl0922/ttyd@sha256:abcdef1234567890", wt.Image)
	}
}

func TestBuildStatefulSet_WebTerminalCustomResources(t *testing.T) {
	instance := newTestInstance("wt-resources")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.Resources = openclawv1alpha1.ResourcesSpec{
		Requests: openclawv1alpha1.ResourceList{CPU: "100m", Memory: "128Mi"},
		Limits:   openclawv1alpha1.ResourceList{CPU: "500m", Memory: "256Mi"},
	}

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	cpuReq := wt.Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "100m" {
		t.Errorf("cpu request = %v, want 100m", cpuReq.String())
	}
	memReq := wt.Resources.Requests[corev1.ResourceMemory]
	if memReq.Cmp(resource.MustParse("128Mi")) != 0 {
		t.Errorf("memory request = %v, want 128Mi", memReq.String())
	}
	cpuLim := wt.Resources.Limits[corev1.ResourceCPU]
	if cpuLim.String() != "500m" {
		t.Errorf("cpu limit = %v, want 500m", cpuLim.String())
	}
	memLim := wt.Resources.Limits[corev1.ResourceMemory]
	if memLim.Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("memory limit = %v, want 256Mi", memLim.String())
	}
}

func TestBuildStatefulSet_WebTerminalReadOnly(t *testing.T) {
	instance := newTestInstance("wt-readonly")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.ReadOnly = true

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	// Command should include -R flag
	if !strings.Contains(wt.Command[2], "-R") {
		t.Errorf("web-terminal command should contain '-R' for read-only, got %q", wt.Command[2])
	}

	// Data mount should be read-only
	for _, m := range wt.VolumeMounts {
		if m.Name == "data" && !m.ReadOnly {
			t.Error("data mount should be read-only when readOnly is true")
		}
	}
}

func TestBuildStatefulSet_WebTerminalCredential(t *testing.T) {
	instance := newTestInstance("wt-cred")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.Credential = &openclawv1alpha1.WebTerminalCredentialSpec{
		SecretRef: corev1.LocalObjectReference{Name: "wt-secret"},
	}

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	// Command should include -c flag with credential env vars
	if !strings.Contains(wt.Command[2], `-c "${TTYD_USERNAME}:${TTYD_PASSWORD}"`) {
		t.Errorf("web-terminal command should contain credential flag, got %q", wt.Command[2])
	}

	// Should have TTYD_USERNAME and TTYD_PASSWORD env vars
	if len(wt.Env) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(wt.Env))
	}

	var foundUsername, foundPassword bool
	for _, env := range wt.Env {
		if env.Name == "TTYD_USERNAME" {
			foundUsername = true
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				t.Error("TTYD_USERNAME should use secretKeyRef")
			} else {
				if env.ValueFrom.SecretKeyRef.Name != "wt-secret" {
					t.Errorf("TTYD_USERNAME secret name = %q, want wt-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != "username" {
					t.Errorf("TTYD_USERNAME secret key = %q, want username", env.ValueFrom.SecretKeyRef.Key)
				}
			}
		}
		if env.Name == "TTYD_PASSWORD" {
			foundPassword = true
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				t.Error("TTYD_PASSWORD should use secretKeyRef")
			} else {
				if env.ValueFrom.SecretKeyRef.Name != "wt-secret" {
					t.Errorf("TTYD_PASSWORD secret name = %q, want wt-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != "password" {
					t.Errorf("TTYD_PASSWORD secret key = %q, want password", env.ValueFrom.SecretKeyRef.Key)
				}
			}
		}
	}
	if !foundUsername {
		t.Error("TTYD_USERNAME env var not found")
	}
	if !foundPassword {
		t.Error("TTYD_PASSWORD env var not found")
	}
}

func TestBuildStatefulSet_WebTerminalDisabled(t *testing.T) {
	instance := newTestInstance("no-web-terminal")

	sts := BuildStatefulSet(instance)

	// No web-terminal container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			t.Error("web-terminal container should not be present when disabled")
		}
	}

	// No web-terminal-tmp volume
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == "web-terminal-tmp" {
			t.Error("web-terminal-tmp volume should not be present when disabled")
		}
	}
}

func TestBuildStatefulSet_WebTerminalContainerDefaults(t *testing.T) {
	instance := newTestInstance("wt-defaults")
	instance.Spec.WebTerminal.Enabled = true

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	// Verify Kubernetes default fields are set
	if wt.TerminationMessagePath != corev1.TerminationMessagePathDefault {
		t.Errorf("TerminationMessagePath = %q, want default", wt.TerminationMessagePath)
	}
	if wt.TerminationMessagePolicy != corev1.TerminationMessageReadFile {
		t.Errorf("TerminationMessagePolicy = %v, want ReadFile", wt.TerminationMessagePolicy)
	}
	if wt.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("ImagePullPolicy = %v, want IfNotPresent", wt.ImagePullPolicy)
	}

	// Seccomp profile
	if wt.SecurityContext.SeccompProfile == nil {
		t.Fatal("seccomp profile should be set")
	}
	if wt.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Errorf("seccomp type = %v, want RuntimeDefault", wt.SecurityContext.SeccompProfile.Type)
	}
}

func TestBuildService_WithWebTerminal(t *testing.T) {
	instance := newTestInstance("svc-web-terminal")
	instance.Spec.WebTerminal.Enabled = true

	svc := BuildService(instance)

	if len(svc.Spec.Ports) != 3 {
		t.Fatalf("expected 3 ports with web terminal, got %d", len(svc.Spec.Ports))
	}

	assertServicePort(t, svc.Spec.Ports, "gateway", int32(GatewayPort))
	assertServicePort(t, svc.Spec.Ports, "canvas", int32(CanvasPort))
	assertServicePort(t, svc.Spec.Ports, "web-terminal", int32(WebTerminalPort))
}

func TestBuildNetworkPolicy_WebTerminalIngressPort(t *testing.T) {
	instance := newTestInstance("np-web-terminal")
	instance.Spec.WebTerminal.Enabled = true

	np := BuildNetworkPolicy(instance)

	// Default ingress rule should have 3 ports (gateway, canvas, web-terminal)
	if len(np.Spec.Ingress) == 0 {
		t.Fatal("expected at least one ingress rule")
	}

	ports := np.Spec.Ingress[0].Ports
	if len(ports) != 3 {
		t.Fatalf("expected 3 ingress ports with web terminal, got %d", len(ports))
	}

	// Verify web-terminal port is present
	found := false
	for _, p := range ports {
		if p.Port != nil && p.Port.IntValue() == WebTerminalPort {
			found = true
			break
		}
	}
	if !found {
		t.Error("web-terminal port not found in NetworkPolicy ingress ports")
	}
}

func TestBuildStatefulSet_WebTerminalReadOnlyWithCredential(t *testing.T) {
	instance := newTestInstance("wt-readonly-cred")
	instance.Spec.WebTerminal.Enabled = true
	instance.Spec.WebTerminal.ReadOnly = true
	instance.Spec.WebTerminal.Credential = &openclawv1alpha1.WebTerminalCredentialSpec{
		SecretRef: corev1.LocalObjectReference{Name: "cred-secret"},
	}

	sts := BuildStatefulSet(instance)

	var wt corev1.Container
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == "web-terminal" {
			wt = c
			break
		}
	}

	// Command should include both -R and -c flags
	if !strings.Contains(wt.Command[2], "-R") {
		t.Errorf("command should contain '-R', got %q", wt.Command[2])
	}
	if !strings.Contains(wt.Command[2], `-c "${TTYD_USERNAME}:${TTYD_PASSWORD}"`) {
		t.Errorf("command should contain credential flag, got %q", wt.Command[2])
	}
}
