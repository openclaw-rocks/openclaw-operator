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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

var _ = Describe("Periodic Backup CronJob", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Second * 1
	)

	Context("When creating an instance with spec.backup.schedule", func() {
		var namespace string

		BeforeEach(func() {
			namespace = "test-backup-" + time.Now().Format("20060102150405")
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create the s3-backup-credentials Secret in the operator namespace.
			// The operator reads credentials from its own namespace, so we create
			// it in the namespace where the operator runs (set via OPERATOR_NAMESPACE
			// env or defaulting to "openclaw-operator-system").
			operatorNS := os.Getenv("OPERATOR_NAMESPACE")
			if operatorNS == "" {
				operatorNS = "openclaw-operator-system"
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s3-backup-credentials",
					Namespace: operatorNS,
				},
				StringData: map[string]string{
					"S3_BUCKET":            "test-bucket",
					"S3_ACCESS_KEY_ID":     "test-key",
					"S3_SECRET_ACCESS_KEY": "test-secret",
					"S3_ENDPOINT":          "https://s3.example.com",
				},
			}
			err := k8sClient.Create(ctx, secret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			// Clean up the S3 credentials Secret so other tests that expect
			// "no S3 credentials" are not affected by our setup.
			operatorNS := os.Getenv("OPERATOR_NAMESPACE")
			if operatorNS == "" {
				operatorNS = "openclaw-operator-system"
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s3-backup-credentials",
					Namespace: operatorNS,
				},
			}
			_ = k8sClient.Delete(ctx, secret)

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("Should create a CronJob when schedule is set", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "backup-cron-test"
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
					Backup: openclawv1alpha1.BackupSpec{
						Schedule: "0 2 * * *",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Verify CronJob is created
			cronJob := &batchv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName + "-backup-periodic",
					Namespace: namespace,
				}, cronJob)
			}, timeout, interval).Should(Succeed())

			Expect(cronJob.Spec.Schedule).To(Equal("0 2 * * *"))
			Expect(cronJob.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent))

			// Verify ScheduledBackupReady condition
			Eventually(func() bool {
				updatedInstance := &openclawv1alpha1.OpenClawInstance{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName,
					Namespace: namespace,
				}, updatedInstance); err != nil {
					return false
				}
				for _, c := range updatedInstance.Status.Conditions {
					if c.Type == openclawv1alpha1.ConditionTypeScheduledBackupReady && c.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			// Clean up
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			// Verify CronJob is deleted via owner reference
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName + "-backup-periodic",
					Namespace: namespace,
				}, cronJob)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("Should delete CronJob when schedule is removed", func() {
			if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
				Skip("Skipping resource validation in minimal mode")
			}

			instanceName := "backup-cron-remove"
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
					Backup: openclawv1alpha1.BackupSpec{
						Schedule: "0 3 * * *",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			// Wait for CronJob to be created
			cronJob := &batchv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName + "-backup-periodic",
					Namespace: namespace,
				}, cronJob)
			}, timeout, interval).Should(Succeed())

			// Remove the schedule
			updatedInstance := &openclawv1alpha1.OpenClawInstance{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      instanceName,
				Namespace: namespace,
			}, updatedInstance)).Should(Succeed())
			updatedInstance.Spec.Backup.Schedule = ""
			Expect(k8sClient.Update(ctx, updatedInstance)).Should(Succeed())

			// Verify CronJob is deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      instanceName + "-backup-periodic",
					Namespace: namespace,
				}, cronJob)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			// Clean up
			Expect(k8sClient.Delete(ctx, updatedInstance)).Should(Succeed())
		})
	})
})
