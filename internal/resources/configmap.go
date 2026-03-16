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
	"net/url"
	"sort"

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
// The enrichment pipeline (gateway auth, device auth, tailscale, browser,
// gateway bind, skill packs) always runs on the provided bytes.
func BuildConfigMapFromBytes(instance *openclawv1alpha1.OpenClawInstance, baseConfig []byte, gatewayToken string, skillPacks *ResolvedSkillPacks) *corev1.ConfigMap {
	labels := Labels(instance)

	configBytes := baseConfig
	if len(configBytes) == 0 {
		configBytes = []byte("{}")
	}

	// Enrichment pipeline: gateway auth -> device auth -> tailscale -> browser -> gateway bind -> trusted proxies -> control UI origins -> skill packs
	if gatewayToken != "" {
		if enriched, err := enrichConfigWithGatewayAuth(configBytes, gatewayToken); err == nil {
			configBytes = enriched
		}
	}
	if enriched, err := enrichConfigWithDeviceAuth(configBytes); err == nil {
		configBytes = enriched
	}
	if instance.Spec.Tailscale.Enabled {
		if enriched, err := enrichConfigWithTailscale(configBytes, instance); err == nil {
			configBytes = enriched
		}
	}
	if instance.Spec.Chromium.Enabled {
		if enriched, err := enrichConfigWithBrowser(configBytes, instance); err == nil {
			configBytes = enriched
		}
	}
	if enriched, err := enrichConfigWithGatewayBind(configBytes, instance); err == nil {
		configBytes = enriched
	}
	if enriched, err := enrichConfigWithTrustedProxies(configBytes); err == nil {
		configBytes = enriched
	}
	if enriched, err := enrichConfigWithControlUIOrigins(configBytes, instance); err == nil {
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

	// Add chromium CDP proxy nginx config when enabled
	if instance.Spec.Chromium.Enabled {
		data[ChromiumProxyNginxConfigKey] = chromiumProxyNginxConfig(instance)
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

// enrichConfigWithDeviceAuth injects gateway.controlUi.dangerouslyDisableDeviceAuth=true
// into the config JSON. Device pairing is fundamentally incompatible with Kubernetes
// because (1) users cannot approve pairing from inside a container, (2) connections
// always come through the nginx proxy sidecar (non-local), and (3) mDNS does not work
// in K8s. If the user has already set the field, the config is returned unchanged.
func enrichConfigWithDeviceAuth(configJSON []byte) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	controlUI, _ := gw["controlUi"].(map[string]interface{})
	if controlUI == nil {
		controlUI = make(map[string]interface{})
	}

	// If the user already set dangerouslyDisableDeviceAuth, don't override
	if _, ok := controlUI["dangerouslyDisableDeviceAuth"]; ok {
		return configJSON, nil
	}

	controlUI["dangerouslyDisableDeviceAuth"] = true
	gw["controlUi"] = controlUI
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
//
// Key settings injected:
//   - attachOnly=true: skips local browser binary detection. Without this,
//     OpenClaw checks for a local Chrome/Chromium binary first, fails with
//     "No supported browser found", and never attempts the remote CDP connection.
//   - remoteCdpTimeoutMs=30000: gives the browser service time to become ready
//     on startup, avoiding permanent failure when tool registration wins the
//     race against browser service initialization.
//   - cdpUrl on "default" and "chrome" profiles: resolved at config build time
//     to the headless CDP Service DNS name (not an env var reference).
//
// The "chrome" profile must be redirected because LLMs frequently pass
// profile="chrome" explicitly in browser tool calls, bypassing defaultProfile.
// Without this override the built-in "chrome" profile falls back to the
// extension relay which does not work in a headless container.
// Does not override user-set values.
func enrichConfigWithBrowser(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
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

	// attachOnly=true skips local browser binary detection. In a container
	// there is no local Chrome binary - without this flag OpenClaw fails with
	// "No supported browser found" and never attempts the remote CDP connection.
	if _, ok := browser["attachOnly"]; !ok {
		browser["attachOnly"] = true
	}

	// remoteCdpTimeoutMs gives the browser service time to become ready.
	// OpenClaw's tool registration can fire before the browser service is
	// fully initialized. Without a timeout, the failure is cached permanently
	// for the pod's lifetime.
	if _, ok := browser["remoteCdpTimeoutMs"]; !ok {
		browser["remoteCdpTimeoutMs"] = float64(30000)
	}

	profiles, _ := browser["profiles"].(map[string]interface{})
	if profiles == nil {
		profiles = make(map[string]interface{})
	}

	// Build the resolved CDP URL at config generation time using the headless
	// CDP Service DNS name. Previous versions used ${OPENCLAW_CHROMIUM_CDP}
	// env var interpolation, but this proved unreliable across OpenClaw versions.
	cdpURL := fmt.Sprintf("http://%s.%s.svc:%d",
		ChromiumCDPServiceName(instance), instance.Namespace, ChromiumPort)

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

// enrichConfigWithTrustedProxies ensures 127.0.0.0/8 is present in
// gateway.trustedProxies. The nginx proxy sidecar forwards all traffic from
// 127.0.0.1, so the gateway must trust that CIDR to honor proxy headers
// (X-Forwarded-For, etc.). Unlike other enrichments this merges with any
// user-supplied entries rather than skipping when the field is already set.
func enrichConfigWithTrustedProxies(configJSON []byte) ([]byte, error) {
	const loopbackCIDR = "127.0.0.0/8"

	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	// Read existing trustedProxies (may be user-set)
	var proxies []interface{}
	if existing, ok := gw["trustedProxies"].([]interface{}); ok {
		proxies = existing
	}

	// Check if loopback CIDR is already present
	for _, p := range proxies {
		if s, ok := p.(string); ok && s == loopbackCIDR {
			return configJSON, nil
		}
	}

	proxies = append(proxies, loopbackCIDR)
	gw["trustedProxies"] = proxies
	config["gateway"] = gw

	return json.Marshal(config)
}

// enrichConfigWithControlUIOrigins injects gateway.controlUi.allowedOrigins
// into the config JSON. Origins are derived from localhost (always), ingress
// hosts (scheme from TLS config), and spec.gateway.controlUiOrigins (explicit).
// If the user has already set gateway.controlUi.allowedOrigins, the config is
// returned unchanged (user override wins).
func enrichConfigWithControlUIOrigins(configJSON []byte, instance *openclawv1alpha1.OpenClawInstance) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // not a JSON object, return unchanged
	}

	gw, _ := config["gateway"].(map[string]interface{})
	if gw == nil {
		gw = make(map[string]interface{})
	}

	controlUI, _ := gw["controlUi"].(map[string]interface{})
	if controlUI == nil {
		controlUI = make(map[string]interface{})
	}

	// If the user already set allowedOrigins, don't override
	if _, ok := controlUI["allowedOrigins"]; ok {
		return configJSON, nil
	}

	origins := deriveControlUIOrigins(instance)
	if len(origins) == 0 {
		return configJSON, nil
	}

	// Convert to []interface{} for JSON marshaling
	originsIface := make([]interface{}, len(origins))
	for i, o := range origins {
		originsIface[i] = o
	}

	controlUI["allowedOrigins"] = originsIface
	gw["controlUi"] = controlUI
	config["gateway"] = gw

	return json.Marshal(config)
}

// deriveControlUIOrigins builds a deduplicated, sorted list of origins from:
// 1. Localhost (always): http://localhost:18789, http://127.0.0.1:18789
// 2. Ingress hosts: https:// if host appears in TLS config, http:// otherwise
// 3. CRD field: spec.gateway.controlUiOrigins (explicit extras)
func deriveControlUIOrigins(instance *openclawv1alpha1.OpenClawInstance) []string {
	seen := make(map[string]struct{})
	var origins []string

	add := func(origin string) {
		if _, exists := seen[origin]; !exists {
			seen[origin] = struct{}{}
			origins = append(origins, origin)
		}
	}

	// Always include localhost origins for port-forwarding
	add(fmt.Sprintf("http://localhost:%d", GatewayPort))
	add(fmt.Sprintf("http://127.0.0.1:%d", GatewayPort))

	// Build TLS host lookup for scheme determination
	tlsHosts := make(map[string]struct{})
	for _, tls := range instance.Spec.Networking.Ingress.TLS {
		for _, h := range tls.Hosts {
			tlsHosts[h] = struct{}{}
		}
	}

	// Derive origins from ingress hosts
	for _, ingressHost := range instance.Spec.Networking.Ingress.Hosts {
		host := ingressHost.Host
		if host == "" {
			continue
		}
		scheme := "http"
		if _, isTLS := tlsHosts[host]; isTLS {
			scheme = "https"
		}
		add(fmt.Sprintf("%s://%s", scheme, host))
	}

	// Add explicit origins from CRD field
	for _, origin := range instance.Spec.Gateway.ControlUIOrigins {
		if origin != "" {
			add(origin)
		}
	}

	sort.Strings(origins)
	return origins
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

// chromiumProxyNginxConfig returns the nginx HTTP configuration for the
// chromium CDP proxy sidecar. It sits between OpenClaw and the browserless
// sidecar, routing WebSocket connections to the /chromium endpoint with
// Chrome launch args (anti-bot flags + user ExtraArgs) via the `launch`
// query parameter.
//
// Why /chromium specifically: browserless v2 deprecated DEFAULT_LAUNCH_ARGS
// and only applies the `launch` query parameter on its /chromium WebSocket
// endpoint (which launches a new Chrome process). Other endpoints like
// /devtools/browser/<id> connect to already-running Chrome and ignore
// launch args entirely. By routing all WebSocket upgrades to /chromium,
// every browser session gets fresh Chrome with the correct flags.
//
// HTTP requests (health checks, /json/version) pass through unchanged.
func chromiumProxyNginxConfig(instance *openclawv1alpha1.OpenClawInstance) string {
	args := deduplicateArgs(DefaultChromiumLaunchArgs, instance.Spec.Chromium.ExtraArgs)

	launchJSON, _ := json.Marshal(map[string]interface{}{"args": args})
	encoded := url.QueryEscape(string(launchJSON))

	return fmt.Sprintf(`worker_processes 1;
pid /tmp/nginx.pid;
error_log /dev/stderr warn;

events {
    worker_connections 64;
}

http {
    # Redirect temp/cache dirs to writable /tmp (rootfs is read-only)
    client_body_temp_path /tmp/client_body;
    proxy_temp_path /tmp/proxy;
    fastcgi_temp_path /tmp/fastcgi;
    uwsgi_temp_path /tmp/uwsgi;
    scgi_temp_path /tmp/scgi;

    server {
        listen 0.0.0.0:%d;

        # WebSocket connections route to /chromium with launch args.
        # browserless v2 only applies launch args on the /chromium endpoint
        # (launches new Chrome), not /devtools/browser/ (existing Chrome).
        location @chromium_ws {
            rewrite ^ /chromium?launch=%s break;
            proxy_pass http://127.0.0.1:%d;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_buffering off;
            proxy_read_timeout 86400s;
            proxy_send_timeout 86400s;
        }

        location / {
            # Route WebSocket upgrades to @chromium_ws via internal redirect.
            error_page 418 = @chromium_ws;
            if ($http_upgrade ~* "websocket") {
                return 418;
            }

            # HTTP requests pass through to browserless unchanged.
            proxy_pass http://127.0.0.1:%d;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_buffering off;
        }
    }
}
`, ChromiumPort, encoded, BrowserlessInternalPort, BrowserlessInternalPort)
}

// deduplicateArgs merges default and extra Chrome launch args, removing
// duplicates. When the same flag appears in both lists, the extra (user)
// value wins. Flags are compared by their key (everything before '=').
func deduplicateArgs(defaults, extras []string) []string {
	seen := make(map[string]int) // flag key → index in result
	result := make([]string, 0, len(defaults)+len(extras))

	for _, arg := range defaults {
		key := argKey(arg)
		seen[key] = len(result)
		result = append(result, arg)
	}
	for _, arg := range extras {
		key := argKey(arg)
		if idx, ok := seen[key]; ok {
			result[idx] = arg // user value overrides default
		} else {
			seen[key] = len(result)
			result = append(result, arg)
		}
	}
	return result
}

// argKey extracts the flag key from a Chrome arg for deduplication.
// "--user-agent=Foo" → "--user-agent", "--no-sandbox" → "--no-sandbox".
func argKey(arg string) string {
	for i, c := range arg {
		if c == '=' {
			return arg[:i]
		}
	}
	return arg
}
