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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
