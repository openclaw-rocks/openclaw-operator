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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// protectedConfigKeys are config paths that cannot be modified via self-config
// to prevent breaking gateway authentication.
var protectedConfigKeys = map[string]bool{
	"gateway": true, // block entire gateway subtree for safety
}

// protectedEnvVars are environment variable names that cannot be overridden
// via self-config because they are operator-managed.
var protectedEnvVars = map[string]bool{
	"HOME":                      true,
	"OPENCLAW_DISABLE_BONJOUR":  true,
	"OPENCLAW_GATEWAY_TOKEN":    true,
	"OPENCLAW_INSTANCE_NAME":    true,
	"OPENCLAW_NAMESPACE":        true,
	"PATH":                      true,
	"CHROMIUM_URL":              true,
	"OLLAMA_HOST":               true,
	"TS_AUTHKEY":                true,
	"TS_HOSTNAME":               true,
	"NODE_EXTRA_CA_CERTS":       true,
	"NPM_CONFIG_CACHE":          true,
	"NPM_CONFIG_IGNORE_SCRIPTS": true,
}

// determineActions inspects which action categories a SelfConfig request uses.
func determineActions(sc *openclawv1alpha1.OpenClawSelfConfig) []openclawv1alpha1.SelfConfigAction {
	var actions []openclawv1alpha1.SelfConfigAction
	if len(sc.Spec.AddSkills) > 0 || len(sc.Spec.RemoveSkills) > 0 {
		actions = append(actions, openclawv1alpha1.SelfConfigActionSkills)
	}
	if sc.Spec.ConfigPatch != nil {
		actions = append(actions, openclawv1alpha1.SelfConfigActionConfig)
	}
	if len(sc.Spec.AddWorkspaceFiles) > 0 || len(sc.Spec.RemoveWorkspaceFiles) > 0 {
		actions = append(actions, openclawv1alpha1.SelfConfigActionWorkspaceFiles)
	}
	if len(sc.Spec.AddEnvVars) > 0 || len(sc.Spec.RemoveEnvVars) > 0 {
		actions = append(actions, openclawv1alpha1.SelfConfigActionEnvVars)
	}
	return actions
}

// checkAllowedActions validates that all requested actions are in the allowed list.
// Returns a list of denied action names, or nil if all are allowed.
func checkAllowedActions(requested, allowed []openclawv1alpha1.SelfConfigAction) []openclawv1alpha1.SelfConfigAction {
	allowedSet := make(map[openclawv1alpha1.SelfConfigAction]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}

	var denied []openclawv1alpha1.SelfConfigAction
	for _, a := range requested {
		if !allowedSet[a] {
			denied = append(denied, a)
		}
	}
	return denied
}

// applySkillChanges adds and removes skills from the instance spec.
func applySkillChanges(instance *openclawv1alpha1.OpenClawInstance, sc *openclawv1alpha1.OpenClawSelfConfig) {
	// Remove skills
	if len(sc.Spec.RemoveSkills) > 0 {
		removeSet := make(map[string]bool, len(sc.Spec.RemoveSkills))
		for _, s := range sc.Spec.RemoveSkills {
			removeSet[s] = true
		}
		filtered := make([]string, 0, len(instance.Spec.Skills))
		for _, s := range instance.Spec.Skills {
			if !removeSet[s] {
				filtered = append(filtered, s)
			}
		}
		instance.Spec.Skills = filtered
	}

	// Add skills (deduplicate, preserve order)
	if len(sc.Spec.AddSkills) > 0 {
		existing := make(map[string]bool, len(instance.Spec.Skills))
		for _, s := range instance.Spec.Skills {
			existing[s] = true
		}
		for _, s := range sc.Spec.AddSkills {
			if !existing[s] {
				instance.Spec.Skills = append(instance.Spec.Skills, s)
				existing[s] = true
			}
		}
	}
}

// applyConfigPatch deep-merges the config patch into the instance config.
// Returns an error if protected keys are present in the patch.
func applyConfigPatch(instance *openclawv1alpha1.OpenClawInstance, sc *openclawv1alpha1.OpenClawSelfConfig) error {
	if sc.Spec.ConfigPatch == nil || len(sc.Spec.ConfigPatch.Raw) == 0 {
		return nil
	}

	// Parse patch to check for protected keys
	var patch map[string]interface{}
	if err := json.Unmarshal(sc.Spec.ConfigPatch.Raw, &patch); err != nil {
		return fmt.Errorf("invalid config patch JSON: %w", err)
	}

	for key := range patch {
		if protectedConfigKeys[key] {
			return fmt.Errorf("config key %q is protected and cannot be modified via self-config", key)
		}
	}

	// Parse existing config (or start empty)
	var base map[string]interface{}
	if instance.Spec.Config.Raw != nil && len(instance.Spec.Config.Raw.Raw) > 0 {
		if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &base); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		base = make(map[string]interface{})
	}

	// Deep merge
	merged := deepMerge(base, patch)
	raw, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	if instance.Spec.Config.Raw == nil {
		instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{}
	}
	instance.Spec.Config.Raw.RawExtension = runtime.RawExtension{Raw: raw}
	return nil
}

// deepMerge recursively merges src into dst. Arrays are replaced, not merged.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(dst))
	for k, v := range dst {
		result[k] = v
	}
	for k, v := range src {
		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := result[k].(map[string]interface{}); ok {
				result[k] = deepMerge(dstMap, srcMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}

// applyWorkspaceFileChanges adds and removes workspace files.
func applyWorkspaceFileChanges(instance *openclawv1alpha1.OpenClawInstance, sc *openclawv1alpha1.OpenClawSelfConfig) {
	// Initialize workspace if needed
	if instance.Spec.Workspace == nil {
		instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{}
	}
	if instance.Spec.Workspace.InitialFiles == nil {
		instance.Spec.Workspace.InitialFiles = make(map[string]string)
	}

	// Remove files
	for _, name := range sc.Spec.RemoveWorkspaceFiles {
		delete(instance.Spec.Workspace.InitialFiles, name)
	}

	// Add files
	for name, content := range sc.Spec.AddWorkspaceFiles {
		instance.Spec.Workspace.InitialFiles[name] = content
	}
}

// applyEnvVarChanges adds and removes environment variables.
// Returns an error if protected env vars are targeted.
func applyEnvVarChanges(instance *openclawv1alpha1.OpenClawInstance, sc *openclawv1alpha1.OpenClawSelfConfig) error {
	// Check for protected env var additions
	for _, ev := range sc.Spec.AddEnvVars {
		if protectedEnvVars[ev.Name] {
			return fmt.Errorf("environment variable %q is protected and cannot be modified via self-config", ev.Name)
		}
	}

	// Check for protected env var removals
	for _, name := range sc.Spec.RemoveEnvVars {
		if protectedEnvVars[name] {
			return fmt.Errorf("environment variable %q is protected and cannot be removed via self-config", name)
		}
	}

	// Remove env vars
	if len(sc.Spec.RemoveEnvVars) > 0 {
		removeSet := make(map[string]bool, len(sc.Spec.RemoveEnvVars))
		for _, name := range sc.Spec.RemoveEnvVars {
			removeSet[name] = true
		}
		filtered := make([]corev1.EnvVar, 0, len(instance.Spec.Env))
		for _, ev := range instance.Spec.Env {
			if !removeSet[ev.Name] {
				filtered = append(filtered, ev)
			}
		}
		instance.Spec.Env = filtered
	}

	// Add env vars (replace existing with same name, or append)
	for _, newEV := range sc.Spec.AddEnvVars {
		found := false
		for i, ev := range instance.Spec.Env {
			if ev.Name == newEV.Name {
				instance.Spec.Env[i] = corev1.EnvVar{Name: newEV.Name, Value: newEV.Value}
				found = true
				break
			}
		}
		if !found {
			instance.Spec.Env = append(instance.Spec.Env, corev1.EnvVar{
				Name:  newEV.Name,
				Value: newEV.Value,
			})
		}
	}

	return nil
}
