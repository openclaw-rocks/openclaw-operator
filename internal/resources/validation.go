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
	"fmt"
	"strings"
)

// ValidateWorkspaceFilename checks a single workspace filename.
// Exported so both the webhook and the controller can validate filenames
// (e.g. keys from an external ConfigMap referenced by spec.workspace.configMapRef).
func ValidateWorkspaceFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename must not be empty")
	}
	if len(name) > 253 {
		return fmt.Errorf("filename must be at most 253 characters")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("filename must not contain '/'")
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("filename must not contain '\\'")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("filename must not contain '..'")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("filename must not start with '.'")
	}
	if name == "openclaw.json" {
		return fmt.Errorf("filename 'openclaw.json' is reserved for config")
	}
	return nil
}

// ValidateWorkspaceDirectory checks a single workspace directory path.
// Exported so both the webhook and the controller can validate directory names.
func ValidateWorkspaceDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}
	if len(dir) > 253 {
		return fmt.Errorf("directory must be at most 253 characters")
	}
	if strings.Contains(dir, "\\") {
		return fmt.Errorf("directory must not contain '\\'")
	}
	if strings.Contains(dir, "..") {
		return fmt.Errorf("directory must not contain '..'")
	}
	if strings.HasPrefix(dir, "/") {
		return fmt.Errorf("directory must not be an absolute path")
	}
	return nil
}
