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
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractPackSkills(t *testing.T) {
	skills := []string{
		"@anthropic/mcp-server-fetch",
		"npm:@openclaw/matrix",
		"pack:image-gen",
		"pack:code-runner",
	}
	packs := ExtractPackSkills(skills)
	if len(packs) != 2 {
		t.Fatalf("expected 2 packs, got %d", len(packs))
	}
	if packs[0] != "image-gen" || packs[1] != "code-runner" {
		t.Errorf("unexpected packs: %v", packs)
	}
}

func TestExtractPackSkills_None(t *testing.T) {
	skills := []string{"@anthropic/fetch", "npm:pkg"}
	packs := ExtractPackSkills(skills)
	if len(packs) != 0 {
		t.Fatalf("expected 0 packs, got %d", len(packs))
	}
}

func TestFilterNonPackSkills(t *testing.T) {
	skills := []string{
		"@anthropic/fetch",
		"pack:image-gen",
		"npm:@openclaw/matrix",
		"pack:code-runner",
	}
	filtered := FilterNonPackSkills(skills)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 non-pack skills, got %d", len(filtered))
	}
	if filtered[0] != "@anthropic/fetch" || filtered[1] != "npm:@openclaw/matrix" {
		t.Errorf("unexpected filtered: %v", filtered)
	}
}

func TestSkillPackCMKey(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"SKILL.md", "SKILL.md"},
		{"skills/image-gen/SKILL.md", "skills--image-gen--SKILL.md"},
		{"skills/image-gen/scripts/generate.py", "skills--image-gen--scripts--generate.py"},
	}
	for _, tt := range tests {
		if got := SkillPackCMKey(tt.input); got != tt.expected {
			t.Errorf("SkillPackCMKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveSkillPacks(t *testing.T) {
	cmData := map[string]string{
		"registry.json": `{
			"image-gen": {
				"files": {
					"skills/image-gen/SKILL.md": "image-gen.SKILL.md",
					"skills/image-gen/scripts/generate.py": "image-gen.scripts.generate.py"
				},
				"directories": ["skills/image-gen/scripts"],
				"config": {
					"image-gen": {"enabled": true},
					"openai-image-gen": {"enabled": false}
				}
			}
		}`,
		"image-gen.SKILL.md":              "---\nname: image-gen\n---\n",
		"image-gen.scripts.generate.py":   "#!/usr/bin/env python3\nprint('hello')\n",
	}

	resolved, err := ResolveSkillPacks([]string{"image-gen"}, cmData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved == nil {
		t.Fatal("expected resolved skill packs, got nil")
	}

	// Check files
	if len(resolved.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(resolved.Files))
	}
	skillMDKey := SkillPackCMKey("skills/image-gen/SKILL.md")
	if resolved.Files[skillMDKey] != "---\nname: image-gen\n---\n" {
		t.Errorf("unexpected SKILL.md content: %q", resolved.Files[skillMDKey])
	}
	genPyKey := SkillPackCMKey("skills/image-gen/scripts/generate.py")
	if resolved.Files[genPyKey] != "#!/usr/bin/env python3\nprint('hello')\n" {
		t.Errorf("unexpected generate.py content: %q", resolved.Files[genPyKey])
	}

	// Check path mapping
	if resolved.PathMapping[skillMDKey] != "skills/image-gen/SKILL.md" {
		t.Errorf("unexpected path mapping for SKILL.md: %q", resolved.PathMapping[skillMDKey])
	}
	if resolved.PathMapping[genPyKey] != "skills/image-gen/scripts/generate.py" {
		t.Errorf("unexpected path mapping for generate.py: %q", resolved.PathMapping[genPyKey])
	}

	// Check directories
	if len(resolved.Directories) != 1 || resolved.Directories[0] != "skills/image-gen/scripts" {
		t.Errorf("unexpected directories: %v", resolved.Directories)
	}

	// Check config entries
	if len(resolved.SkillEntries) != 2 {
		t.Errorf("expected 2 skill entries, got %d", len(resolved.SkillEntries))
	}
	imgGen, ok := resolved.SkillEntries["image-gen"].(map[string]interface{})
	if !ok || imgGen["enabled"] != true {
		t.Errorf("expected image-gen enabled: %v", resolved.SkillEntries["image-gen"])
	}
	oaiGen, ok := resolved.SkillEntries["openai-image-gen"].(map[string]interface{})
	if !ok || oaiGen["enabled"] != false {
		t.Errorf("expected openai-image-gen disabled: %v", resolved.SkillEntries["openai-image-gen"])
	}
}

func TestResolveSkillPacks_Empty(t *testing.T) {
	resolved, err := ResolveSkillPacks(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != nil {
		t.Errorf("expected nil for empty pack names, got %v", resolved)
	}
}

func TestResolveSkillPacks_MissingPack(t *testing.T) {
	cmData := map[string]string{
		"registry.json": `{}`,
	}
	_, err := ResolveSkillPacks([]string{"nonexistent"}, cmData)
	if err == nil {
		t.Fatal("expected error for missing pack")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveSkillPacks_MissingFileKey(t *testing.T) {
	cmData := map[string]string{
		"registry.json": `{
			"image-gen": {
				"files": {"skills/image-gen/SKILL.md": "missing-key"},
				"directories": [],
				"config": {}
			}
		}`,
	}
	_, err := ResolveSkillPacks([]string{"image-gen"}, cmData)
	if err == nil {
		t.Fatal("expected error for missing file key")
	}
	if !strings.Contains(err.Error(), "missing ConfigMap key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHasSkillPackFiles(t *testing.T) {
	if HasSkillPackFiles(nil) {
		t.Error("expected false for nil")
	}
	if HasSkillPackFiles(&ResolvedSkillPacks{}) {
		t.Error("expected false for empty")
	}
	if !HasSkillPackFiles(&ResolvedSkillPacks{Files: map[string]string{"a": "b"}}) {
		t.Error("expected true for non-empty files")
	}
}

func TestBuildSkillsScript_FiltersPackEntries(t *testing.T) {
	instance := newTestInstance("skills-filter")
	instance.Spec.Skills = []string{
		"pack:image-gen",
		"npm:@openclaw/matrix",
		"@anthropic/mcp-server-fetch",
	}
	script := BuildSkillsScript(instance)
	if strings.Contains(script, "image-gen") {
		t.Error("script should not contain pack:image-gen")
	}
	if !strings.Contains(script, "npm install") {
		t.Error("script should contain npm install for @openclaw/matrix")
	}
	if !strings.Contains(script, "clawhub install") {
		t.Error("script should contain clawhub install for @anthropic/mcp-server-fetch")
	}
}

func TestBuildSkillsScript_OnlyPackEntries(t *testing.T) {
	instance := newTestInstance("skills-only-packs")
	instance.Spec.Skills = []string{"pack:image-gen", "pack:code-runner"}
	script := BuildSkillsScript(instance)
	if script != "" {
		t.Errorf("expected empty script for pack-only skills, got: %s", script)
	}
}

func TestBuildInitScript_WithSkillPacks(t *testing.T) {
	instance := newTestInstance("skill-pack-init")
	instance.Spec.Workspace = nil
	instance.Spec.Config.Raw = nil

	resolved := &ResolvedSkillPacks{
		Files: map[string]string{
			"skills--image-gen--SKILL.md":              "skill content",
			"skills--image-gen--scripts--generate.py":  "script content",
		},
		PathMapping: map[string]string{
			"skills--image-gen--SKILL.md":              "skills/image-gen/SKILL.md",
			"skills--image-gen--scripts--generate.py":  "skills/image-gen/scripts/generate.py",
		},
		Directories: []string{"skills/image-gen/scripts"},
	}

	script := BuildInitScript(instance, resolved)

	// Should create directories
	if !strings.Contains(script, "mkdir -p /data/workspace/'skills/image-gen/scripts'") {
		t.Error("expected mkdir for skill pack directory")
	}

	// Should copy files with path mapping
	if !strings.Contains(script, "cp /workspace-init/'skills--image-gen--SKILL.md' /data/workspace/'skills/image-gen/SKILL.md'") {
		t.Errorf("expected skill pack file copy in script:\n%s", script)
	}
	if !strings.Contains(script, "cp /workspace-init/'skills--image-gen--scripts--generate.py' /data/workspace/'skills/image-gen/scripts/generate.py'") {
		t.Errorf("expected skill pack script copy in script:\n%s", script)
	}
}

func TestBuildWorkspaceConfigMap_WithSkillPacks(t *testing.T) {
	instance := newTestInstance("ws-skill-packs")

	resolved := &ResolvedSkillPacks{
		Files: map[string]string{
			"skills--image-gen--SKILL.md": "skill content",
		},
	}

	cm := BuildWorkspaceConfigMap(instance, resolved)
	if cm == nil {
		t.Fatal("expected non-nil ConfigMap")
	}
	if cm.Data["skills--image-gen--SKILL.md"] != "skill content" {
		t.Errorf("expected skill pack file in ConfigMap data")
	}
}

func TestEnrichConfigWithSkillPacks(t *testing.T) {
	config := `{"skills": {"entries": {"web-search": {"enabled": true}}}}`
	skillEntries := map[string]interface{}{
		"image-gen":       map[string]interface{}{"enabled": true},
		"openai-image-gen": map[string]interface{}{"enabled": false},
	}

	enriched, err := enrichConfigWithSkillPacks([]byte(config), skillEntries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(enriched, &result); err != nil {
		t.Fatalf("failed to parse enriched config: %v", err)
	}

	skills := result["skills"].(map[string]interface{})
	entries := skills["entries"].(map[string]interface{})

	// User-defined entry should be preserved
	if ws, ok := entries["web-search"].(map[string]interface{}); !ok || ws["enabled"] != true {
		t.Error("web-search entry should be preserved")
	}
	// Skill pack entries should be added
	if ig, ok := entries["image-gen"].(map[string]interface{}); !ok || ig["enabled"] != true {
		t.Error("image-gen should be enabled")
	}
	if oig, ok := entries["openai-image-gen"].(map[string]interface{}); !ok || oig["enabled"] != false {
		t.Error("openai-image-gen should be disabled")
	}
}

func TestEnrichConfigWithSkillPacks_UserOverrideWins(t *testing.T) {
	// User already disabled image-gen — skill pack should NOT override
	config := `{"skills": {"entries": {"image-gen": {"enabled": false}}}}`
	skillEntries := map[string]interface{}{
		"image-gen": map[string]interface{}{"enabled": true},
	}

	enriched, err := enrichConfigWithSkillPacks([]byte(config), skillEntries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(enriched, &result)
	entries := result["skills"].(map[string]interface{})["entries"].(map[string]interface{})
	ig := entries["image-gen"].(map[string]interface{})
	if ig["enabled"] != false {
		t.Error("user override should win — image-gen should remain disabled")
	}
}
