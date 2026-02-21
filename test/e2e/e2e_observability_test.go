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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
	"github.com/openclawrocks/k8s-operator/internal/resources"
)

// prometheusRuleCRDAvailable checks if the PrometheusRule CRD is installed in the cluster.
func prometheusRuleCRDAvailable() bool {
	pr := &unstructured.Unstructured{}
	pr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PrometheusRule",
	})
	err := k8sClient.List(ctx, &unstructured.UnstructuredList{Object: pr.Object})
	return !meta.IsNoMatchError(err)
}

var _ = Describe("Observability - Deep Insights", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Second * 1
	)

	Context("When creating an instance with PrometheusRule enabled", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-prom-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create and cleanup PrometheusRule", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}
			if !prometheusRuleCRDAvailable() {
				Skip("PrometheusRule CRD not installed (prometheus-operator required)")
			}

			instanceName := "prom-rule-test"
			trueVal := true

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
					Observability: openclawv1alpha1.ObservabilitySpec{
						Metrics: openclawv1alpha1.MetricsSpec{
							PrometheusRule: &openclawv1alpha1.PrometheusRuleSpec{
								Enabled: &trueVal,
								Labels: map[string]string{
									"release": "kube-prometheus-stack",
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify PrometheusRule is created
			pr := &unstructured.Unstructured{}
			pr.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "monitoring.coreos.com",
				Version: "v1",
				Kind:    "PrometheusRule",
			})
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.PrometheusRuleName(instance),
					Namespace: namespace,
				}, pr)
			}, timeout, interval).Should(Succeed())

			// Verify custom labels
			labels := pr.GetLabels()
			Expect(labels["release"]).To(Equal("kube-prometheus-stack"))

			// Verify owner reference
			ownerRefs := pr.GetOwnerReferences()
			Expect(ownerRefs).To(HaveLen(1))
			Expect(ownerRefs[0].Name).To(Equal(instanceName))

			// Clean up
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			// Verify PrometheusRule is deleted via garbage collection
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.PrometheusRuleName(instance),
					Namespace: namespace,
				}, pr)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When creating an instance with Grafana dashboards enabled", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-grafana-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create and cleanup Grafana dashboard ConfigMaps", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "grafana-dash-test"
			trueVal := true

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
					Observability: openclawv1alpha1.ObservabilitySpec{
						Metrics: openclawv1alpha1.MetricsSpec{
							GrafanaDashboard: &openclawv1alpha1.GrafanaDashboardSpec{
								Enabled: &trueVal,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify operator dashboard ConfigMap is created
			opCM := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.GrafanaDashboardOperatorName(instance),
					Namespace: namespace,
				}, opCM)
			}, timeout, interval).Should(Succeed())

			// Verify grafana_dashboard label
			Expect(opCM.Labels["grafana_dashboard"]).To(Equal("1"))
			// Verify grafana_folder annotation
			Expect(opCM.Annotations["grafana_folder"]).To(Equal("OpenClaw"))
			// Verify dashboard data key exists
			Expect(opCM.Data).To(HaveKey("openclaw-operator.json"))

			// Verify instance dashboard ConfigMap is created
			instCM := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.GrafanaDashboardInstanceName(instance),
					Namespace: namespace,
				}, instCM)
			}, timeout, interval).Should(Succeed())

			Expect(instCM.Labels["grafana_dashboard"]).To(Equal("1"))
			Expect(instCM.Data).To(HaveKey("openclaw-instance.json"))

			// Clean up
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			// Verify ConfigMaps are deleted via garbage collection
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.GrafanaDashboardOperatorName(instance),
					Namespace: namespace,
				}, opCM)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When disabling PrometheusRule on an existing instance", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-prom-cleanup-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should delete PrometheusRule when disabled", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}
			if !prometheusRuleCRDAvailable() {
				Skip("PrometheusRule CRD not installed (prometheus-operator required)")
			}

			instanceName := "prom-cleanup-test"
			trueVal := true

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
					Observability: openclawv1alpha1.ObservabilitySpec{
						Metrics: openclawv1alpha1.MetricsSpec{
							PrometheusRule: &openclawv1alpha1.PrometheusRuleSpec{
								Enabled: &trueVal,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Wait for PrometheusRule to exist
			pr := &unstructured.Unstructured{}
			pr.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "monitoring.coreos.com",
				Version: "v1",
				Kind:    "PrometheusRule",
			})
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.PrometheusRuleName(instance),
					Namespace: namespace,
				}, pr)
			}, timeout, interval).Should(Succeed())

			// Disable PrometheusRule
			updatedInstance := &openclawv1alpha1.OpenClawInstance{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      instanceName,
				Namespace: namespace,
			}, updatedInstance)).Should(Succeed())

			falseVal := false
			updatedInstance.Spec.Observability.Metrics.PrometheusRule.Enabled = &falseVal
			Expect(k8sClient.Update(ctx, updatedInstance)).Should(Succeed())

			// Verify PrometheusRule is deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resources.PrometheusRuleName(instance),
					Namespace: namespace,
				}, pr)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, updatedInstance)).Should(Succeed())
		})
	})
})
