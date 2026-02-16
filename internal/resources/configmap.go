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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// BuildConfigMap creates a ConfigMap for the OpenClawInstance configuration.
// If gatewayToken is non-empty and the user hasn't already configured
// gateway.auth.token in their config, the token and auth mode are injected.
func BuildConfigMap(instance *openclawv1alpha1.OpenClawInstance, gatewayToken string) *corev1.ConfigMap {
	labels := Labels(instance)

	// Generate openclaw.json content from raw config
	configContent := "{}"
	if instance.Spec.Config.Raw != nil && len(instance.Spec.Config.Raw.Raw) > 0 {
		configBytes := instance.Spec.Config.Raw.Raw
		// Pre-enable modules for configured channels so the gateway does not
		// need to modify the config file on startup (avoids EBUSY on rename).
		if enriched, err := enrichConfigWithModules(configBytes); err == nil {
			configBytes = enriched
		}
		// Inject gateway auth token so mDNS pairing is not required.
		if gatewayToken != "" {
			if enriched, err := enrichConfigWithGatewayAuth(configBytes, gatewayToken); err == nil {
				configBytes = enriched
			}
		}
		configContent = string(configBytes)
	} else if gatewayToken != "" {
		// No raw config at all — create a minimal config with gateway auth
		configContent = `{"gateway":{"auth":{"mode":"token","token":"` + gatewayToken + `"}}}`
	}

	// Try to pretty-print the JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(configContent), &parsed); err == nil {
		if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
			configContent = string(pretty)
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName(instance),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"openclaw.json": configContent,
		},
	}
}

// enrichConfigWithModules detects configured channels and ensures their
// corresponding module entries are present in the modules array. This prevents
// the gateway from needing to auto-enable modules on startup, which causes
// EBUSY errors on atomic rename with certain storage backends (e.g. Longhorn).
func enrichConfigWithModules(configJSON []byte) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	channels, ok := config["channels"].(map[string]interface{})
	if !ok || len(channels) == 0 {
		return configJSON, nil
	}

	// Get or create modules array
	var modules []interface{}
	if existing, ok := config["modules"].([]interface{}); ok {
		modules = existing
	}

	// Index existing module locations
	locationIndex := make(map[string]int) // location -> index in modules
	for i, mod := range modules {
		if m, ok := mod.(map[string]interface{}); ok {
			if loc, ok := m["location"].(string); ok {
				locationIndex[loc] = i
			}
		}
	}

	changed := false
	for name, channelCfg := range channels {
		cm, ok := channelCfg.(map[string]interface{})
		if !ok {
			continue
		}
		enabled, _ := cm["enabled"].(bool)
		if !enabled {
			continue
		}

		location := "MODULES_ROOT/channel-" + name
		if idx, exists := locationIndex[location]; exists {
			// Ensure the existing entry is enabled
			if m, ok := modules[idx].(map[string]interface{}); ok {
				if e, _ := m["enabled"].(bool); !e {
					m["enabled"] = true
					changed = true
				}
			}
		} else {
			modules = append(modules, map[string]interface{}{
				"location": location,
				"enabled":  true,
			})
			changed = true
		}
	}

	if !changed {
		return configJSON, nil
	}

	config["modules"] = modules
	return json.Marshal(config)
}

// enrichConfigWithGatewayAuth injects gateway.auth.mode=token and
// gateway.auth.token into the config JSON. If the user has already set
// gateway.auth.token, the config is returned unchanged (user override wins).
func enrichConfigWithGatewayAuth(configJSON []byte, token string) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	// Navigate into gateway.auth, creating intermediate maps as needed
	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}
	auth, _ := gw["auth"].(map[string]interface{})
	if auth == nil {
		auth = make(map[string]interface{})
	}

	// If the user already set a token, don't override
	if existingToken, ok := auth["token"].(string); ok && existingToken != "" {
		return configJSON, nil
	}

	auth["mode"] = "token"
	auth["token"] = token
	gw["auth"] = auth
	config["gateway"] = gw

	return json.Marshal(config)
}
