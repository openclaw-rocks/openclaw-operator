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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// BuildConfigMap creates a ConfigMap for the OpenClawInstance configuration.
// It injects gateway.bind (lan or loopback depending on Tailscale mode) and
// optionally injects gateway.auth credentials when gatewayToken is non-empty.
// Uses the inline raw config from the instance spec as the base.
func BuildConfigMap(instance *openclawv1alpha1.OpenClawInstance, gatewayToken string) *corev1.ConfigMap {
	// Start with empty config, overlay raw config if present
	configBytes := []byte("{}")
	if instance.Spec.Config.Raw != nil && len(instance.Spec.Config.Raw.Raw) > 0 {
		configBytes = instance.Spec.Config.Raw.Raw
	}

	return BuildConfigMapFromBytes(instance, configBytes, gatewayToken)
}

// BuildConfigMapFromBytes creates a ConfigMap for the OpenClawInstance using
// the provided base config bytes. This allows the controller to pass config
// from any source (inline raw, external ConfigMap, or empty default).
// The enrichment pipeline (gateway auth, tailscale, browser, gateway bind)
// always runs on the provided bytes.
func BuildConfigMapFromBytes(instance *openclawv1alpha1.OpenClawInstance, baseConfig []byte, gatewayToken string) *corev1.ConfigMap {
	labels := Labels(instance)

	configBytes := baseConfig
	if len(configBytes) == 0 {
		configBytes = []byte("{}")
	}

	// Enrichment pipeline: gateway auth -> tailscale -> browser -> gateway bind
	if gatewayToken != "" {
		if enriched, err := enrichConfigWithGatewayAuth(configBytes, gatewayToken); err == nil {
			configBytes = enriched
		}
	}
	if instance.Spec.Tailscale.Enabled {
		if enriched, err := enrichConfigWithTailscale(configBytes, instance); err == nil {
			configBytes = enriched
		}
	}
	if instance.Spec.Chromium.Enabled {
		if enriched, err := enrichConfigWithBrowser(configBytes); err == nil {
			configBytes = enriched
		}
	}
	if enriched, err := enrichConfigWithGatewayBind(configBytes, instance); err == nil {
		configBytes = enriched
	}

	configContent := string(configBytes)

	// Try to pretty-print the JSON
	var parsed interface{}
	if err := json.Unmarshal(configBytes, &parsed); err == nil {
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

	auth["mode"] = "token" //nolint:goconst // OpenClaw auth mode, not k8s Secret key
	auth["token"] = token
	gw["auth"] = auth
	config["gateway"] = gw

	return json.Marshal(config)
}

// enrichConfigWithTailscale injects gateway.tailscale settings into the config JSON.
// Sets gateway.tailscale.mode and gateway.tailscale.resetOnExit.
// If authSSO is enabled, also sets gateway.auth.allowTailscale=true.
// Does not override user-set values.
func enrichConfigWithTailscale(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	// Set tailscale config (respect user overrides)
	ts, _ := gw["tailscale"].(map[string]interface{})
	if ts == nil {
		ts = make(map[string]interface{})
	}
	if _, ok := ts["mode"]; !ok {
		mode := instance.Spec.Tailscale.Mode
		if mode == "" {
			mode = TailscaleModeServe
		}
		ts["mode"] = mode
	}
	if _, ok := ts["resetOnExit"]; !ok {
		ts["resetOnExit"] = true
	}
	gw["tailscale"] = ts

	// Set gateway.auth.allowTailscale when AuthSSO is enabled
	if instance.Spec.Tailscale.AuthSSO {
		auth, _ := gw["auth"].(map[string]interface{})
		if auth == nil {
			auth = make(map[string]interface{})
		}
		if _, ok := auth["allowTailscale"]; !ok {
			auth["allowTailscale"] = true
		}
		gw["auth"] = auth
	}

	config["gateway"] = gw
	return json.Marshal(config)
}

// enrichConfigWithBrowser injects browser config into the config JSON so the
// agent uses the Chromium sidecar instead of the Chrome extension relay.
// Configures both "default" and "chrome" profiles to point at the sidecar CDP
// port. The "chrome" profile must be redirected because LLMs frequently pass
// profile="chrome" explicitly in browser tool calls, bypassing defaultProfile.
// Without this override the built-in "chrome" profile falls back to the
// extension relay which does not work in a headless container.
// Does not override user-set values.
func enrichConfigWithBrowser(configJSON []byte) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	browser, _ := config["browser"].(map[string]interface{})
	if browser == nil {
		browser = make(map[string]interface{})
	}

	// Set defaultProfile to "default" if not already set
	if _, ok := browser["defaultProfile"]; !ok {
		browser["defaultProfile"] = "default"
	}

	profiles, _ := browser["profiles"].(map[string]interface{})
	if profiles == nil {
		profiles = make(map[string]interface{})
	}

	cdpURL := fmt.Sprintf("http://localhost:%d", BrowserlessCDPPort)

	// Configure both "default" and "chrome" profiles to point at the sidecar.
	// LLMs often explicitly pass profile="chrome", so we redirect it to the
	// sidecar CDP endpoint instead of the extension relay.
	for _, profileName := range []string{"default", "chrome"} {
		profile, _ := profiles[profileName].(map[string]interface{})
		if profile == nil {
			profile = make(map[string]interface{})
		}

		// Only set cdpUrl if the user hasn't configured cdpUrl or cdpPort
		if _, hasURL := profile["cdpUrl"]; !hasURL {
			if _, hasPort := profile["cdpPort"]; !hasPort {
				profile["cdpUrl"] = cdpURL
			}
		}

		// color is required by OpenClaw's config validation
		if _, hasColor := profile["color"]; !hasColor {
			profile["color"] = "#4285F4"
		}

		profiles[profileName] = profile
	}

	browser["profiles"] = profiles
	config["browser"] = browser

	return json.Marshal(config)
}

// enrichConfigWithGatewayBind injects gateway.bind into the config JSON.
// When Tailscale serve/funnel is active, sets bind=loopback (required by
// the OpenClaw gateway for Tailscale modes). Otherwise sets bind=lan so
// the gateway listens on the pod IP (required for TCPSocket health probes).
// If the user has already set gateway.bind, the config is returned
// unchanged (user override wins).
func enrichConfigWithGatewayBind(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	// If the user already set bind, don't override
	if _, ok := gw["bind"]; ok {
		return configJSON, nil
	}

	bindValue := GatewayBindLAN
	if IsTailscaleServeOrFunnel(instance) {
		bindValue = GatewayBindLoopback
	}

	gw["bind"] = bindValue
	config["gateway"] = gw

	return json.Marshal(config)
}
