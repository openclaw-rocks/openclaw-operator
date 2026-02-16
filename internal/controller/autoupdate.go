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
	"time"

	"github.com/Masterminds/semver/v3"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
	"github.com/openclawrocks/k8s-operator/internal/resources"
)

const (
	updatePhaseBackingUp      = "BackingUp"
	updatePhaseApplyingUpdate = "ApplyingUpdate"
)

// resolveInitialTag resolves "latest" to a concrete semver tag when auto-update is enabled.
// Returns true if the tag was resolved (caller should requeue), false if no action was taken.
func (r *OpenClawInstanceReconciler) resolveInitialTag(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) (bool, error) {
	logger := log.FromContext(ctx)

	if !isAutoUpdateEnabled(instance) {
		return false, nil
	}
	if instance.Spec.Image.Tag != "latest" {
		return false, nil
	}
	if r.VersionResolver == nil {
		return false, nil
	}

	repo := resources.GetImageRepository(instance)
	version, err := r.VersionResolver.LatestSemver(ctx, repo, nil)
	if err != nil {
		logger.Error(err, "Failed to resolve initial tag from registry, continuing with 'latest'")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "AutoUpdateResolveFailed",
			fmt.Sprintf("Failed to resolve latest version: %v", err))
		return false, nil // non-fatal
	}

	logger.Info("Resolved initial tag from registry", "tag", version)

	// Patch spec.image.tag
	original := instance.DeepCopy()
	instance.Spec.Image.Tag = version
	if err := r.Patch(ctx, instance, client.MergeFrom(original)); err != nil {
		return false, fmt.Errorf("patching image tag: %w", err)
	}

	// Update status
	instance.Status.AutoUpdate.CurrentVersion = version
	if err := r.Status().Update(ctx, instance); err != nil {
		return false, fmt.Errorf("updating auto-update status: %w", err)
	}

	r.Recorder.Event(instance, corev1.EventTypeNormal, "AutoUpdateTagResolved",
		fmt.Sprintf("Resolved 'latest' to %s", version))

	return true, nil
}

// reconcileAutoUpdate handles the periodic version check and update state machine.
// It is called from Reconcile() after successful resource reconciliation.
func (r *OpenClawInstanceReconciler) reconcileAutoUpdate(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If an update is in progress, drive the state machine
	if instance.Status.AutoUpdate.PendingVersion != "" {
		return r.driveUpdateStateMachine(ctx, instance)
	}

	// Guard: skip if auto-update is not enabled or applicable
	if !isAutoUpdateEnabled(instance) {
		return ctrl.Result{}, nil
	}

	if r.VersionResolver == nil {
		return ctrl.Result{}, nil
	}

	// Check if it's time to check for updates
	if !shouldCheckForUpdate(instance) {
		return ctrl.Result{}, nil
	}

	// Perform version check
	repo := resources.GetImageRepository(instance)
	version, err := r.VersionResolver.LatestSemver(ctx, repo, nil)
	if err != nil {
		logger.Error(err, "Failed to check for updates")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "AutoUpdateCheckFailed",
			fmt.Sprintf("Failed to check for updates: %v", err))
		autoUpdateChecksTotal.WithLabelValues(instance.Name, instance.Namespace, "error").Inc()
		// Update last check time even on failure to avoid hammering the registry
		now := metav1.Now()
		instance.Status.AutoUpdate.LastCheckTime = &now
		return ctrl.Result{}, nil // non-fatal
	}

	now := metav1.Now()
	instance.Status.AutoUpdate.LastCheckTime = &now
	instance.Status.AutoUpdate.LatestVersion = version
	autoUpdateChecksTotal.WithLabelValues(instance.Name, instance.Namespace, "success").Inc()

	// Determine current version
	currentTag := instance.Spec.Image.Tag
	if currentTag == "" || currentTag == "latest" {
		// Existing instance still on "latest" — resolve to the concrete version now
		logger.Info("Resolving 'latest' tag to concrete version", "resolved", version)
		original := instance.DeepCopy()
		instance.Spec.Image.Tag = version
		if patchErr := r.Patch(ctx, instance, client.MergeFrom(original)); patchErr != nil {
			return ctrl.Result{}, fmt.Errorf("patching image tag from latest: %w", patchErr)
		}
		instance.Status.AutoUpdate.CurrentVersion = version
		r.Recorder.Event(instance, corev1.EventTypeNormal, "AutoUpdateTagResolved",
			fmt.Sprintf("Resolved 'latest' to %s", version))
		return ctrl.Result{Requeue: true}, nil
	}

	currentVer, err := semver.NewVersion(currentTag)
	if err != nil {
		logger.V(1).Info("Current tag is not a valid semver, skipping update check", "tag", currentTag)
		return ctrl.Result{}, nil
	}

	latestVer, err := semver.NewVersion(version)
	if err != nil {
		logger.V(1).Info("Latest version from registry is not valid semver", "version", version)
		return ctrl.Result{}, nil
	}

	instance.Status.AutoUpdate.CurrentVersion = currentTag

	if !latestVer.GreaterThan(currentVer) {
		logger.V(1).Info("No new version available", "current", currentTag, "latest", version)
		// Clear any stale AutoUpdateAvailable condition
		meta.RemoveStatusCondition(&instance.Status.Conditions, openclawv1alpha1.ConditionTypeAutoUpdateAvailable)
		return ctrl.Result{}, nil
	}

	// New version available — start update
	logger.Info("New version available, starting update", "current", currentTag, "latest", version)
	r.Recorder.Event(instance, corev1.EventTypeNormal, "AutoUpdateAvailable",
		fmt.Sprintf("New version %s available (current: %s)", version, currentTag))

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               openclawv1alpha1.ConditionTypeAutoUpdateAvailable,
		Status:             metav1.ConditionTrue,
		Reason:             "NewVersionAvailable",
		Message:            fmt.Sprintf("Version %s is available (current: %s)", version, currentTag),
		LastTransitionTime: metav1.Now(),
	})

	instance.Status.AutoUpdate.PendingVersion = version
	instance.Status.AutoUpdate.UpdatePhase = ""
	instance.Status.AutoUpdate.LastUpdateError = ""

	return ctrl.Result{Requeue: true}, nil
}

// driveUpdateStateMachine manages the multi-step update process.
//
// State machine:
//  1. Set phase = Updating
//  2. If backupBeforeUpdate && persistence enabled:
//     a. Scale StatefulSet to 0 → requeue
//     b. Wait for pods to terminate → requeue
//     c. Create backup job → requeue
//     d. Poll job → requeue
//     e. On failure: abort update
//  3. Patch spec.image.tag = pendingVersion
//  4. Clear pendingVersion, set currentVersion, lastUpdateTime
func (r *OpenClawInstanceReconciler) driveUpdateStateMachine(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	pendingVersion := instance.Status.AutoUpdate.PendingVersion

	// Step 1: Set phase to Updating
	if instance.Status.Phase != openclawv1alpha1.PhaseUpdating {
		instance.Status.Phase = openclawv1alpha1.PhaseUpdating
		updatePhaseMetric(instance.Name, instance.Namespace, instance.Status.Phase)
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Step 2: Backup if needed
	needsBackup := instance.Spec.AutoUpdate.BackupBeforeUpdate == nil || *instance.Spec.AutoUpdate.BackupBeforeUpdate
	persistenceEnabled := instance.Spec.Storage.Persistence.Enabled == nil || *instance.Spec.Storage.Persistence.Enabled

	if needsBackup && persistenceEnabled {
		switch instance.Status.AutoUpdate.UpdatePhase {
		case "", updatePhaseBackingUp:
			result, done, err := r.drivePreUpdateBackup(ctx, instance)
			if err != nil {
				return r.abortUpdate(ctx, instance, fmt.Sprintf("backup failed: %v", err))
			}
			if !done {
				// Still in progress
				instance.Status.AutoUpdate.UpdatePhase = updatePhaseBackingUp
				if statusErr := r.Status().Update(ctx, instance); statusErr != nil {
					return ctrl.Result{}, statusErr
				}
				return result, nil
			}
			// Backup complete, fall through to apply update
		}
	}

	// Step 3: Apply the update — patch spec.image.tag
	instance.Status.AutoUpdate.UpdatePhase = updatePhaseApplyingUpdate
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Applying auto-update", "from", instance.Spec.Image.Tag, "to", pendingVersion)

	original := instance.DeepCopy()
	instance.Spec.Image.Tag = pendingVersion
	if err := r.Patch(ctx, instance, client.MergeFrom(original)); err != nil {
		return r.abortUpdate(ctx, instance, fmt.Sprintf("failed to patch image tag: %v", err))
	}

	// Step 4: Update status
	now := metav1.Now()
	instance.Status.AutoUpdate.CurrentVersion = pendingVersion
	instance.Status.AutoUpdate.PendingVersion = ""
	instance.Status.AutoUpdate.UpdatePhase = ""
	instance.Status.AutoUpdate.LastUpdateTime = &now
	instance.Status.AutoUpdate.LastUpdateError = ""
	instance.Status.Phase = openclawv1alpha1.PhaseProvisioning

	meta.RemoveStatusCondition(&instance.Status.Conditions, openclawv1alpha1.ConditionTypeAutoUpdateAvailable)

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	autoUpdateAppliedTotal.WithLabelValues(instance.Name, instance.Namespace).Inc()
	r.Recorder.Event(instance, corev1.EventTypeNormal, "AutoUpdateApplied",
		fmt.Sprintf("Updated image tag to %s", pendingVersion))

	logger.Info("Auto-update applied successfully", "version", pendingVersion)
	return ctrl.Result{Requeue: true}, nil
}

// drivePreUpdateBackup handles the backup steps before applying an update.
// Returns (result, done, err) where done=true means backup completed successfully.
func (r *OpenClawInstanceReconciler) drivePreUpdateBackup(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) (requeueResult ctrl.Result, done bool, retErr error) {
	logger := log.FromContext(ctx)

	// Scale down StatefulSet
	sts := &appsv1.StatefulSet{}
	stsKey := client.ObjectKey{Name: resources.StatefulSetName(instance), Namespace: instance.Namespace}
	if getErr := r.Get(ctx, stsKey, sts); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// No StatefulSet, skip directly to applying update
			return ctrl.Result{}, true, nil
		}
		return ctrl.Result{}, false, getErr
	}

	if sts.Spec.Replicas == nil || *sts.Spec.Replicas > 0 {
		logger.Info("Scaling down StatefulSet for pre-update backup")
		zero := int32(0)
		original := sts.DeepCopy()
		sts.Spec.Replicas = &zero
		if patchErr := r.Patch(ctx, sts, client.MergeFrom(original)); patchErr != nil {
			return ctrl.Result{}, false, patchErr
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, false, nil
	}

	// Wait for pods to terminate
	podList := &corev1.PodList{}
	if listErr := r.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(resources.SelectorLabels(instance)),
	); listErr != nil {
		return ctrl.Result{}, false, listErr
	}
	if len(podList.Items) > 0 {
		logger.Info("Waiting for pods to terminate for pre-update backup", "count", len(podList.Items))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, false, nil
	}

	// Get B2 credentials
	creds, err := r.getB2Credentials(ctx)
	if err != nil {
		return ctrl.Result{}, false, fmt.Errorf("failed to get B2 credentials: %w", err)
	}

	// Create or check backup job
	jobName := preUpdateBackupJobName(instance)
	existingJob, err := r.getJob(ctx, jobName, instance.Namespace)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, false, err
	}

	if apierrors.IsNotFound(err) || existingJob == nil {
		// Create the backup job
		tenantID := getTenantID(instance)
		timestamp := time.Now().UTC().Format("2006-01-02T150405Z")
		b2Path := fmt.Sprintf("backups/%s/%s/pre-update-%s", tenantID, instance.Name, timestamp)
		pvcName := pvcNameForInstance(instance)
		labels := backupLabels(instance, "pre-update-backup")

		job := buildRcloneJob(jobName, instance.Namespace, pvcName, b2Path, labels, creds, true)
		if err := controllerutil.SetControllerReference(instance, job, r.Scheme); err != nil {
			return ctrl.Result{}, false, err
		}

		logger.Info("Creating pre-update backup job", "job", jobName, "b2Path", b2Path)
		if err := r.Create(ctx, job); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{RequeueAfter: 10 * time.Second}, false, nil
			}
			return ctrl.Result{}, false, err
		}
		r.Recorder.Event(instance, corev1.EventTypeNormal, "PreUpdateBackupStarted",
			fmt.Sprintf("Pre-update backup job %s created", jobName))
		return ctrl.Result{RequeueAfter: 10 * time.Second}, false, nil
	}

	// Job exists — check status
	finished, condType := isJobFinished(existingJob)
	if !finished {
		logger.Info("Pre-update backup job still running", "job", jobName)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, false, nil
	}

	if condType == batchv1.JobFailed {
		return ctrl.Result{}, false, fmt.Errorf("pre-update backup job %s failed", jobName)
	}

	// Backup succeeded
	logger.Info("Pre-update backup completed successfully", "job", jobName)
	r.Recorder.Event(instance, corev1.EventTypeNormal, "PreUpdateBackupComplete",
		"Pre-update backup completed successfully")

	return ctrl.Result{}, true, nil
}

// abortUpdate clears the update state and returns to Running phase.
func (r *OpenClawInstanceReconciler) abortUpdate(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance, reason string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(nil, "Aborting auto-update", "reason", reason)

	instance.Status.AutoUpdate.PendingVersion = ""
	instance.Status.AutoUpdate.UpdatePhase = ""
	instance.Status.AutoUpdate.LastUpdateError = reason
	instance.Status.Phase = openclawv1alpha1.PhaseRunning

	meta.RemoveStatusCondition(&instance.Status.Conditions, openclawv1alpha1.ConditionTypeAutoUpdateAvailable)

	r.Recorder.Event(instance, corev1.EventTypeWarning, "AutoUpdateAborted", reason)

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Scale StatefulSet back up (if it was scaled down)
	sts := &appsv1.StatefulSet{}
	stsKey := client.ObjectKey{Name: resources.StatefulSetName(instance), Namespace: instance.Namespace}
	if err := r.Get(ctx, stsKey, sts); err == nil {
		if sts.Spec.Replicas != nil && *sts.Spec.Replicas == 0 {
			one := int32(1)
			original := sts.DeepCopy()
			sts.Spec.Replicas = &one
			if patchErr := r.Patch(ctx, sts, client.MergeFrom(original)); patchErr != nil {
				logger.Error(patchErr, "Failed to scale StatefulSet back up after aborted update")
			}
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// preUpdateBackupJobName returns a deterministic name for the pre-update backup Job.
func preUpdateBackupJobName(instance *openclawv1alpha1.OpenClawInstance) string {
	return instance.Name + "-pre-update-backup"
}

// isAutoUpdateEnabled returns true if auto-update is enabled and no digest pin is set.
func isAutoUpdateEnabled(instance *openclawv1alpha1.OpenClawInstance) bool {
	if instance.Spec.AutoUpdate.Enabled == nil || !*instance.Spec.AutoUpdate.Enabled {
		return false
	}
	// Digest pin overrides auto-update
	if instance.Spec.Image.Digest != "" {
		return false
	}
	return true
}

// shouldCheckForUpdate returns true if enough time has elapsed since the last check.
func shouldCheckForUpdate(instance *openclawv1alpha1.OpenClawInstance) bool {
	if instance.Status.AutoUpdate.LastCheckTime == nil {
		return true
	}
	interval := parseCheckInterval(instance.Spec.AutoUpdate.CheckInterval)
	return time.Since(instance.Status.AutoUpdate.LastCheckTime.Time) >= interval
}

// parseCheckInterval parses the check interval string with min/max bounds.
func parseCheckInterval(s string) time.Duration {
	if s == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	if d < time.Hour {
		return time.Hour
	}
	if d > 168*time.Hour {
		return 168 * time.Hour
	}
	return d
}
