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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

func newTestInstance() *openclawv1alpha1.OpenClawInstance {
	return &openclawv1alpha1.OpenClawInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inst1",
			Namespace: "test-ns",
		},
		Spec: openclawv1alpha1.OpenClawInstanceSpec{},
	}
}

func newTestSelfConfig() *openclawv1alpha1.OpenClawSelfConfig {
	return &openclawv1alpha1.OpenClawSelfConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sc1",
			Namespace: "test-ns",
		},
		Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
			InstanceRef: "inst1",
		},
	}
}

func TestDetermineActions_Skills(t *testing.T) {
	sc := newTestSelfConfig()
	sc.Spec.AddSkills = []string{"@anthropic/mcp-server-fetch"}

	actions := determineActions(sc)
	if len(actions) != 1 || actions[0] != openclawv1alpha1.SelfConfigActionSkills {
		t.Errorf("expected [skills], got %v", actions)
	}
}

func TestDetermineActions_Multiple(t *testing.T) {
	sc := newTestSelfConfig()
	sc.Spec.AddSkills = []string{"skill1"}
	sc.Spec.RemoveEnvVars = []string{"FOO"}

	actions := determineActions(sc)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
}

func TestDetermineActions_Empty(t *testing.T) {
	sc := newTestSelfConfig()

	actions := determineActions(sc)
	if len(actions) != 0 {
		t.Errorf("expected no actions, got %v", actions)
	}
}

func TestCheckAllowedActions_AllAllowed(t *testing.T) {
	requested := []openclawv1alpha1.SelfConfigAction{
		openclawv1alpha1.SelfConfigActionSkills,
		openclawv1alpha1.SelfConfigActionConfig,
	}
	allowed := []openclawv1alpha1.SelfConfigAction{
		openclawv1alpha1.SelfConfigActionSkills,
		openclawv1alpha1.SelfConfigActionConfig,
		openclawv1alpha1.SelfConfigActionEnvVars,
	}

	denied := checkAllowedActions(requested, allowed)
	if len(denied) != 0 {
		t.Errorf("expected no denied, got %v", denied)
	}
}

func TestCheckAllowedActions_SomeDenied(t *testing.T) {
	requested := []openclawv1alpha1.SelfConfigAction{
		openclawv1alpha1.SelfConfigActionSkills,
		openclawv1alpha1.SelfConfigActionEnvVars,
	}
	allowed := []openclawv1alpha1.SelfConfigAction{
		openclawv1alpha1.SelfConfigActionSkills,
	}

	denied := checkAllowedActions(requested, allowed)
	if len(denied) != 1 || denied[0] != openclawv1alpha1.SelfConfigActionEnvVars {
		t.Errorf("expected [envVars] denied, got %v", denied)
	}
}

func TestCheckAllowedActions_EmptyAllowed(t *testing.T) {
	requested := []openclawv1alpha1.SelfConfigAction{
		openclawv1alpha1.SelfConfigActionSkills,
	}
	denied := checkAllowedActions(requested, nil)
	if len(denied) != 1 {
		t.Errorf("expected 1 denied, got %v", denied)
	}
}

func TestApplySkillChanges_Add(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Skills = []string{"existing-skill"}

	sc := newTestSelfConfig()
	sc.Spec.AddSkills = []string{"new-skill"}

	applySkillChanges(instance, sc)

	if len(instance.Spec.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(instance.Spec.Skills))
	}
	if instance.Spec.Skills[1] != "new-skill" {
		t.Errorf("expected new-skill, got %q", instance.Spec.Skills[1])
	}
}

func TestApplySkillChanges_AddDeduplicate(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Skills = []string{"existing-skill"}

	sc := newTestSelfConfig()
	sc.Spec.AddSkills = []string{"existing-skill", "new-skill"}

	applySkillChanges(instance, sc)

	if len(instance.Spec.Skills) != 2 {
		t.Fatalf("expected 2 skills (deduplicated), got %d", len(instance.Spec.Skills))
	}
}

func TestApplySkillChanges_Remove(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Skills = []string{"skill-a", "skill-b", "skill-c"}

	sc := newTestSelfConfig()
	sc.Spec.RemoveSkills = []string{"skill-b"}

	applySkillChanges(instance, sc)

	if len(instance.Spec.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(instance.Spec.Skills))
	}
	for _, s := range instance.Spec.Skills {
		if s == "skill-b" {
			t.Error("skill-b should have been removed")
		}
	}
}

func TestApplyConfigPatch_Merge(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"mcpServers":{"existing":{"command":"node"}},"key":"value"}`)},
	}

	sc := newTestSelfConfig()
	sc.Spec.ConfigPatch = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"mcpServers":{"new":{"command":"python"}},"newKey":"newValue"}`)},
	}

	if err := applyConfigPatch(instance, sc); err != nil {
		t.Fatalf("applyConfigPatch failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Existing key preserved
	if result["key"] != "value" {
		t.Error("existing key 'key' should be preserved")
	}
	// New key added
	if result["newKey"] != "newValue" {
		t.Error("new key 'newKey' should be added")
	}
	// Both MCP servers present
	servers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers should be a map")
	}
	if _, ok := servers["existing"]; !ok {
		t.Error("existing MCP server should be preserved")
	}
	if _, ok := servers["new"]; !ok {
		t.Error("new MCP server should be added")
	}
}

func TestApplyConfigPatch_ProtectedKey(t *testing.T) {
	instance := newTestInstance()
	sc := newTestSelfConfig()
	sc.Spec.ConfigPatch = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"gateway":{"auth":{"token":"hacked"}}}`)},
	}

	err := applyConfigPatch(instance, sc)
	if err == nil {
		t.Error("expected error for protected config key 'gateway'")
	}
}

func TestApplyConfigPatch_EmptyBase(t *testing.T) {
	instance := newTestInstance()
	// No existing config

	sc := newTestSelfConfig()
	sc.Spec.ConfigPatch = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
	}

	if err := applyConfigPatch(instance, sc); err != nil {
		t.Fatalf("applyConfigPatch failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if result["key"] != "value" {
		t.Error("key should be set")
	}
}

func TestApplyWorkspaceFileChanges_Add(t *testing.T) {
	instance := newTestInstance()

	sc := newTestSelfConfig()
	sc.Spec.AddWorkspaceFiles = map[string]string{
		"notes.md": "# Notes",
	}

	applyWorkspaceFileChanges(instance, sc)

	if instance.Spec.Workspace == nil {
		t.Fatal("workspace should be initialized")
	}
	if instance.Spec.Workspace.InitialFiles["notes.md"] != "# Notes" {
		t.Error("notes.md should be added")
	}
}

func TestApplyWorkspaceFileChanges_Remove(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
		InitialFiles: map[string]string{
			"keep.md":   "keep",
			"remove.md": "remove",
		},
	}

	sc := newTestSelfConfig()
	sc.Spec.RemoveWorkspaceFiles = []string{"remove.md"}

	applyWorkspaceFileChanges(instance, sc)

	if _, ok := instance.Spec.Workspace.InitialFiles["remove.md"]; ok {
		t.Error("remove.md should have been removed")
	}
	if instance.Spec.Workspace.InitialFiles["keep.md"] != "keep" {
		t.Error("keep.md should be preserved")
	}
}

func TestApplyEnvVarChanges_Add(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "EXISTING", Value: "value1"},
	}

	sc := newTestSelfConfig()
	sc.Spec.AddEnvVars = []openclawv1alpha1.SelfConfigEnvVar{
		{Name: "NEW_VAR", Value: "new_value"},
	}

	if err := applyEnvVarChanges(instance, sc); err != nil {
		t.Fatalf("applyEnvVarChanges failed: %v", err)
	}

	if len(instance.Spec.Env) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(instance.Spec.Env))
	}
}

func TestApplyEnvVarChanges_Replace(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "MY_VAR", Value: "old"},
	}

	sc := newTestSelfConfig()
	sc.Spec.AddEnvVars = []openclawv1alpha1.SelfConfigEnvVar{
		{Name: "MY_VAR", Value: "new"},
	}

	if err := applyEnvVarChanges(instance, sc); err != nil {
		t.Fatalf("applyEnvVarChanges failed: %v", err)
	}

	if len(instance.Spec.Env) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(instance.Spec.Env))
	}
	if instance.Spec.Env[0].Value != "new" {
		t.Errorf("expected value 'new', got %q", instance.Spec.Env[0].Value)
	}
}

func TestApplyEnvVarChanges_Remove(t *testing.T) {
	instance := newTestInstance()
	instance.Spec.Env = []corev1.EnvVar{
		{Name: "KEEP", Value: "yes"},
		{Name: "REMOVE", Value: "bye"},
	}

	sc := newTestSelfConfig()
	sc.Spec.RemoveEnvVars = []string{"REMOVE"}

	if err := applyEnvVarChanges(instance, sc); err != nil {
		t.Fatalf("applyEnvVarChanges failed: %v", err)
	}

	if len(instance.Spec.Env) != 1 || instance.Spec.Env[0].Name != "KEEP" {
		t.Error("should only have KEEP env var")
	}
}

func TestApplyEnvVarChanges_ProtectedAdd(t *testing.T) {
	instance := newTestInstance()

	sc := newTestSelfConfig()
	sc.Spec.AddEnvVars = []openclawv1alpha1.SelfConfigEnvVar{
		{Name: "HOME", Value: "/hacked"},
	}

	err := applyEnvVarChanges(instance, sc)
	if err == nil {
		t.Error("expected error for protected env var HOME")
	}
}

func TestApplyEnvVarChanges_ProtectedRemove(t *testing.T) {
	instance := newTestInstance()

	sc := newTestSelfConfig()
	sc.Spec.RemoveEnvVars = []string{"OPENCLAW_GATEWAY_TOKEN"}

	err := applyEnvVarChanges(instance, sc)
	if err == nil {
		t.Error("expected error for removing protected env var OPENCLAW_GATEWAY_TOKEN")
	}
}

func TestDeepMerge(t *testing.T) {
	dst := map[string]interface{}{
		"a": "1",
		"b": map[string]interface{}{
			"c": "2",
			"d": "3",
		},
	}
	src := map[string]interface{}{
		"b": map[string]interface{}{
			"d": "4",
			"e": "5",
		},
		"f": "6",
	}

	result := deepMerge(dst, src)

	if result["a"] != "1" {
		t.Error("a should be preserved")
	}
	b, ok := result["b"].(map[string]interface{})
	if !ok {
		t.Fatal("b should be a map")
	}
	if b["c"] != "2" {
		t.Error("b.c should be preserved")
	}
	if b["d"] != "4" {
		t.Error("b.d should be overwritten by src")
	}
	if b["e"] != "5" {
		t.Error("b.e should be added from src")
	}
	if result["f"] != "6" {
		t.Error("f should be added from src")
	}
}
