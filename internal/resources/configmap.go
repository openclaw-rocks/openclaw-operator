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
// It always sets gateway.bind=loopback (the proxy sidecar handles external
// access) and optionally injects gateway.auth credentials when gatewayToken
// is non-empty. Also includes the nginx stream config for the proxy sidecar.
// Uses the inline raw config from the instance spec as the base.
func BuildConfigMap(instance *openclawv1alpha1.OpenClawInstance, gatewayToken string, skillPacks *ResolvedSkillPacks) *corev1.ConfigMap {
	// Start with empty config, overlay raw config if present
	configBytes := []byte("{}")
	if instance.Spec.Config.Raw != nil && len(instance.Spec.Config.Raw.Raw) > 0 {
		configBytes = instance.Spec.Config.Raw.Raw
	}

	return BuildConfigMapFromBytes(instance, configBytes, gatewayToken, skillPacks)
}

// BuildConfigMapFromBytes creates a ConfigMap for the OpenClawInstance using
// the provided base config bytes. This allows the controller to pass config
// from any source (inline raw, external ConfigMap, or empty default).
// The enrichment pipeline (gateway auth, tailscale, browser, gateway bind)
// always runs on the provided bytes.
func BuildConfigMapFromBytes(instance *openclawv1alpha1.OpenClawInstance, baseConfig []byte, gatewayToken string, skillPacks *ResolvedSkillPacks) *corev1.ConfigMap {
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
	if skillPacks != nil && len(skillPacks.SkillEntries) > 0 {
		if enriched, err := enrichConfigWithSkillPacks(configBytes, skillPacks.SkillEntries); err == nil {
			configBytes = enriched
		}
	}

	configContent := string(configBytes)

	// Try to pretty-print the JSON
	var parsed interface{}
	if err := json.Unmarshal(configBytes, &parsed); err == nil {
		if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
			configContent = string(pretty)
		}
	}

	data := map[string]string{
		"openclaw.json": configContent,
		NginxConfigKey:  nginxStreamConfig(),
	}

	// Add Tailscale serve config when enabled (sidecar reads this via TS_SERVE_CONFIG)
	if instance.Spec.Tailscale.Enabled {
		data[TailscaleServeConfigKey] = BuildTailscaleServeConfig(instance)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName(instance),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
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

// enrichConfigWithTailscale injects Tailscale-related settings into the config JSON.
// The Tailscale sidecar handles serve/funnel declaratively via TS_SERVE_CONFIG,
// so we no longer set gateway.tailscale.mode or gateway.tailscale.resetOnExit.
// If authSSO is enabled, sets gateway.auth.allowTailscale=true so the main
// container accepts tailnet-authenticated requests.
// Does not override user-set values.
func enrichConfigWithTailscale(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
	// Only need to inject config when AuthSSO is enabled
	if !instance.Spec.Tailscale.AuthSSO {
		return configJSON, nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	// Set gateway.auth.allowTailscale when AuthSSO is enabled
	auth, _ := gw["auth"].(map[string]interface{})
	if auth == nil {
		auth = make(map[string]interface{})
	}
	if _, ok := auth["allowTailscale"]; !ok {
		auth["allowTailscale"] = true
	}
	gw["auth"] = auth

	config["gateway"] = gw
	return json.Marshal(config)
}

// tailscaleServeConfig is the JSON structure for TS_SERVE_CONFIG.
// The sidecar reads this to declaratively configure serve or funnel.
type tailscaleServeConfig struct {
	TCP map[string]*tailscaleTCPHandler `json:"TCP"`
	Web map[string]*tailscaleWebConfig  `json:"Web,omitempty"`
	// AllowFunnel controls whether Tailscale Funnel (public internet) is enabled
	AllowFunnel map[string]bool `json:"AllowFunnel,omitempty"`
}

type tailscaleTCPHandler struct {
	HTTPS bool `json:"HTTPS"`
}

type tailscaleWebConfig struct {
	Handlers map[string]*tailscaleWebHandler `json:"Handlers"`
}

type tailscaleWebHandler struct {
	Proxy string `json:"Proxy"`
}

// BuildTailscaleServeConfig generates the TS_SERVE_CONFIG JSON for the sidecar.
// It proxies HTTPS traffic to the gateway on 127.0.0.1:GatewayPort.
// In funnel mode, AllowFunnel is set to expose the instance publicly.
func BuildTailscaleServeConfig(instance *openclawv1alpha1.OpenClawInstance) string {
	proxy := fmt.Sprintf("http://127.0.0.1:%d", GatewayPort)

	cfg := tailscaleServeConfig{
		TCP: map[string]*tailscaleTCPHandler{
			"443": {HTTPS: true},
		},
		Web: map[string]*tailscaleWebConfig{
			"${TS_CERT_DOMAIN}:443": {
				Handlers: map[string]*tailscaleWebHandler{
					"/": {Proxy: proxy},
				},
			},
		},
	}

	mode := instance.Spec.Tailscale.Mode
	if mode == "" {
		mode = TailscaleModeServe
	}
	if mode == TailscaleModeFunnel {
		cfg.AllowFunnel = map[string]bool{
			"${TS_CERT_DOMAIN}:443": true,
		}
	}

	data, _ := json.Marshal(cfg)
	return string(data)
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

	// Use ${OPENCLAW_CHROMIUM_CDP} env var (resolved at runtime by OpenClaw)
	// which contains the pod IP. The browser control service treats any
	// 127.x.x.x address as "local/managed" and tries to bind the port.
	// A non-loopback pod IP activates remote/attach-only mode so the
	// service just connects to the existing chromium sidecar.
	cdpURL := "${OPENCLAW_CHROMIUM_CDP}"

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

// enrichConfigWithGatewayBind injects gateway.bind=loopback into the config
// JSON. The gateway proxy sidecar handles external access, so the gateway
// process always binds to loopback. If the user has already set gateway.bind,
// the config is returned unchanged (user override wins).
func enrichConfigWithGatewayBind(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
	_ = instance // signature kept for enrichment pipeline consistency
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

	gw["bind"] = GatewayBindLoopback
	config["gateway"] = gw

	return json.Marshal(config)
}

// enrichConfigWithSkillPacks injects skills.entries from resolved skill packs
// into the config JSON. Skill pack entries are set first, then any existing
// user-defined entries are overlaid, so user overrides always win.
func enrichConfigWithSkillPacks(configJSON []byte, skillEntries map[string]interface{}) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil
	}

	skills, _ := config["skills"].(map[string]interface{})
	if skills == nil {
		skills = make(map[string]interface{})
	}

	entries, _ := skills["entries"].(map[string]interface{})
	if entries == nil {
		entries = make(map[string]interface{})
	}

	// Set skill pack entries (only if user hasn't already set them)
	for name, value := range skillEntries {
		if _, exists := entries[name]; !exists {
			entries[name] = value
		}
	}

	skills["entries"] = entries
	config["skills"] = skills

	return json.Marshal(config)
}

// nginxStreamConfig returns the static nginx stream configuration for the
// gateway reverse proxy sidecar. It proxies external traffic on dedicated
// ports to the gateway and canvas processes listening on loopback.
func nginxStreamConfig() string {
	return fmt.Sprintf(`worker_processes 1;
pid /tmp/nginx.pid;
error_log /dev/stderr warn;

events {
    worker_connections 128;
}

stream {
    server {
        listen 0.0.0.0:%d;
        proxy_pass 127.0.0.1:%d;
    }
    server {
        listen 0.0.0.0:%d;
        proxy_pass 127.0.0.1:%d;
    }
}
`, GatewayProxyPort, GatewayPort, CanvasProxyPort, CanvasPort)
}
