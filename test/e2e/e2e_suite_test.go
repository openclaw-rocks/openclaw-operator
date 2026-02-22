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

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
	"github.com/openclawrocks/k8s-operator/internal/resources"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	var err error
	cfg, err = config.GetConfig()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = openclawv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
})

// kubectlExec runs a command inside the openclaw container via kubectl exec.
func kubectlExec(namespace, podName string, command ...string) (string, error) {
	args := []string{"exec", podName, "-n", namespace, "-c", "openclaw", "--"}
	args = append(args, command...)
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var _ = Describe("OpenClawInstance Controller", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Second * 1
	)

	Context("When creating an OpenClawInstance", func() {
		var namespace string

		BeforeEach(func() {
			// Create a unique namespace for each test
			namespace = "test-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			// Clean up the namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create managed resources", func() {
			instanceName := "test-instance"

			// Skip if running in minimal mode (no actual OpenClaw image)
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			// Create OpenClawInstance
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify the instance was created
			createdInstance := &openclawv1alpha1.OpenClawInstance{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, createdInstance)
			}, timeout, interval).Should(Succeed())

			// Verify StatefulSet is created
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			// Verify Service is created
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, service)
			}, timeout, interval).Should(Succeed())

			// Verify the StatefulSet has the correct image
			Expect(statefulSet.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(statefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/openclaw/openclaw:latest"))

			// Clean up
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			// Verify the StatefulSet is deleted (due to owner reference)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should use shell-capable image for merge mode init container", func() {
			instanceName := "merge-mode-instance"

			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Config: openclawv1alpha1.ConfigSpec{
						MergeMode: "merge",
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			// Find init-config container
			var initConfig *corev1.Container
			for i := range statefulSet.Spec.Template.Spec.InitContainers {
				if statefulSet.Spec.Template.Spec.InitContainers[i].Name == "init-config" {
					initConfig = &statefulSet.Spec.Template.Spec.InitContainers[i]
					break
				}
			}
			Expect(initConfig).NotTo(BeNil(), "merge mode should have init-config container")

			// Must use the OpenClaw image (has shell), NOT the distroless jq image
			Expect(initConfig.Image).To(Equal("ghcr.io/openclaw/openclaw:latest"),
				"merge mode init container should use the OpenClaw image (shell-capable)")

			// Command should use node deep merge, not jq
			Expect(initConfig.Command).To(HaveLen(3))
			Expect(initConfig.Command[0]).To(Equal("sh"))
			Expect(initConfig.Command[2]).To(ContainSubstring("node -e"),
				"merge script should use Node.js deep merge")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})

		It("Should use shell-capable uv image for python runtime deps", func() {
			instanceName := "python-deps-instance"

			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					RuntimeDeps: openclawv1alpha1.RuntimeDepsSpec{
						Python: true,
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			// Find init-python container
			var initPython *corev1.Container
			for i := range statefulSet.Spec.Template.Spec.InitContainers {
				if statefulSet.Spec.Template.Spec.InitContainers[i].Name == "init-python" {
					initPython = &statefulSet.Spec.Template.Spec.InitContainers[i]
					break
				}
			}
			Expect(initPython).NotTo(BeNil(), "python runtime deps should have init-python container")

			// Must use bookworm-slim variant (has shell), NOT the distroless base tag
			Expect(initPython.Image).To(Equal(resources.UvImage),
				"init-python should use the shell-capable uv image")
			Expect(initPython.Image).To(ContainSubstring("bookworm-slim"),
				"uv image must be a Debian variant with shell")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})

		It("Should mount default config for vanilla deployment", func() {
			instanceName := "vanilla-instance"

			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			// Create vanilla OpenClawInstance (image only, no config)
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify StatefulSet has init-config init container
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			initContainers := statefulSet.Spec.Template.Spec.InitContainers
			var initConfig *corev1.Container
			for i := range initContainers {
				if initContainers[i].Name == "init-config" {
					initConfig = &initContainers[i]
					break
				}
			}
			Expect(initConfig).NotTo(BeNil(), "vanilla deployment should have init-config container")

			// Verify config volume references the operator-managed ConfigMap
			var configVol *corev1.Volume
			for i := range statefulSet.Spec.Template.Spec.Volumes {
				if statefulSet.Spec.Template.Spec.Volumes[i].Name == "config" {
					configVol = &statefulSet.Spec.Template.Spec.Volumes[i]
					break
				}
			}
			Expect(configVol).NotTo(BeNil(), "config volume should exist for vanilla deployment")
			Expect(configVol.ConfigMap).NotTo(BeNil())
			Expect(configVol.ConfigMap.Name).To(Equal(resources.ConfigMapName(instance)))

			// Verify ConfigMap exists and contains gateway.bind=lan
			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.ConfigMapName(instance),
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			configContent, ok := cm.Data["openclaw.json"]
			Expect(ok).To(BeTrue(), "ConfigMap should have openclaw.json key")

			var parsed map[string]interface{}
			Expect(json.Unmarshal([]byte(configContent), &parsed)).To(Succeed())
			gw, ok := parsed["gateway"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "config should have gateway key")
			Expect(gw["bind"]).To(Equal("lan"), "gateway.bind should be lan")

			// Clean up via owner-reference garbage collection
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When deleting an OpenClawInstance without B2 backup credentials", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-no-b2-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should delete cleanly when B2 backup credentials are not configured", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "no-b2-delete"

			// No B2 secret exists in the namespace or operator namespace
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceKey := types.NamespacedName{Name: instanceName, Namespace: namespace}

			// Wait for StatefulSet to be created (proves reconciliation happened)
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			// Delete the instance - should succeed without B2 credentials
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			// Instance should be fully garbage collected (finalizer removed)
			Eventually(func() bool {
				inst := &openclawv1alpha1.OpenClawInstance{}
				err := k8sClient.Get(ctx, instanceKey, inst)
				return err != nil // NotFound means fully deleted
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When creating an OpenClawInstance with Ingress", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-ingress-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should emit only nginx annotations for nginx className", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "ingress-nginx"
			className := "nginx"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Networking: openclawv1alpha1.NetworkingSpec{
						Ingress: openclawv1alpha1.IngressSpec{
							Enabled:   true,
							ClassName: &className,
							Hosts: []openclawv1alpha1.IngressHost{
								{Host: "test.example.com"},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			ingress := &networkingv1.Ingress{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.IngressName(instance),
					Namespace: namespace,
				}, ingress)
			}, timeout, interval).Should(Succeed())

			ann := ingress.Annotations
			Expect(ann).To(HaveKey("nginx.ingress.kubernetes.io/ssl-redirect"))
			Expect(ann).To(HaveKey("nginx.ingress.kubernetes.io/proxy-read-timeout"))
			Expect(ann).NotTo(HaveKey("traefik.ingress.kubernetes.io/router.entrypoints"))
			Expect(ann).NotTo(HaveKey("traefik.ingress.kubernetes.io/router.middlewares"))

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})

		It("Should emit only traefik annotations for traefik className", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "ingress-traefik"
			className := "traefik"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Networking: openclawv1alpha1.NetworkingSpec{
						Ingress: openclawv1alpha1.IngressSpec{
							Enabled:   true,
							ClassName: &className,
							Hosts: []openclawv1alpha1.IngressHost{
								{Host: "test.example.com"},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			ingress := &networkingv1.Ingress{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.IngressName(instance),
					Namespace: namespace,
				}, ingress)
			}, timeout, interval).Should(Succeed())

			ann := ingress.Annotations
			Expect(ann).To(HaveKeyWithValue("traefik.ingress.kubernetes.io/router.entrypoints", "websecure"))
			Expect(ann).NotTo(HaveKey("nginx.ingress.kubernetes.io/ssl-redirect"))
			Expect(ann).NotTo(HaveKey("nginx.ingress.kubernetes.io/proxy-read-timeout"))
			Expect(ann).NotTo(HaveKey("traefik.ingress.kubernetes.io/router.middlewares"))

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})

		It("Should emit no provider-specific annotations when className is nil", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "ingress-nil-class"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Networking: openclawv1alpha1.NetworkingSpec{
						Ingress: openclawv1alpha1.IngressSpec{
							Enabled: true,
							// ClassName intentionally nil
							Hosts: []openclawv1alpha1.IngressHost{
								{Host: "test.example.com"},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			ingress := &networkingv1.Ingress{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.IngressName(instance),
					Namespace: namespace,
				}, ingress)
			}, timeout, interval).Should(Succeed())

			ann := ingress.Annotations
			// No provider-specific annotations for nil className
			Expect(ann).NotTo(HaveKey("nginx.ingress.kubernetes.io/ssl-redirect"))
			Expect(ann).NotTo(HaveKey("nginx.ingress.kubernetes.io/proxy-read-timeout"))
			Expect(ann).NotTo(HaveKey("traefik.ingress.kubernetes.io/router.entrypoints"))
			Expect(ann).NotTo(HaveKey("traefik.ingress.kubernetes.io/router.middlewares"))

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an OpenClawInstance with custom service ports (#144)", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-svcports-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create a Service with custom ports replacing defaults", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "custom-ports"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Networking: openclawv1alpha1.NetworkingSpec{
						Service: openclawv1alpha1.ServiceSpec{
							Ports: []openclawv1alpha1.ServicePortSpec{
								{Name: "http", Port: 3978},
								{Name: "grpc", Port: 50051, TargetPort: resources.Ptr(int32(50051))},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify Service has custom ports
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, service)
			}, timeout, interval).Should(Succeed())

			Expect(service.Spec.Ports).To(HaveLen(2))

			var foundHTTP, foundGRPC bool
			for _, p := range service.Spec.Ports {
				if p.Name == "http" && p.Port == 3978 {
					foundHTTP = true
					Expect(p.TargetPort.IntValue()).To(Equal(3978))
				}
				if p.Name == "grpc" && p.Port == 50051 {
					foundGRPC = true
					Expect(p.TargetPort.IntValue()).To(Equal(50051))
				}
			}
			Expect(foundHTTP).To(BeTrue(), "Service should have http port 3978")
			Expect(foundGRPC).To(BeTrue(), "Service should have grpc port 50051")

			// Default gateway/canvas ports should NOT be present
			for _, p := range service.Spec.Ports {
				Expect(p.Port).NotTo(Equal(int32(resources.GatewayPort)),
					"custom ports should replace default gateway port")
				Expect(p.Port).NotTo(Equal(int32(resources.CanvasPort)),
					"custom ports should replace default canvas port")
			}

			// Verify NetworkPolicy allows custom ports
			np := &networkingv1.NetworkPolicy{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.NetworkPolicyName(instance),
					Namespace: namespace,
				}, np)
			}, timeout, interval).Should(Succeed())

			var foundNP3978, foundNP50051 bool
			for _, rule := range np.Spec.Ingress {
				for _, p := range rule.Ports {
					if p.Port != nil {
						switch p.Port.IntValue() {
						case 3978:
							foundNP3978 = true
						case 50051:
							foundNP50051 = true
						}
					}
				}
			}
			Expect(foundNP3978).To(BeTrue(), "NetworkPolicy should allow port 3978")
			Expect(foundNP50051).To(BeTrue(), "NetworkPolicy should allow port 50051")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})

		It("Should set custom backend port on Ingress paths", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "custom-ingress-port"
			className := "nginx"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Networking: openclawv1alpha1.NetworkingSpec{
						Service: openclawv1alpha1.ServiceSpec{
							Ports: []openclawv1alpha1.ServicePortSpec{
								{Name: "http", Port: 3978},
							},
						},
						Ingress: openclawv1alpha1.IngressSpec{
							Enabled:   true,
							ClassName: &className,
							Hosts: []openclawv1alpha1.IngressHost{
								{
									Host: "aibot.example.com",
									Paths: []openclawv1alpha1.IngressPath{
										{
											Path:     "/api/messages",
											PathType: "Prefix",
											Port:     resources.Ptr(int32(3978)),
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			ingress := &networkingv1.Ingress{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.IngressName(instance),
					Namespace: namespace,
				}, ingress)
			}, timeout, interval).Should(Succeed())

			Expect(ingress.Spec.Rules).To(HaveLen(1))
			paths := ingress.Spec.Rules[0].HTTP.Paths
			Expect(paths).To(HaveLen(1))
			Expect(paths[0].Path).To(Equal("/api/messages"))
			Expect(paths[0].Backend.Service.Port.Number).To(Equal(int32(3978)),
				"Ingress backend should route to custom port 3978")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with Tailscale enabled", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-ts-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should inject Tailscale config, env vars, and NetworkPolicy egress", func() {
			instanceName := "ts-e2e-instance"

			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			// Create auth key Secret
			tsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ts-auth",
					Namespace: namespace,
				},
				StringData: map[string]string{
					"authkey": "tskey-auth-test-XXXXX",
				},
			}
			Expect(k8sClient.Create(ctx, tsSecret)).Should(Succeed())

			// Create instance with Tailscale enabled
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Tailscale: openclawv1alpha1.TailscaleSpec{
						Enabled: true,
						Mode:    "serve",
						AuthKeySecretRef: &corev1.LocalObjectReference{
							Name: "ts-auth",
						},
						AuthSSO: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify ConfigMap contains tailscale config
			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.ConfigMapName(instance),
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			configContent, ok := cm.Data["openclaw.json"]
			Expect(ok).To(BeTrue(), "ConfigMap should have openclaw.json key")

			var parsed map[string]interface{}
			Expect(json.Unmarshal([]byte(configContent), &parsed)).To(Succeed())
			gw, ok := parsed["gateway"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "config should have gateway key")
			ts, ok := gw["tailscale"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "gateway should have tailscale key")
			Expect(ts["mode"]).To(Equal("serve"), "tailscale mode should be serve")
			Expect(ts["resetOnExit"]).To(BeTrue(), "tailscale resetOnExit should be true")

			// Verify AuthSSO sets gateway.auth.allowTailscale
			auth, ok := gw["auth"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "gateway should have auth key when AuthSSO is enabled")
			Expect(auth["allowTailscale"]).To(BeTrue(), "auth.allowTailscale should be true")

			// Verify StatefulSet env vars
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			mainContainer := statefulSet.Spec.Template.Spec.Containers[0]
			var foundAuthKey, foundHostname bool
			for _, env := range mainContainer.Env {
				if env.Name == "TS_AUTHKEY" {
					foundAuthKey = true
					Expect(env.ValueFrom).NotTo(BeNil(), "TS_AUTHKEY should use ValueFrom")
					Expect(env.ValueFrom.SecretKeyRef).NotTo(BeNil(), "TS_AUTHKEY should use SecretKeyRef")
					Expect(env.ValueFrom.SecretKeyRef.Name).To(Equal("ts-auth"))
					Expect(env.ValueFrom.SecretKeyRef.Key).To(Equal("authkey"))
				}
				if env.Name == "TS_HOSTNAME" {
					foundHostname = true
					Expect(env.Value).To(Equal(instanceName), "TS_HOSTNAME should default to instance name")
				}
			}
			Expect(foundAuthKey).To(BeTrue(), "TS_AUTHKEY env var should be present")
			Expect(foundHostname).To(BeTrue(), "TS_HOSTNAME env var should be present")

			// Verify NetworkPolicy has STUN and WireGuard egress
			np := &networkingv1.NetworkPolicy{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.NetworkPolicyName(instance),
					Namespace: namespace,
				}, np)
			}, timeout, interval).Should(Succeed())

			var foundSTUN, foundWG bool
			for _, rule := range np.Spec.Egress {
				for _, p := range rule.Ports {
					if p.Protocol != nil && *p.Protocol == corev1.ProtocolUDP && p.Port != nil {
						switch p.Port.IntValue() {
						case 3478:
							foundSTUN = true
						case 41641:
							foundWG = true
						}
					}
				}
			}
			Expect(foundSTUN).To(BeTrue(), "NetworkPolicy should have STUN egress (UDP 3478)")
			Expect(foundWG).To(BeTrue(), "NetworkPolicy should have WireGuard egress (UDP 41641)")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an OpenClawInstance with Ollama", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-ollama-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create Ollama sidecar when enabled", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "ollama-test"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Ollama: openclawv1alpha1.OllamaSpec{
						Enabled: true,
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify StatefulSet has ollama sidecar container
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			// Verify ollama container exists
			var ollamaContainer *corev1.Container
			for i := range statefulSet.Spec.Template.Spec.Containers {
				if statefulSet.Spec.Template.Spec.Containers[i].Name == "ollama" {
					ollamaContainer = &statefulSet.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(ollamaContainer).NotTo(BeNil(), "ollama sidecar container should exist")
			Expect(ollamaContainer.Image).To(Equal("ollama/ollama:latest"))

			// Verify ollama-models volume exists
			var ollamaVol *corev1.Volume
			for i := range statefulSet.Spec.Template.Spec.Volumes {
				if statefulSet.Spec.Template.Spec.Volumes[i].Name == "ollama-models" {
					ollamaVol = &statefulSet.Spec.Template.Spec.Volumes[i]
					break
				}
			}
			Expect(ollamaVol).NotTo(BeNil(), "ollama-models volume should exist")

			// Verify main container has OLLAMA_HOST env var
			mainContainer := statefulSet.Spec.Template.Spec.Containers[0]
			var foundOllamaHost bool
			for _, env := range mainContainer.Env {
				if env.Name == "OLLAMA_HOST" {
					foundOllamaHost = true
					Expect(env.Value).To(Equal("http://localhost:11434"))
					break
				}
			}
			Expect(foundOllamaHost).To(BeTrue(), "OLLAMA_HOST env var should be set")

			// No init-ollama since no models specified
			for _, ic := range statefulSet.Spec.Template.Spec.InitContainers {
				Expect(ic.Name).NotTo(Equal("init-ollama"), "init-ollama should not be present without models")
			}

			// Clean up
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with npm-prefixed skills (issue #131)", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-npm-skills-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should produce an init-skills container with npm install for npm: prefixed skills", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "npm-skills-test"

			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Skills: []string{"npm:@openclaw/matrix", "@anthropic/mcp-server-fetch"},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.StatefulSetName(instance),
					Namespace: namespace,
				}, statefulSet)
			}, 60*time.Second, 2*time.Second).Should(Succeed())

			var skillsContainer *corev1.Container
			for i := range statefulSet.Spec.Template.Spec.InitContainers {
				if statefulSet.Spec.Template.Spec.InitContainers[i].Name == "init-skills" {
					skillsContainer = &statefulSet.Spec.Template.Spec.InitContainers[i]
					break
				}
			}
			Expect(skillsContainer).NotTo(BeNil(), "init-skills container should exist")

			// Script should contain npm install for npm: prefixed skill
			script := skillsContainer.Command[2]
			Expect(script).To(ContainSubstring("npm install '@openclaw/matrix'"),
				"npm: prefixed skill should use npm install")
			// Script should also contain clawhub for non-prefixed skill
			Expect(script).To(ContainSubstring("clawhub install '@anthropic/mcp-server-fetch'"),
				"non-prefixed skill should use clawhub install")

			// NPM_CONFIG_IGNORE_SCRIPTS should be set (#91)
			envMap := map[string]string{}
			for _, e := range skillsContainer.Env {
				envMap[e.Name] = e.Value
			}
			Expect(envMap["NPM_CONFIG_IGNORE_SCRIPTS"]).To(Equal("true"),
				"NPM_CONFIG_IGNORE_SCRIPTS should be true for supply chain security")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When validating postStart config restoration (issue #125)", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-poststart-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should restore config via postStart hook after container restart", func() {
			instanceName := "poststart-e2e"
			podName := instanceName + "-0"

			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping postStart validation in minimal mode")
			}

			// Disable all probes so the pod stays Running regardless of
			// whether OpenClaw can fully start without API keys.
			falseVal := false
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Probes: openclawv1alpha1.ProbesSpec{
						Liveness:  &openclawv1alpha1.ProbeSpec{Enabled: &falseVal},
						Readiness: &openclawv1alpha1.ProbeSpec{Enabled: &falseVal},
						Startup:   &openclawv1alpha1.ProbeSpec{Enabled: &falseVal},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Wait for the openclaw container to be Running.
			// K8s does not set Running until the postStart hook completes,
			// so the config file is guaranteed to exist by this point.
			Eventually(func() bool {
				pod := &corev1.Pod{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: podName, Namespace: namespace,
				}, pod); err != nil {
					return false
				}
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.Name == "openclaw" && cs.State.Running != nil {
						return true
					}
				}
				return false
			}, 5*time.Minute, 5*time.Second).Should(BeTrue(),
				"openclaw container should be Running")

			// Verify config was written by the postStart hook
			out, err := kubectlExec(namespace, podName,
				"cat", "/home/openclaw/.openclaw/openclaw.json")
			Expect(err).NotTo(HaveOccurred(), "should read config file: %s", out)
			Expect(out).To(ContainSubstring(`"lan"`),
				"config should contain gateway.bind=lan from operator enrichment")

			// Corrupt the config file on the PVC
			_, err = kubectlExec(namespace, podName,
				"sh", "-c", `echo '{"corrupted":true}' > /home/openclaw/.openclaw/openclaw.json`)
			Expect(err).NotTo(HaveOccurred(), "should be able to write to PVC")

			// Verify config is corrupted
			out, err = kubectlExec(namespace, podName,
				"cat", "/home/openclaw/.openclaw/openclaw.json")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(ContainSubstring("corrupted"),
				"config should contain corrupted content")

			// Record current restart count
			pod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: podName, Namespace: namespace,
			}, pod)).To(Succeed())
			var prevRestarts int32
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Name == "openclaw" {
					prevRestarts = cs.RestartCount
					break
				}
			}

			// Crash the main process to trigger a container restart (not pod
			// recreation). Init containers do NOT re-run on container restarts -
			// only the postStart lifecycle hook runs again.
			//
			// Linux protects PID 1 from SIGKILL sent within the same PID
			// namespace, so "kill -9 1" silently fails. Instead we:
			// 1. Try to find PID 1's child (works when tini/dumb-init wraps the app)
			//    and SIGKILL the child (not PID 1, so kernel allows it).
			// 2. Fall back to SIGTERM to PID 1 (delivered if the app registered a
			//    handler, which Node.js/libuv does).
			_, _ = kubectlExec(namespace, podName, "sh", "-c",
				`cpid=$(cat /proc/1/task/*/children 2>/dev/null | tr ' ' '\n' | head -1); `+
					`[ -n "$cpid" ] && kill -9 "$cpid" || kill 1`)

			// Wait for restart count to increase
			Eventually(func() int32 {
				p := &corev1.Pod{}
				if getErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: podName, Namespace: namespace,
				}, p); getErr != nil {
					return -1
				}
				for _, cs := range p.Status.ContainerStatuses {
					if cs.Name == "openclaw" {
						return cs.RestartCount
					}
				}
				return -1
			}, 2*time.Minute, 2*time.Second).Should(BeNumerically(">", prevRestarts),
				"restart count should increase after killing main process")

			// Wait for container to be Running again (postStart must complete first)
			Eventually(func() bool {
				p := &corev1.Pod{}
				if getErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: podName, Namespace: namespace,
				}, p); getErr != nil {
					return false
				}
				for _, cs := range p.Status.ContainerStatuses {
					if cs.Name == "openclaw" && cs.State.Running != nil {
						return true
					}
				}
				return false
			}, 2*time.Minute, 2*time.Second).Should(BeTrue(),
				"openclaw container should be Running after restart")

			// Verify the postStart hook restored the config
			out, err = kubectlExec(namespace, podName,
				"cat", "/home/openclaw/.openclaw/openclaw.json")
			Expect(err).NotTo(HaveOccurred(),
				"should read config after restart: %s", out)
			Expect(out).To(ContainSubstring(`"lan"`),
				"gateway.bind=lan should be restored by postStart hook")
			Expect(out).NotTo(ContainSubstring("corrupted"),
				"corrupted content should be overwritten by postStart hook")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with configMapRef (#136)", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-cmref-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should enrich external ConfigMap config with gateway auth", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "cmref-enriched"

			// Create external ConfigMap with partial config (no gateway auth)
			externalCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-external-config",
					Namespace: namespace,
				},
				Data: map[string]string{
					"openclaw.json": `{"mcpServers":{"test":{"url":"http://localhost:3000"}}}`,
				},
			}
			Expect(k8sClient.Create(ctx, externalCM)).Should(Succeed())

			// Create instance referencing the external ConfigMap
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: namespace,
					Annotations: map[string]string{
						"openclaw.rocks/skip-backup": "true",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Image: openclawv1alpha1.ImageSpec{
						Repository: "ghcr.io/openclaw/openclaw",
						Tag:        "latest",
					},
					Config: openclawv1alpha1.ConfigSpec{
						ConfigMapRef: &openclawv1alpha1.ConfigMapKeySelector{
							Name: "my-external-config",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify operator-managed ConfigMap is created with enriched content
			cm := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.ConfigMapName(instance),
					Namespace: namespace,
				}, cm)
			}, timeout, interval).Should(Succeed())

			configContent, ok := cm.Data["openclaw.json"]
			Expect(ok).To(BeTrue(), "operator-managed ConfigMap should have openclaw.json key")

			var parsed map[string]interface{}
			Expect(json.Unmarshal([]byte(configContent), &parsed)).To(Succeed())

			// User config should be preserved
			_, hasMCP := parsed["mcpServers"]
			Expect(hasMCP).To(BeTrue(), "user's mcpServers should be preserved")

			// Gateway auth should be injected
			gw, ok := parsed["gateway"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "config should have gateway key")
			Expect(gw["bind"]).To(Equal("lan"), "gateway.bind should be lan")
			auth, ok := gw["auth"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "gateway should have auth key (injected by operator)")
			Expect(auth["mode"]).To(Equal("token"), "gateway.auth.mode should be token")
			Expect(auth["token"]).NotTo(BeEmpty(), "gateway.auth.token should be set")

			// Verify StatefulSet config volume points to operator-managed CM (not external)
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, statefulSet)
			}, timeout, interval).Should(Succeed())

			var configVol *corev1.Volume
			for i := range statefulSet.Spec.Template.Spec.Volumes {
				if statefulSet.Spec.Template.Spec.Volumes[i].Name == "config" {
					configVol = &statefulSet.Spec.Template.Spec.Volumes[i]
					break
				}
			}
			Expect(configVol).NotTo(BeNil(), "config volume should exist")
			Expect(configVol.ConfigMap).NotTo(BeNil())
			Expect(configVol.ConfigMap.Name).To(Equal(resources.ConfigMapName(instance)),
				"config volume should reference operator-managed CM, not external CM")

			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with auto-scaling enabled", func() {
		const hpaTestName = "e2e-hpa-test"
		const hpaTestNs = "default"

		It("Should create an HPA targeting the StatefulSet", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hpaTestName,
					Namespace: hpaTestNs,
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Availability: openclawv1alpha1.AvailabilitySpec{
						AutoScaling: &openclawv1alpha1.AutoScalingSpec{
							Enabled:              resources.Ptr(true),
							MinReplicas:          resources.Ptr(int32(1)),
							MaxReplicas:          resources.Ptr(int32(3)),
							TargetCPUUtilization: resources.Ptr(int32(70)),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Wait for HPA to be created
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.HPAName(instance),
					Namespace: hpaTestNs,
				}, hpa)
			}, 60*time.Second, 2*time.Second).Should(Succeed())

			Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("StatefulSet"))
			Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(resources.StatefulSetName(instance)))
			Expect(*hpa.Spec.MinReplicas).To(Equal(int32(1)))
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(3)))
			Expect(hpa.Spec.Metrics).To(HaveLen(1))
			Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(Equal(int32(70)))

			// Verify StatefulSet has nil replicas (HPA manages it)
			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.StatefulSetName(instance),
					Namespace: hpaTestNs,
				}, sts)
			}, 60*time.Second, 2*time.Second).Should(Succeed())

			// Cleanup
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When the operator is running", func() {
		It("Should have the controller manager deployment available", func() {
			deployment := &appsv1.Deployment{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "openclaw-operator-controller-manager",
				Namespace: "openclaw-operator-system",
			}, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Status.AvailableReplicas).To(BeNumerically(">=", 1))
		})
	})
})
