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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openclawrocks/k8s-operator/internal/resources"
)

// ghFile returns a GitHub Contents API JSON response for a file.
func ghFile(content string) string {
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	resp := contentsResponse{Content: b64, Encoding: "base64"}
	data, _ := json.Marshal(resp)
	return string(data)
}

// newTestResolver creates a resolver pointing at the httptest server.
func newTestResolver(server *httptest.Server, token string) *Resolver {
	r := NewResolver(5*time.Minute, token)
	r.baseURL = server.URL
	return r
}

func TestParsePackRef(t *testing.T) {
	tests := []struct {
		input   string
		owner   string
		repo    string
		path    string
		ref     string
		wantErr bool
	}{
		{"openclaw-rocks/skills/image-gen", "openclaw-rocks", "skills", "image-gen", "", false},
		{"openclaw-rocks/skills/image-gen@v1.0.0", "openclaw-rocks", "skills", "image-gen", "v1.0.0", false},
		{"myorg/private-skills/custom-tool@main", "myorg", "private-skills", "custom-tool", "main", false},
		{"openclaw-rocks/skills/nested/deep/path@abc123", "openclaw-rocks", "skills", "nested/deep/path", "abc123", false},
		{"owner/repo", "", "", "", "", true},
		{"invalid", "", "", "", "", true},
	}

	for _, tt := range tests {
		ref, err := parsePackRef(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parsePackRef(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePackRef(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if ref.Owner != tt.owner || ref.Repo != tt.repo || ref.Path != tt.path || ref.Ref != tt.ref {
			t.Errorf("parsePackRef(%q) = {%s, %s, %s, %s}, want {%s, %s, %s, %s}",
				tt.input, ref.Owner, ref.Repo, ref.Path, ref.Ref,
				tt.owner, tt.repo, tt.path, tt.ref)
		}
	}
}

func TestResolve_Success(t *testing.T) {
	manifestJSON := `{
		"files": {
			"skills/image-gen/SKILL.md": "SKILL.md",
			"skills/image-gen/scripts/generate.py": "scripts/generate.py"
		},
		"directories": ["skills/image-gen/scripts"],
		"config": {
			"image-gen": {"enabled": true},
			"openai-image-gen": {"enabled": false}
		}
	}`

	skillMD := "---\nname: image-gen\n---\n"
	generatePy := "#!/usr/bin/env python3\nprint('hello')\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "image-gen/skillpack.json"):
			_, _ = w.Write([]byte(ghFile(manifestJSON)))
		case strings.HasSuffix(path, "image-gen/SKILL.md"):
			_, _ = w.Write([]byte(ghFile(skillMD)))
		case strings.HasSuffix(path, "image-gen/scripts/generate.py"):
			_, _ = w.Write([]byte(ghFile(generatePy)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	resolved, err := resolver.Resolve(context.Background(), []string{"test-owner/test-repo/image-gen"})
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
	skillMDKey := resources.SkillPackCMKey("skills/image-gen/SKILL.md")
	if resolved.Files[skillMDKey] != skillMD {
		t.Errorf("unexpected SKILL.md content: %q", resolved.Files[skillMDKey])
	}
	genPyKey := resources.SkillPackCMKey("skills/image-gen/scripts/generate.py")
	if resolved.Files[genPyKey] != generatePy {
		t.Errorf("unexpected generate.py content: %q", resolved.Files[genPyKey])
	}

	// Check path mapping
	if resolved.PathMapping[skillMDKey] != "skills/image-gen/SKILL.md" {
		t.Errorf("unexpected path mapping: %q", resolved.PathMapping[skillMDKey])
	}
	if resolved.PathMapping[genPyKey] != "skills/image-gen/scripts/generate.py" {
		t.Errorf("unexpected path mapping: %q", resolved.PathMapping[genPyKey])
	}

	// Check directories
	if len(resolved.Directories) != 1 || resolved.Directories[0] != "skills/image-gen/scripts" {
		t.Errorf("unexpected directories: %v", resolved.Directories)
	}

	// Check config
	if len(resolved.SkillEntries) != 2 {
		t.Errorf("expected 2 skill entries, got %d", len(resolved.SkillEntries))
	}
	imgGen, ok := resolved.SkillEntries["image-gen"].(map[string]interface{})
	if !ok || imgGen["enabled"] != true {
		t.Errorf("expected image-gen enabled: %v", resolved.SkillEntries["image-gen"])
	}
}

func TestResolve_Empty(t *testing.T) {
	resolver := NewResolver(5*time.Minute, "")
	resolved, err := resolver.Resolve(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != nil {
		t.Errorf("expected nil for empty pack names, got %v", resolved)
	}
}

func TestResolve_MissingManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/missing-pack"})
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolve_MissingFile(t *testing.T) {
	manifestJSON := `{
		"files": {"skills/test/SKILL.md": "SKILL.md"},
		"directories": [],
		"config": {}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "skillpack.json") {
			_, _ = w.Write([]byte(ghFile(manifestJSON)))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/test-pack"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolve_Caching(t *testing.T) {
	callCount := 0
	manifestJSON := `{"files": {}, "directories": [], "config": {}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_, _ = w.Write([]byte(ghFile(manifestJSON)))
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	// First call
	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/test"})
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	firstCallCount := callCount

	// Second call should use cache
	_, err = resolver.Resolve(context.Background(), []string{"owner/repo/test"})
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if callCount != firstCallCount {
		t.Errorf("expected cached result, but got %d additional API calls", callCount-firstCallCount)
	}
}

func TestResolve_AuthHeader(t *testing.T) {
	var gotAuth string
	manifestJSON := `{"files": {}, "directories": [], "config": {}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(ghFile(manifestJSON)))
	}))
	defer server.Close()

	resolver := newTestResolver(server, "ghp_test_token_123")

	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer ghp_test_token_123" {
		t.Errorf("expected auth header 'Bearer ghp_test_token_123', got %q", gotAuth)
	}
}

func TestResolve_NoAuthWhenTokenEmpty(t *testing.T) {
	var gotAuth string
	manifestJSON := `{"files": {}, "directories": [], "config": {}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(ghFile(manifestJSON)))
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "" {
		t.Errorf("expected no auth header, got %q", gotAuth)
	}
}

func TestResolve_WithRef(t *testing.T) {
	var gotRef string
	manifestJSON := `{"files": {}, "directories": [], "config": {}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRef = r.URL.Query().Get("ref")
		_, _ = w.Write([]byte(ghFile(manifestJSON)))
	}))
	defer server.Close()

	resolver := newTestResolver(server, "")

	_, err := resolver.Resolve(context.Background(), []string{"owner/repo/test@v1.2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotRef != "v1.2.3" {
		t.Errorf("expected ref 'v1.2.3', got %q", gotRef)
	}
}

func TestResolve_InvalidRef(t *testing.T) {
	resolver := NewResolver(5*time.Minute, "")

	_, err := resolver.Resolve(context.Background(), []string{"invalid"})
	if err == nil {
		t.Fatal("expected error for invalid pack reference")
	}
	if !strings.Contains(err.Error(), "invalid pack reference") {
		t.Errorf("unexpected error: %v", err)
	}
}
