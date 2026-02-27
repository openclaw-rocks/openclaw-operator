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

package skillpacks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/openclawrocks/k8s-operator/internal/resources"
)

const defaultBaseURL = "https://api.github.com"

// Resolver fetches skill pack manifests and files from GitHub repositories.
type Resolver struct {
	cacheTTL    time.Duration
	httpClient  *http.Client
	githubToken string
	baseURL     string // GitHub API base URL (overridable for tests)

	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	resolved  *resources.ResolvedSkillPacks
	fetchedAt time.Time
}

// packRef is a parsed pack:owner/repo/path[@ref] reference.
type packRef struct {
	Owner string
	Repo  string
	Path  string
	Ref   string // empty means default branch
}

// manifest is the skillpack.json structure.
type manifest struct {
	Files       map[string]string      `json:"files"`
	Directories []string               `json:"directories"`
	Config      map[string]interface{} `json:"config"`
}

// contentsResponse is the GitHub Contents API response for a single file.
type contentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// NewResolver creates a new GitHub-based skill pack resolver.
func NewResolver(cacheTTL time.Duration, githubToken string) *Resolver {
	return &Resolver{
		cacheTTL:    cacheTTL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		githubToken: githubToken,
		baseURL:     defaultBaseURL,
		cache:       make(map[string]*cacheEntry),
	}
}

// Resolve fetches and resolves the given pack names from GitHub.
// Returns nil if packNames is empty.
func (r *Resolver) Resolve(ctx context.Context, packNames []string) (*resources.ResolvedSkillPacks, error) {
	if len(packNames) == 0 {
		return nil, nil
	}

	// Sort for deterministic output
	sorted := make([]string, len(packNames))
	copy(sorted, packNames)
	sort.Strings(sorted)

	merged := &resources.ResolvedSkillPacks{
		Files:        make(map[string]string),
		PathMapping:  make(map[string]string),
		SkillEntries: make(map[string]interface{}),
	}

	for _, name := range sorted {
		resolved, err := r.resolvePack(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("skill pack %q: %w", name, err)
		}

		// Merge into combined result
		for k, v := range resolved.Files {
			merged.Files[k] = v
		}
		for k, v := range resolved.PathMapping {
			merged.PathMapping[k] = v
		}
		merged.Directories = append(merged.Directories, resolved.Directories...)
		for k, v := range resolved.SkillEntries {
			merged.SkillEntries[k] = v
		}
	}

	// Deduplicate and sort directories
	dirSet := make(map[string]bool)
	for _, d := range merged.Directories {
		dirSet[d] = true
	}
	merged.Directories = nil
	for d := range dirSet {
		merged.Directories = append(merged.Directories, d)
	}
	sort.Strings(merged.Directories)

	return merged, nil
}

// resolvePack resolves a single pack reference, using cache if valid.
func (r *Resolver) resolvePack(ctx context.Context, name string) (*resources.ResolvedSkillPacks, error) {
	r.mu.RLock()
	if entry, ok := r.cache[name]; ok && time.Since(entry.fetchedAt) < r.cacheTTL {
		resolved := entry.resolved
		r.mu.RUnlock()
		return resolved, nil
	}
	r.mu.RUnlock()

	resolved, err := r.fetchPack(ctx, name)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[name] = &cacheEntry{resolved: resolved, fetchedAt: time.Now()}
	r.mu.Unlock()

	return resolved, nil
}

// fetchPack fetches a skill pack manifest and its files from GitHub.
func (r *Resolver) fetchPack(ctx context.Context, name string) (*resources.ResolvedSkillPacks, error) {
	ref, err := parsePackRef(name)
	if err != nil {
		return nil, err
	}

	// Fetch skillpack.json
	manifestPath := ref.Path + "/skillpack.json"
	manifestBytes, err := r.fetchFile(ctx, ref.Owner, ref.Repo, manifestPath, ref.Ref)
	if err != nil {
		return nil, fmt.Errorf("fetching skillpack.json: %w", err)
	}

	var m manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return nil, fmt.Errorf("parsing skillpack.json: %w", err)
	}

	resolved := &resources.ResolvedSkillPacks{
		Files:        make(map[string]string),
		PathMapping:  make(map[string]string),
		Directories:  m.Directories,
		SkillEntries: make(map[string]interface{}),
	}

	// Fetch each file listed in the manifest
	for wsPath, repoRelPath := range m.Files {
		filePath := ref.Path + "/" + repoRelPath
		content, err := r.fetchFile(ctx, ref.Owner, ref.Repo, filePath, ref.Ref)
		if err != nil {
			return nil, fmt.Errorf("fetching file %q: %w", repoRelPath, err)
		}
		cmKey := resources.SkillPackCMKey(wsPath)
		resolved.Files[cmKey] = string(content)
		resolved.PathMapping[cmKey] = wsPath
	}

	// Copy config entries
	for k, v := range m.Config {
		resolved.SkillEntries[k] = v
	}

	return resolved, nil
}

// fetchFile retrieves a single file from GitHub using the Contents API.
func (r *Resolver) fetchFile(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", r.baseURL, owner, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if r.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.githubToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found: %s/%s/%s", owner, repo, path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s/%s/%s", resp.StatusCode, owner, repo, path)
	}

	var cr contentsResponse
	if decErr := json.NewDecoder(resp.Body).Decode(&cr); decErr != nil {
		return nil, fmt.Errorf("decoding response: %w", decErr)
	}

	if cr.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding %q (expected base64)", cr.Encoding)
	}

	// GitHub base64 content may contain newlines
	cleaned := strings.ReplaceAll(cr.Content, "\n", "")
	data, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 content: %w", err)
	}

	return data, nil
}

// parsePackRef parses "owner/repo/path[@ref]" into its components.
func parsePackRef(name string) (*packRef, error) {
	// Split off optional @ref
	ref := ""
	atIdx := strings.LastIndex(name, "@")
	base := name
	if atIdx > 0 {
		ref = name[atIdx+1:]
		base = name[:atIdx]
	}

	// Split into owner/repo/path (minimum 3 segments)
	parts := strings.SplitN(base, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid pack reference %q: expected owner/repo/path[@ref]", name)
	}

	return &packRef{
		Owner: parts[0],
		Repo:  parts[1],
		Path:  parts[2],
		Ref:   ref,
	}, nil
}
