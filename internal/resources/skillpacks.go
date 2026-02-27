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
	"fmt"
	"sort"
	"strings"
)

const (
	// SkillPackRegistryKey is the ConfigMap data key for the skill pack registry manifest.
	SkillPackRegistryKey = "registry.json"

	// SkillPackConfigMapName is the name of the ConfigMap containing skill pack definitions.
	SkillPackConfigMapName = "openclaw-skill-packs"

	// SkillPackPrefix is the prefix for skill pack entries in the skills list.
	SkillPackPrefix = "pack:"
)

// SkillPackRegistry is the top-level registry.json structure mapping pack names to definitions.
type SkillPackRegistry map[string]SkillPackDef

// SkillPackDef defines a single skill pack in the registry.
type SkillPackDef struct {
	// Files maps workspace-relative paths to ConfigMap data keys.
	Files map[string]string `json:"files"`

	// Directories to create in the workspace (mkdir -p).
	Directories []string `json:"directories"`

	// Config entries to inject into config.raw.skills.entries.
	// Each key is a skill name, value is the config object (e.g., {"enabled": true}).
	Config map[string]interface{} `json:"config"`
}

// ResolvedSkillPacks contains resolved workspace files, directories, and config for skill packs.
type ResolvedSkillPacks struct {
	// Files maps ConfigMap-safe keys to file content.
	Files map[string]string

	// PathMapping maps ConfigMap-safe keys to workspace-relative paths.
	PathMapping map[string]string

	// Directories to create in the workspace.
	Directories []string

	// SkillEntries to inject into config.raw.skills.entries.
	SkillEntries map[string]interface{}
}

// ExtractPackSkills returns pack names from the skills list (entries with "pack:" prefix).
func ExtractPackSkills(skills []string) []string {
	var packs []string
	for _, s := range skills {
		if name, ok := strings.CutPrefix(s, SkillPackPrefix); ok {
			packs = append(packs, name)
		}
	}
	return packs
}

// FilterNonPackSkills returns skills that are NOT pack: prefixed.
func FilterNonPackSkills(skills []string) []string {
	var result []string
	for _, s := range skills {
		if !strings.HasPrefix(s, SkillPackPrefix) {
			result = append(result, s)
		}
	}
	return result
}

// SkillPackCMKey converts a workspace-relative path to a ConfigMap-safe key.
// Replaces "/" with "--" to comply with ConfigMap key naming rules.
func SkillPackCMKey(wsPath string) string {
	return strings.ReplaceAll(wsPath, "/", "--")
}

// ResolveSkillPacks resolves pack: skills against the registry ConfigMap data.
// Returns nil if no pack names are provided.
func ResolveSkillPacks(packNames []string, cmData map[string]string) (*ResolvedSkillPacks, error) {
	if len(packNames) == 0 {
		return nil, nil
	}

	registryJSON, ok := cmData[SkillPackRegistryKey]
	if !ok {
		return nil, fmt.Errorf("skill pack ConfigMap missing %s key", SkillPackRegistryKey)
	}

	var registry SkillPackRegistry
	if err := json.Unmarshal([]byte(registryJSON), &registry); err != nil {
		return nil, fmt.Errorf("failed to parse skill pack registry: %w", err)
	}

	resolved := &ResolvedSkillPacks{
		Files:        make(map[string]string),
		PathMapping:  make(map[string]string),
		SkillEntries: make(map[string]interface{}),
	}

	// Sort pack names for deterministic output
	sorted := make([]string, len(packNames))
	copy(sorted, packNames)
	sort.Strings(sorted)

	for _, name := range sorted {
		pack, found := registry[name]
		if !found {
			return nil, fmt.Errorf("skill pack %q not found in registry", name)
		}

		// Resolve files
		for wsPath, cmKey := range pack.Files {
			content, found := cmData[cmKey]
			if !found {
				return nil, fmt.Errorf("skill pack %q: file %q references missing ConfigMap key %q", name, wsPath, cmKey)
			}
			safeCMKey := SkillPackCMKey(wsPath)
			resolved.Files[safeCMKey] = content
			resolved.PathMapping[safeCMKey] = wsPath
		}

		// Merge directories
		resolved.Directories = append(resolved.Directories, pack.Directories...)

		// Merge config entries
		for k, v := range pack.Config {
			resolved.SkillEntries[k] = v
		}
	}

	// Deduplicate and sort directories
	dirSet := make(map[string]bool)
	for _, d := range resolved.Directories {
		dirSet[d] = true
	}
	resolved.Directories = nil
	for d := range dirSet {
		resolved.Directories = append(resolved.Directories, d)
	}
	sort.Strings(resolved.Directories)

	return resolved, nil
}

// HasSkillPackFiles returns true if the resolved skill packs contain any workspace files.
func HasSkillPackFiles(sp *ResolvedSkillPacks) bool {
	return sp != nil && len(sp.Files) > 0
}
