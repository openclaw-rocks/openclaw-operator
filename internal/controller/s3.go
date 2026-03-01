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
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
	"github.com/openclawrocks/k8s-operator/internal/resources"
)

const (
	// BackupSecretName is the name of the Secret containing S3 credentials
	BackupSecretName = "s3-backup-credentials" // #nosec G101 -- not a credential, just a Secret resource name

	// RcloneImage is the pinned rclone container image
	RcloneImage = "rclone/rclone:1.68"

	// AnnotationSkipBackup allows skipping backup on delete
	AnnotationSkipBackup = "openclaw.rocks/skip-backup"

	// LabelTenant is the label key for the tenant ID
	LabelTenant = "openclaw.rocks/tenant"

	// LabelInstance is the label key for the instance ID
	LabelInstance = "openclaw.rocks/instance"

	// LabelManagedBy is the label key for the manager
	LabelManagedBy = "app.kubernetes.io/managed-by"
)

// s3Credentials holds the S3 credential values read from a Secret
type s3Credentials struct {
	Bucket   string
	KeyID    string
	AppKey   string
	Endpoint string
	Region   string // optional - only needed for S3 providers with custom regions (e.g., MinIO)
}

// getTenantID extracts the tenant ID from the instance label or falls back to namespace
func getTenantID(instance *openclawv1alpha1.OpenClawInstance) string {
	if tenant, ok := instance.Labels[LabelTenant]; ok && tenant != "" {
		return tenant
	}
	// Fallback: extract from namespace (oc-tenant-{id} -> {id})
	ns := instance.Namespace
	if strings.HasPrefix(ns, "oc-tenant-") {
		return strings.TrimPrefix(ns, "oc-tenant-")
	}
	return ns
}

// getS3Credentials reads the S3 backup credentials Secret from the operator namespace
func (r *OpenClawInstanceReconciler) getS3Credentials(ctx context.Context) (*s3Credentials, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      BackupSecretName,
		Namespace: r.OperatorNamespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("failed to get S3 credentials secret %s/%s: %w", r.OperatorNamespace, BackupSecretName, err)
	}

	get := func(key string) (string, error) {
		v, ok := secret.Data[key]
		if !ok || len(v) == 0 {
			return "", fmt.Errorf("S3 credentials secret missing key %q", key)
		}
		return string(v), nil
	}

	bucket, err := get("S3_BUCKET")
	if err != nil {
		return nil, err
	}
	keyID, err := get("S3_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}
	appKey, err := get("S3_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	endpoint, err := get("S3_ENDPOINT")
	if err != nil {
		return nil, err
	}

	// S3_REGION is optional - only needed for providers with custom regions (e.g., MinIO)
	region := string(secret.Data["S3_REGION"])

	return &s3Credentials{
		Bucket:   bucket,
		KeyID:    keyID,
		AppKey:   appKey,
		Endpoint: endpoint,
		Region:   region,
	}, nil
}

// buildRcloneJob creates a batch/v1 Job that runs rclone to sync data between a PVC and S3.
// For backup: src=PVC mount, dst=S3 remote path
// For restore: src=S3 remote path, dst=PVC mount
func buildRcloneJob(
	name, namespace, pvcName string,
	remotePath string,
	labels map[string]string,
	creds *s3Credentials,
	isBackup bool,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
) *batchv1.Job {
	backoffLimit := int32(3)
	ttl := int32(86400) // 24h

	// rclone remote config via env vars
	// :s3: is used because S3-compatible API works with rclone's S3 backend
	rcloneRemotePath := fmt.Sprintf(":s3:%s/%s", creds.Bucket, remotePath)

	var args []string
	if isBackup {
		// PVC -> S3
		args = []string{"sync", "/data/", rcloneRemotePath, "--s3-provider=Other", "--s3-endpoint=$(S3_ENDPOINT)", "--s3-access-key-id=$(S3_ACCESS_KEY_ID)", "--s3-secret-access-key=$(S3_SECRET_ACCESS_KEY)", "--transfers=8", "--checkers=16", "-v"}
	} else {
		// S3 -> PVC
		args = []string{"sync", rcloneRemotePath, "/data/", "--s3-provider=Other", "--s3-endpoint=$(S3_ENDPOINT)", "--s3-access-key-id=$(S3_ACCESS_KEY_ID)", "--s3-secret-access-key=$(S3_SECRET_ACCESS_KEY)", "--transfers=8", "--checkers=16", "-v"}
	}

	if creds.Region != "" {
		args = append(args, "--s3-region=$(S3_REGION)")
	}

	env := []corev1.EnvVar{
		{Name: "S3_ENDPOINT", Value: creds.Endpoint},
		{Name: "S3_ACCESS_KEY_ID", Value: creds.KeyID},
		{Name: "S3_SECRET_ACCESS_KEY", Value: creds.AppKey},
	}
	if creds.Region != "" {
		env = append(env, corev1.EnvVar{Name: "S3_REGION", Value: creds.Region})
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					NodeSelector:  nodeSelector,
					Tolerations:   tolerations,
					// Match the fsGroup/runAsUser from the OpenClaw StatefulSet
					// so the rclone container can read/write the PVC data
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  int64Ptr(1000),
						RunAsGroup: int64Ptr(1000),
						FSGroup:    int64Ptr(1000),
					},
					Containers: []corev1.Container{
						{
							Name:    "rclone",
							Image:   RcloneImage,
							Command: []string{"rclone"},
							Args:    args,
							Env:     env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

// backupJobName returns a deterministic name for the backup Job
func backupJobName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-backup"
}

// restoreJobName returns a deterministic name for the restore Job
func restoreJobName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-restore"
}

// backupLabels returns labels for a backup/restore Job
func backupLabels(instance *openclawv1alpha1.OpenClawInstance, jobType string) map[string]string {
	return map[string]string{
		LabelManagedBy:            "openclaw-operator",
		LabelTenant:               getTenantID(instance),
		LabelInstance:             instance.Name,
		"openclaw.rocks/job-type": jobType,
	}
}

// isJobFinished checks whether the given Job has completed or failed
func isJobFinished(job *batchv1.Job) (bool, batchv1.JobConditionType) {
	for _, c := range job.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			return true, c.Type
		}
	}
	return false, ""
}

// pvcName returns the PVC name for the instance (delegates to resources package)
func pvcNameForInstance(instance *openclawv1alpha1.OpenClawInstance) string {
	return resources.PVCName(instance)
}

// getJob fetches a Job by name and namespace, returns nil if not found
func (r *OpenClawInstanceReconciler) getJob(ctx context.Context, name, namespace string) (*batchv1.Job, error) {
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// backupCronJobName returns a deterministic name for the periodic backup CronJob
func backupCronJobName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-backup-periodic"
}

// buildBackupCronJob creates a batch/v1 CronJob for periodic S3 backups.
// The CronJob mounts the PVC read-only and uses pod affinity to co-locate
// on the same node as the StatefulSet pod (required for RWO PVCs).
func buildBackupCronJob(
	instance *openclawv1alpha1.OpenClawInstance,
	creds *s3Credentials,
) *batchv1.CronJob {
	name := backupCronJobName(instance)
	labels := backupLabels(instance, "periodic-backup")
	pvcName := pvcNameForInstance(instance)
	tenantID := getTenantID(instance)

	historyLimit := int32(3)
	if instance.Spec.Backup.HistoryLimit != nil {
		historyLimit = *instance.Spec.Backup.HistoryLimit
	}
	failedHistoryLimit := int32(1)
	if instance.Spec.Backup.FailedHistoryLimit != nil {
		failedHistoryLimit = *instance.Spec.Backup.FailedHistoryLimit
	}

	backoffLimit := int32(3)
	ttl := int32(86400) // 24h
	gracePeriod := int64(30)

	// Shell command: compute timestamped S3 path and run rclone sync
	// Uses $(date) for unique path per run under periodic/ prefix
	rcloneCmd := fmt.Sprintf(
		`TIMESTAMP=$(date -u +%%Y%%m%%dT%%H%%M%%SZ) && `+
			`rclone sync /data/ ":s3:%s/backups/%s/%s/periodic/${TIMESTAMP}" `+
			`--s3-provider=Other `+
			`--s3-endpoint="${S3_ENDPOINT}" `+
			`--s3-access-key-id="${S3_ACCESS_KEY_ID}" `+
			`--s3-secret-access-key="${S3_SECRET_ACCESS_KEY}" `+
			`--transfers=8 --checkers=16 -v`,
		creds.Bucket, tenantID, instance.Name,
	)

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   instance.Spec.Backup.Schedule,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &historyLimit,
			FailedJobsHistoryLimit:     &failedHistoryLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit:            &backoffLimit,
					TTLSecondsAfterFinished: &ttl,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							RestartPolicy:                 corev1.RestartPolicyOnFailure,
							DNSPolicy:                     corev1.DNSClusterFirst,
							SchedulerName:                 "default-scheduler",
							TerminationGracePeriodSeconds: &gracePeriod,
							SecurityContext: &corev1.PodSecurityContext{
								RunAsUser:  int64Ptr(1000),
								RunAsGroup: int64Ptr(1000),
								FSGroup:    int64Ptr(1000),
							},
							// Pod affinity: require scheduling on the same node as the
							// StatefulSet pod so the RWO PVC can be mounted read-only.
							Affinity: &corev1.Affinity{
								PodAffinity: &corev1.PodAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"app.kubernetes.io/name":     "openclaw",
													"app.kubernetes.io/instance": instance.Name,
												},
											},
											TopologyKey: "kubernetes.io/hostname",
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:            "rclone",
									Image:           RcloneImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									Command:         []string{"sh", "-c", rcloneCmd},
									Env: []corev1.EnvVar{
										{Name: "S3_ENDPOINT", Value: creds.Endpoint},
										{Name: "S3_ACCESS_KEY_ID", Value: creds.KeyID},
										{Name: "S3_SECRET_ACCESS_KEY", Value: creds.AppKey},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "data",
											MountPath: "/data",
											ReadOnly:  true,
										},
									},
									TerminationMessagePath:   "/dev/termination-log",
									TerminationMessagePolicy: corev1.TerminationMessageReadFile,
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "data",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: pvcName,
											ReadOnly:  true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// reconcileBackupCronJob creates or deletes the periodic backup CronJob based on spec.backup.schedule.
func (r *OpenClawInstanceReconciler) reconcileBackupCronJob(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) error {
	logger := log.FromContext(ctx)

	// If no schedule is set, delete any existing CronJob and clear condition
	if instance.Spec.Backup.Schedule == "" {
		return r.cleanupBackupCronJob(ctx, instance)
	}

	// Check persistence is enabled
	if instance.Spec.Storage.Persistence.Enabled != nil && !*instance.Spec.Storage.Persistence.Enabled {
		logger.Info("Scheduled backup requested but persistence is disabled, skipping CronJob creation")
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               openclawv1alpha1.ConditionTypeScheduledBackupReady,
			Status:             metav1.ConditionFalse,
			Reason:             "PersistenceDisabled",
			Message:            "Periodic backups require persistence to be enabled",
			ObservedGeneration: instance.Generation,
		})
		return r.cleanupBackupCronJob(ctx, instance)
	}

	// Get S3 credentials
	creds, err := r.getS3Credentials(ctx)
	if err != nil {
		logger.Info("Scheduled backup requested but S3 credentials not found, skipping CronJob creation", "error", err)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               openclawv1alpha1.ConditionTypeScheduledBackupReady,
			Status:             metav1.ConditionFalse,
			Reason:             "S3CredentialsMissing",
			Message:            "S3 credentials secret not found in operator namespace - create s3-backup-credentials Secret to enable periodic backups",
			ObservedGeneration: instance.Generation,
		})
		return nil
	}

	// Build desired CronJob
	desired := buildBackupCronJob(instance, creds)

	// CreateOrUpdate the CronJob
	obj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupCronJobName(instance),
			Namespace: instance.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Spec = desired.Spec
		return controllerutil.SetControllerReference(instance, obj, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to reconcile backup CronJob: %w", err)
	}

	instance.Status.ManagedResources.BackupCronJob = obj.Name

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               openclawv1alpha1.ConditionTypeScheduledBackupReady,
		Status:             metav1.ConditionTrue,
		Reason:             "CronJobReady",
		Message:            fmt.Sprintf("Periodic backup CronJob %q created with schedule %q", obj.Name, instance.Spec.Backup.Schedule),
		ObservedGeneration: instance.Generation,
	})

	logger.V(1).Info("Backup CronJob reconciled", "name", obj.Name, "schedule", instance.Spec.Backup.Schedule)
	return nil
}

// cleanupBackupCronJob deletes the backup CronJob if it exists and clears status.
func (r *OpenClawInstanceReconciler) cleanupBackupCronJob(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) error {
	cronJob := &batchv1.CronJob{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      backupCronJobName(instance),
		Namespace: instance.Namespace,
	}, cronJob)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Already gone, clear status
			instance.Status.ManagedResources.BackupCronJob = ""
			meta.RemoveStatusCondition(&instance.Status.Conditions, openclawv1alpha1.ConditionTypeScheduledBackupReady)
			return nil
		}
		return fmt.Errorf("failed to get backup CronJob for cleanup: %w", err)
	}

	if err := r.Delete(ctx, cronJob); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete backup CronJob: %w", err)
	}

	instance.Status.ManagedResources.BackupCronJob = ""
	meta.RemoveStatusCondition(&instance.Status.Conditions, openclawv1alpha1.ConditionTypeScheduledBackupReady)
	return nil
}
