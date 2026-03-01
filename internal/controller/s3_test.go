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

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

var _ = Describe("S3 Helpers", func() {
	Context("getTenantID", func() {
		It("Should return the tenant label value when present", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "oc-tenant-cus_123",
					Labels: map[string]string{
						LabelTenant: "cus_456",
					},
				},
			}
			Expect(getTenantID(instance)).To(Equal("cus_456"))
		})

		It("Should extract tenant from namespace when label is missing", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "oc-tenant-cus_789",
				},
			}
			Expect(getTenantID(instance)).To(Equal("cus_789"))
		})

		It("Should return namespace as-is when not in oc-tenant format", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}
			Expect(getTenantID(instance)).To(Equal("default"))
		})
	})

	Context("buildRcloneJob", func() {
		var creds *s3Credentials

		BeforeEach(func() {
			creds = &s3Credentials{
				Bucket:   "test-bucket",
				KeyID:    "key123",
				AppKey:   "secret456",
				Endpoint: "https://s3.us-west-000.backblazeb2.com",
			}
		})

		It("Should build a backup Job with correct args and SecurityContext", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-t1",
				},
			}
			labels := backupLabels(instance, "backup")
			job := buildRcloneJob("myinst-backup", "oc-tenant-t1", "myinst-data", "backups/t1/myinst/2026-01-01T000000Z", labels, creds, true, nil, nil)

			Expect(job.Name).To(Equal("myinst-backup"))
			Expect(job.Namespace).To(Equal("oc-tenant-t1"))
			Expect(*job.Spec.BackoffLimit).To(Equal(int32(3)))
			Expect(*job.Spec.TTLSecondsAfterFinished).To(Equal(int32(86400)))

			// Verify container
			container := job.Spec.Template.Spec.Containers[0]
			Expect(container.Image).To(Equal(RcloneImage))
			Expect(container.Args[0]).To(Equal("sync"))
			Expect(container.Args[1]).To(Equal("/data/")) // PVC source for backup

			// Verify SecurityContext
			podSC := job.Spec.Template.Spec.SecurityContext
			Expect(*podSC.RunAsUser).To(Equal(int64(1000)))
			Expect(*podSC.RunAsGroup).To(Equal(int64(1000)))
			Expect(*podSC.FSGroup).To(Equal(int64(1000)))

			// Verify PVC volume
			vol := job.Spec.Template.Spec.Volumes[0]
			Expect(vol.PersistentVolumeClaim.ClaimName).To(Equal("myinst-data"))

			// Verify env vars
			var envNames []string
			for _, e := range container.Env {
				envNames = append(envNames, e.Name)
			}
			Expect(envNames).To(ContainElements("S3_ENDPOINT", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY"))
			Expect(envNames).NotTo(ContainElement("S3_REGION"))

			// Verify no --s3-region flag when Region is empty
			for _, arg := range container.Args {
				Expect(arg).NotTo(HavePrefix("--s3-region"))
			}
		})

		It("Should build a restore Job with S3 as source", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-t1",
				},
			}
			labels := backupLabels(instance, "restore")
			job := buildRcloneJob("myinst-restore", "oc-tenant-t1", "myinst-data", "backups/t1/myinst/2026-01-01T000000Z", labels, creds, false, nil, nil)

			container := job.Spec.Template.Spec.Containers[0]
			Expect(container.Args[0]).To(Equal("sync"))
			// For restore, dest is /data/
			Expect(container.Args[2]).To(Equal("/data/"))

			vol := job.Spec.Template.Spec.Volumes[0]
			Expect(vol.PersistentVolumeClaim.ClaimName).To(Equal("myinst-data"))
		})

		It("Should propagate nodeSelector and tolerations to Job pod", func() {
			nodeSelector := map[string]string{"openclaw.rocks/nodepool": "openclaw"}
			tolerations := []corev1.Toleration{
				{
					Key:      "openclaw.rocks/dedicated",
					Operator: corev1.TolerationOpEqual,
					Value:    "openclaw",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-t1",
				},
			}
			labels := backupLabels(instance, "backup")
			job := buildRcloneJob("myinst-backup", "oc-tenant-t1", "myinst-data", "backups/t1/myinst/2026-01-01T000000Z", labels, creds, true, nodeSelector, tolerations)

			Expect(job.Spec.Template.Spec.NodeSelector).To(Equal(nodeSelector))
			Expect(job.Spec.Template.Spec.Tolerations).To(HaveLen(1))
			Expect(job.Spec.Template.Spec.Tolerations[0].Key).To(Equal("openclaw.rocks/dedicated"))
			Expect(job.Spec.Template.Spec.Tolerations[0].Value).To(Equal("openclaw"))
		})

		It("Should include --s3-region flag and S3_REGION env var when Region is set", func() {
			creds.Region = "eu-west-1"
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-t1",
				},
			}
			labels := backupLabels(instance, "backup")
			job := buildRcloneJob("myinst-backup", "oc-tenant-t1", "myinst-data", "backups/t1/myinst/2026-01-01T000000Z", labels, creds, true, nil, nil)

			container := job.Spec.Template.Spec.Containers[0]

			// Verify --s3-region flag is present
			Expect(container.Args).To(ContainElement("--s3-region=$(S3_REGION)"))

			// Verify S3_REGION env var is present with correct value
			var regionEnv *corev1.EnvVar
			for i, e := range container.Env {
				if e.Name == "S3_REGION" {
					regionEnv = &container.Env[i]
					break
				}
			}
			Expect(regionEnv).NotTo(BeNil())
			Expect(regionEnv.Value).To(Equal("eu-west-1"))
		})
	})

	Context("isJobFinished", func() {
		It("Should return false for an active Job", func() {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "default"},
			}
			finished, _ := isJobFinished(job)
			Expect(finished).To(BeFalse())
		})

		It("Should return true with Complete for a succeeded Job", func() {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "default"},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			finished, condType := isJobFinished(job)
			Expect(finished).To(BeTrue())
			Expect(condType).To(Equal(batchv1.JobComplete))
		})

		It("Should return true with Failed for a failed Job", func() {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "default"},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobFailed,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			finished, condType := isJobFinished(job)
			Expect(finished).To(BeTrue())
			Expect(condType).To(Equal(batchv1.JobFailed))
		})
	})

	Context("backupLabels", func() {
		It("Should include tenant, instance, and job-type labels", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-cus_123",
					Labels: map[string]string{
						LabelTenant: "cus_123",
					},
				},
			}
			labels := backupLabels(instance, "backup")
			Expect(labels[LabelTenant]).To(Equal("cus_123"))
			Expect(labels[LabelInstance]).To(Equal("myinst"))
			Expect(labels["openclaw.rocks/job-type"]).To(Equal("backup"))
			Expect(labels[LabelManagedBy]).To(Equal("openclaw-operator"))
		})
	})

	Context("backupCronJobName", func() {
		It("Should return instance name with -backup-periodic suffix", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "my-agent"},
			}
			Expect(backupCronJobName(instance)).To(Equal("my-agent-backup-periodic"))
		})
	})

	Context("buildBackupCronJob", func() {
		var creds *s3Credentials
		var instance *openclawv1alpha1.OpenClawInstance

		BeforeEach(func() {
			creds = &s3Credentials{
				Bucket:   "test-bucket",
				KeyID:    "key123",
				AppKey:   "secret456",
				Endpoint: "https://s3.us-west-000.backblazeb2.com",
			}
			instance = &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myinst",
					Namespace: "oc-tenant-t1",
					Labels: map[string]string{
						LabelTenant: "cus_123",
					},
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Backup: openclawv1alpha1.BackupSpec{
						Schedule: "0 2 * * *",
					},
				},
			}
		})

		It("Should set the cron schedule from spec", func() {
			cronJob := buildBackupCronJob(instance, creds)
			Expect(cronJob.Spec.Schedule).To(Equal("0 2 * * *"))
		})

		It("Should set ConcurrencyPolicy to Forbid", func() {
			cronJob := buildBackupCronJob(instance, creds)
			Expect(cronJob.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent))
		})

		It("Should use default history limits when not specified", func() {
			cronJob := buildBackupCronJob(instance, creds)
			Expect(*cronJob.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(3)))
			Expect(*cronJob.Spec.FailedJobsHistoryLimit).To(Equal(int32(1)))
		})

		It("Should use custom history limits when specified", func() {
			historyLimit := int32(5)
			failedLimit := int32(2)
			instance.Spec.Backup.HistoryLimit = &historyLimit
			instance.Spec.Backup.FailedHistoryLimit = &failedLimit

			cronJob := buildBackupCronJob(instance, creds)
			Expect(*cronJob.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(5)))
			Expect(*cronJob.Spec.FailedJobsHistoryLimit).To(Equal(int32(2)))
		})

		It("Should mount PVC read-only", func() {
			cronJob := buildBackupCronJob(instance, creds)
			container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
			Expect(container.VolumeMounts[0].ReadOnly).To(BeTrue())
			Expect(container.VolumeMounts[0].MountPath).To(Equal("/data"))

			vol := cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes[0]
			Expect(vol.PersistentVolumeClaim.ReadOnly).To(BeTrue())
		})

		It("Should set pod affinity for co-location with StatefulSet pod", func() {
			cronJob := buildBackupCronJob(instance, creds)
			affinity := cronJob.Spec.JobTemplate.Spec.Template.Spec.Affinity
			Expect(affinity).NotTo(BeNil())
			Expect(affinity.PodAffinity).NotTo(BeNil())
			Expect(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))

			term := affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0]
			Expect(term.TopologyKey).To(Equal("kubernetes.io/hostname"))
			Expect(term.LabelSelector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/name", "openclaw"))
			Expect(term.LabelSelector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/instance", "myinst"))
		})

		It("Should use shell command with timestamped S3 path under periodic/ prefix", func() {
			cronJob := buildBackupCronJob(instance, creds)
			container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
			Expect(container.Command).To(HaveLen(3))
			Expect(container.Command[0]).To(Equal("sh"))
			Expect(container.Command[1]).To(Equal("-c"))
			// Verify the shell script references periodic/ path with timestamp
			Expect(container.Command[2]).To(ContainSubstring("periodic/${TIMESTAMP}"))
			Expect(container.Command[2]).To(ContainSubstring("TIMESTAMP=$(date -u"))
			Expect(container.Command[2]).To(ContainSubstring("rclone sync /data/"))
			Expect(container.Command[2]).To(ContainSubstring(":s3:test-bucket/backups/cus_123/myinst/periodic/"))
		})

		It("Should set security context with UID/GID 1000", func() {
			cronJob := buildBackupCronJob(instance, creds)
			podSC := cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
			Expect(*podSC.RunAsUser).To(Equal(int64(1000)))
			Expect(*podSC.RunAsGroup).To(Equal(int64(1000)))
			Expect(*podSC.FSGroup).To(Equal(int64(1000)))
		})

		It("Should use rclone image and set S3 env vars", func() {
			cronJob := buildBackupCronJob(instance, creds)
			container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
			Expect(container.Image).To(Equal(RcloneImage))
			Expect(container.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))

			var envNames []string
			for _, e := range container.Env {
				envNames = append(envNames, e.Name)
			}
			Expect(envNames).To(ContainElements("S3_ENDPOINT", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY"))
		})

		It("Should set periodic-backup label", func() {
			cronJob := buildBackupCronJob(instance, creds)
			Expect(cronJob.Labels["openclaw.rocks/job-type"]).To(Equal("periodic-backup"))
		})

		It("Should set explicit Kubernetes default fields", func() {
			cronJob := buildBackupCronJob(instance, creds)
			spec := cronJob.Spec.JobTemplate.Spec.Template.Spec
			Expect(spec.RestartPolicy).To(Equal(corev1.RestartPolicyOnFailure))
			Expect(spec.DNSPolicy).To(Equal(corev1.DNSClusterFirst))
			Expect(spec.SchedulerName).To(Equal("default-scheduler"))
			Expect(spec.TerminationGracePeriodSeconds).NotTo(BeNil())

			container := spec.Containers[0]
			Expect(container.TerminationMessagePath).To(Equal("/dev/termination-log"))
			Expect(container.TerminationMessagePolicy).To(Equal(corev1.TerminationMessageReadFile))
		})
	})
})
