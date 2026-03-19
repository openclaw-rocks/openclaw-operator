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

package e2e

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
	"github.com/openclawrocks/k8s-operator/internal/resources"
)

// cdpCommand represents a Chrome DevTools Protocol command sent over WebSocket.
type cdpCommand struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// cdpResponse represents a Chrome DevTools Protocol response received over WebSocket.
type cdpResponse struct {
	ID     int                    `json:"id"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  map[string]interface{} `json:"error,omitempty"`
	Method string                 `json:"method,omitempty"`
}

// cdpSessionCommand represents a CDP command sent to a specific target session.
type cdpSessionCommand struct {
	ID        int                    `json:"id"`
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params,omitempty"`
	SessionID string                 `json:"sessionId"`
}

var _ = Describe("Chromium CDP Functional Tests", Ordered, func() {
	var (
		namespace    string
		instanceName string
		localPort    int
		portFwdCmd   *exec.Cmd
		podName      string
	)

	BeforeAll(func() {
		if os.Getenv("E2E_SKIP_CDP_FUNCTIONAL") == "true" {
			Skip("Skipping CDP functional tests (E2E_SKIP_CDP_FUNCTIONAL=true)")
		}
		if os.Getenv("E2E_SKIP_RESOURCE_VALIDATION") == "true" {
			Skip("Skipping CDP functional tests in minimal mode")
		}

		instanceName = "cdp-func-test"
		namespace = "test-cdp-" + time.Now().Format("20060102150405")

		By("Creating test namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		By("Creating OpenClawInstance with chromium enabled")
		instance := &openclawv1alpha1.OpenClawInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: namespace,
				Annotations: map[string]string{
					"openclaw.rocks/skip-backup": "true",
				},
			},
			Spec: openclawv1alpha1.OpenClawInstanceSpec{
				Image: openclawv1alpha1.ImageSpec{
					Repository: "ghcr.io/openclaw/openclaw",
					Tag:        "latest",
				},
				Chromium: openclawv1alpha1.ChromiumSpec{
					Enabled: true,
				},
			},
		}
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

		By("Waiting for StatefulSet to be created")
		sts := &appsv1.StatefulSet{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      resources.StatefulSetName(instance),
				Namespace: namespace,
			}, sts)
		}, 60*time.Second, 2*time.Second).Should(Succeed())

		By("Waiting for pod to exist")
		Eventually(func() string {
			podList := &corev1.PodList{}
			err := k8sClient.List(ctx, podList,
				client.InNamespace(namespace),
				client.MatchingLabels{
					"app.kubernetes.io/instance": instanceName,
					"app.kubernetes.io/name":     "openclaw",
				},
			)
			if err != nil || len(podList.Items) == 0 {
				return ""
			}
			podName = podList.Items[0].Name
			return podName
		}, 120*time.Second, 3*time.Second).ShouldNot(BeEmpty())

		By("Waiting for pod to be in Running phase with chromium init container ready")
		Eventually(func() bool {
			pod := &corev1.Pod{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      podName,
				Namespace: namespace,
			}, pod)
			if err != nil {
				return false
			}
			if pod.Status.Phase != corev1.PodRunning {
				GinkgoWriter.Printf("Pod phase: %s (waiting for Running)\n", pod.Status.Phase)
				return false
			}
			for _, cs := range pod.Status.InitContainerStatuses {
				if cs.Name == "chromium" && cs.Ready {
					return true
				}
			}
			return false
		}, 5*time.Minute, 3*time.Second).Should(BeTrue())

		By("Finding a free local port for port-forward")
		listener, err := net.Listen("tcp", ":0")
		Expect(err).NotTo(HaveOccurred())
		localPort = listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		By(fmt.Sprintf("Starting port-forward to pod %s on local port %d", podName, localPort))
		portFwdCmd = exec.Command("kubectl", "port-forward",
			fmt.Sprintf("pod/%s", podName),
			fmt.Sprintf("%d:%d", localPort, resources.ChromiumPort),
			"-n", namespace,
		)
		portFwdCmd.Stdout = GinkgoWriter
		portFwdCmd.Stderr = GinkgoWriter
		Expect(portFwdCmd.Start()).To(Succeed())

		By("Waiting for port-forward to be ready")
		Eventually(func() error {
			// Check if port-forward process exited unexpectedly
			if portFwdCmd.ProcessState != nil {
				return fmt.Errorf("port-forward process exited: %s", portFwdCmd.ProcessState)
			}
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/version", localPort))
			if err != nil {
				return err
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status: %d", resp.StatusCode)
			}
			return nil
		}, 60*time.Second, 2*time.Second).Should(Succeed())

		GinkgoWriter.Printf("CDP port-forward ready on localhost:%d\n", localPort)
	})

	AfterAll(func() {
		if portFwdCmd != nil && portFwdCmd.Process != nil {
			By("Killing port-forward process")
			_ = portFwdCmd.Process.Kill()
			_ = portFwdCmd.Wait()
		}

		if namespace != "" {
			By("Deleting test namespace")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}
			_ = k8sClient.Delete(ctx, ns)
		}
	})

	It("Tier 1: /json/version endpoint responds with Chrome version info", func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/version", localPort))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		var versionInfo map[string]interface{}
		Expect(json.Unmarshal(body, &versionInfo)).To(Succeed())

		GinkgoWriter.Printf("CDP version info: %s\n", string(body))

		browser, ok := versionInfo["Browser"].(string)
		Expect(ok).To(BeTrue(), "response should have a Browser field")
		Expect(browser).To(SatisfyAny(
			ContainSubstring("HeadlessChrome"),
			ContainSubstring("Chrome"),
		), "Browser field should contain Chrome or HeadlessChrome")

		wsURL, ok := versionInfo["webSocketDebuggerUrl"].(string)
		Expect(ok).To(BeTrue(), "response should have a webSocketDebuggerUrl field")
		Expect(wsURL).NotTo(BeEmpty(), "webSocketDebuggerUrl should not be empty")
	})

	It("Tier 2: navigates to a page and captures screenshot via CDP WebSocket", func() {
		By("Getting browser debugger WebSocket URL from /json/version")
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/json/version", localPort))
		Expect(err).NotTo(HaveOccurred())
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).NotTo(HaveOccurred())

		var versionInfo map[string]interface{}
		Expect(json.Unmarshal(body, &versionInfo)).To(Succeed())

		browserWSURL, ok := versionInfo["webSocketDebuggerUrl"].(string)
		Expect(ok).To(BeTrue(), "response should have webSocketDebuggerUrl")

		// Rewrite the WebSocket URL to use our local port-forward port.
		// Chrome returns ws://127.0.0.1:9222/... but we need ws://localhost:<localPort>/...
		browserWSURL = rewriteCDPWebSocketURL(browserWSURL, localPort)

		By(fmt.Sprintf("Connecting to browser CDP WebSocket at %s", browserWSURL))
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
		browserWS, _, err := dialer.Dial(browserWSURL, nil)
		Expect(err).NotTo(HaveOccurred())
		defer browserWS.Close()

		By("Creating a new target via Target.createTarget")
		createCmd := cdpCommand{
			ID:     1,
			Method: "Target.createTarget",
			Params: map[string]interface{}{
				"url": "about:blank",
			},
		}
		Expect(browserWS.WriteJSON(createCmd)).To(Succeed())

		createResp := readCDPResponse(browserWS, 1, 10*time.Second)
		Expect(createResp).NotTo(BeNil(), "should receive response for Target.createTarget")
		Expect(createResp.Error).To(BeNil(), "Target.createTarget should not return an error")

		targetID, ok := createResp.Result["targetId"].(string)
		Expect(ok).To(BeTrue(), "createTarget result should have targetId")
		GinkgoWriter.Printf("Created target: %s\n", targetID)

		By("Attaching to target via Target.attachToTarget")
		attachCmd := cdpCommand{
			ID:     2,
			Method: "Target.attachToTarget",
			Params: map[string]interface{}{
				"targetId": targetID,
				"flatten":  true,
			},
		}
		Expect(browserWS.WriteJSON(attachCmd)).To(Succeed())

		attachResp := readCDPResponse(browserWS, 2, 10*time.Second)
		Expect(attachResp).NotTo(BeNil(), "should receive response for Target.attachToTarget")
		Expect(attachResp.Error).To(BeNil(), "Target.attachToTarget should not return an error")

		sessionID, ok := attachResp.Result["sessionId"].(string)
		Expect(ok).To(BeTrue(), "attachToTarget result should have sessionId")
		GinkgoWriter.Printf("Attached to session: %s\n", sessionID)

		By("Enabling Page domain")
		enableCmd := cdpSessionCommand{
			ID:        3,
			Method:    "Page.enable",
			SessionID: sessionID,
		}
		Expect(browserWS.WriteJSON(enableCmd)).To(Succeed())

		enableResp := readCDPResponse(browserWS, 3, 10*time.Second)
		Expect(enableResp).NotTo(BeNil(), "should receive response for Page.enable")

		By("Navigating to https://example.com")
		navigateCmd := cdpSessionCommand{
			ID:     4,
			Method: "Page.navigate",
			Params: map[string]interface{}{
				"url": "https://example.com",
			},
			SessionID: sessionID,
		}
		Expect(browserWS.WriteJSON(navigateCmd)).To(Succeed())

		navigateResp := readCDPResponse(browserWS, 4, 30*time.Second)
		Expect(navigateResp).NotTo(BeNil(), "should receive response for Page.navigate")
		Expect(navigateResp.Error).To(BeNil(), "Page.navigate should not return an error")

		By("Waiting for page load")
		waitForLoadEvent(browserWS, 15*time.Second)

		By("Capturing screenshot via Page.captureScreenshot")
		screenshotCmd := cdpSessionCommand{
			ID:     5,
			Method: "Page.captureScreenshot",
			Params: map[string]interface{}{
				"format": "png",
			},
			SessionID: sessionID,
		}
		Expect(browserWS.WriteJSON(screenshotCmd)).To(Succeed())

		screenshotResp := readCDPResponse(browserWS, 5, 15*time.Second)
		Expect(screenshotResp).NotTo(BeNil(), "should receive response for Page.captureScreenshot")
		Expect(screenshotResp.Error).To(BeNil(), "Page.captureScreenshot should not return an error")

		data, ok := screenshotResp.Result["data"].(string)
		Expect(ok).To(BeTrue(), "screenshot result should have a data field")
		Expect(data).NotTo(BeEmpty(), "screenshot data should not be empty")

		GinkgoWriter.Printf("Screenshot captured: %d bytes of base64 PNG data\n", len(data))

		By("Closing the target")
		closeCmd := cdpCommand{
			ID:     6,
			Method: "Target.closeTarget",
			Params: map[string]interface{}{
				"targetId": targetID,
			},
		}
		Expect(browserWS.WriteJSON(closeCmd)).To(Succeed())
		readCDPResponse(browserWS, 6, 5*time.Second)
	})

	It("Tier 3: CDP is reachable via headless Service DNS from within cluster", func() {
		cdpServiceName := resources.ChromiumCDPServiceName(&openclawv1alpha1.OpenClawInstance{
			ObjectMeta: metav1.ObjectMeta{Name: instanceName},
		})
		cdpURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/json/version",
			cdpServiceName, namespace, resources.ChromiumPort,
		)

		// Use a unique pod name to avoid conflicts
		testPodName := fmt.Sprintf("cdp-test-%d", time.Now().UnixNano()%100000)

		// Verify the headless CDP Service has endpoints pointing to the pod,
		// then use curl from a temporary pod to confirm CDP responds via
		// the Service DNS name.
		By("Verifying CDP headless Service has endpoints")
		endpoints := &corev1.Endpoints{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      cdpServiceName,
				Namespace: namespace,
			}, endpoints)
			if err != nil {
				return false
			}
			for _, subset := range endpoints.Subsets {
				if len(subset.Addresses) > 0 {
					return true
				}
			}
			return false
		}, 30*time.Second, 2*time.Second).Should(BeTrue(),
			"CDP headless Service should have at least one endpoint address")

		By(fmt.Sprintf("Running curl from a temporary pod to %s", cdpURL))
		// Chrome's DevTools HTTP server returns 500 for requests with long
		// Host headers (like Kubernetes service DNS names). Override the
		// Host header to localhost so Chrome handles the request correctly.
		cmd := exec.Command("kubectl", "run", testPodName,
			"--rm", "-i",
			"--restart=Never",
			"--timeout=60s",
			"--namespace", namespace,
			"--image=curlimages/curl",
			"--", "curl", "-sf", "--max-time", "10",
			"-H", "Host: localhost",
			cdpURL,
		)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		GinkgoWriter.Printf("kubectl run output: %s\n", outputStr)

		Expect(err).NotTo(HaveOccurred(),
			"curl to CDP service should succeed, output: %s", outputStr)
		Expect(outputStr).To(ContainSubstring("webSocketDebuggerUrl"),
			"response from CDP headless Service should contain webSocketDebuggerUrl")
	})
})

// gwMessage represents a message in the OpenClaw gateway WebSocket protocol.
type gwMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Event   string          `json:"event,omitempty"`
	Method  string          `json:"method,omitempty"`
	OK      *bool           `json:"ok,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// gwConnectPayload is used to extract session defaults from the connect response.
type gwConnectPayload struct {
	Snapshot struct {
		SessionDefaults struct {
			MainSessionKey string `json:"mainSessionKey"`
		} `json:"sessionDefaults"`
	} `json:"snapshot"`
}

// gwChatPayload represents the payload of a chat event from the gateway.
type gwChatPayload struct {
	State      string `json:"state"`
	StopReason string `json:"stopReason,omitempty"`
}

// gwToolResultPayload represents the payload of a tool.result event.
type gwToolResultPayload struct {
	Tool   string                 `json:"tool"`
	Result map[string]interface{} `json:"result,omitempty"`
}

// gwDevicePairRequestPayload represents a device.pair.requested event payload.
type gwDevicePairRequestPayload struct {
	RequestID string `json:"requestId"`
}

// randomHex generates a random hex string suitable for use as a request ID.
func randomHex() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

var _ = Describe("Chromium Full Integration Tests", Ordered, func() {
	var (
		namespace    string
		instanceName string
		localPort    int
		portFwdCmd   *exec.Cmd
		podName      string
	)

	BeforeAll(func() {
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			Skip("Skipping full integration tests (OPENROUTER_API_KEY not set)")
		}

		instanceName = "cdp-integration"
		namespace = "test-cdp-int-" + time.Now().Format("20060102150405")

		By("Creating test namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		By("Creating OpenClawInstance with chromium and OpenRouter config")
		instance := &openclawv1alpha1.OpenClawInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: namespace,
				Annotations: map[string]string{
					"openclaw.rocks/skip-backup": "true",
				},
			},
			Spec: openclawv1alpha1.OpenClawInstanceSpec{
				Image: openclawv1alpha1.ImageSpec{
					Repository: "ghcr.io/openclaw/openclaw",
					Tag:        "latest",
				},
				Chromium: openclawv1alpha1.ChromiumSpec{
					Enabled: true,
				},
				Env: []corev1.EnvVar{
					{
						Name:  "OPENROUTER_API_KEY",
						Value: apiKey,
					},
				},
			},
		}
		instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
			RawExtension: runtime.RawExtension{Raw: []byte(`{
				"models": {
					"providers": {
						"openrouter": {
							"baseUrl": "https://openrouter.ai/api/v1",
							"apiKey": "${OPENROUTER_API_KEY}",
							"api": "openai-completions",
							"models": [
								{
									"id": "deepseek/deepseek-chat",
									"name": "DeepSeek V3",
									"contextWindow": 163840
								}
							]
						}
					}
				},
				"agents": {
					"defaults": {
						"model": {
							"primary": "openrouter/deepseek/deepseek-chat"
						}
					}
				}
			}`)},
		}
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

		By("Waiting for StatefulSet to be created")
		sts := &appsv1.StatefulSet{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      resources.StatefulSetName(instance),
				Namespace: namespace,
			}, sts)
		}, 60*time.Second, 2*time.Second).Should(Succeed())

		By("Waiting for pod to exist")
		Eventually(func() string {
			podList := &corev1.PodList{}
			err := k8sClient.List(ctx, podList,
				client.InNamespace(namespace),
				client.MatchingLabels{
					"app.kubernetes.io/instance": instanceName,
					"app.kubernetes.io/name":     "openclaw",
				},
			)
			if err != nil || len(podList.Items) == 0 {
				return ""
			}
			podName = podList.Items[0].Name
			return podName
		}, 120*time.Second, 3*time.Second).ShouldNot(BeEmpty())

		By("Waiting for all containers to be ready")
		Eventually(func() bool {
			pod := &corev1.Pod{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      podName,
				Namespace: namespace,
			}, pod)
			if err != nil {
				return false
			}
			if pod.Status.Phase != corev1.PodRunning {
				GinkgoWriter.Printf("Pod phase: %s (waiting for Running)\n", pod.Status.Phase)
				return false
			}
			// Check all regular containers are ready
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					GinkgoWriter.Printf("Container %s not ready\n", cs.Name)
					return false
				}
			}
			// Check init containers (chromium runs as a restartable init container)
			for _, cs := range pod.Status.InitContainerStatuses {
				if !cs.Ready {
					GinkgoWriter.Printf("Init container %s not ready\n", cs.Name)
					return false
				}
			}
			return true
		}, 10*time.Minute, 5*time.Second).Should(BeTrue())

		By("Finding a free local port for port-forward")
		listener, err := net.Listen("tcp", ":0")
		Expect(err).NotTo(HaveOccurred())
		localPort = listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		By(fmt.Sprintf("Starting port-forward to pod %s gateway on local port %d", podName, localPort))
		portFwdCmd = exec.Command("kubectl", "port-forward",
			fmt.Sprintf("pod/%s", podName),
			fmt.Sprintf("%d:%d", localPort, resources.GatewayPort),
			"-n", namespace,
		)
		portFwdCmd.Stdout = GinkgoWriter
		portFwdCmd.Stderr = GinkgoWriter
		Expect(portFwdCmd.Start()).To(Succeed())

		By("Waiting for port-forward to be ready")
		Eventually(func() error {
			if portFwdCmd.ProcessState != nil {
				return fmt.Errorf("port-forward process exited: %s", portFwdCmd.ProcessState)
			}
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		}, 60*time.Second, 2*time.Second).Should(Succeed())

		GinkgoWriter.Printf("Gateway port-forward ready on localhost:%d\n", localPort)
	})

	AfterAll(func() {
		if portFwdCmd != nil && portFwdCmd.Process != nil {
			By("Killing port-forward process")
			_ = portFwdCmd.Process.Kill()
			_ = portFwdCmd.Wait()
		}

		if namespace != "" {
			By("Deleting test namespace")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}
			_ = k8sClient.Delete(ctx, ns)
		}
	})

	It("Should take a screenshot of openclaw.rocks via the agent pipeline", func() {
		By("Reading the gateway token from the auto-generated Secret")
		tokenSecret := &corev1.Secret{}
		secretName := resources.GatewayTokenSecretName(&openclawv1alpha1.OpenClawInstance{
			ObjectMeta: metav1.ObjectMeta{Name: instanceName},
		})
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: namespace,
		}, tokenSecret)).To(Succeed())

		gatewayToken := string(tokenSecret.Data["token"])
		Expect(gatewayToken).NotTo(BeEmpty(), "gateway token should not be empty")
		GinkgoWriter.Printf("Gateway token retrieved (length=%d)\n", len(gatewayToken))

		By("Connecting WebSocket to gateway")
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
		wsURL := fmt.Sprintf("ws://localhost:%d", localPort)
		// Origin header must match the gateway's allowedOrigins config (which uses
		// the gateway's own port, not our random port-forward port).
		wsHeaders := http.Header{}
		wsHeaders.Set("Origin", fmt.Sprintf("http://localhost:%d", resources.GatewayPort))
		ws, _, err := dialer.Dial(wsURL, wsHeaders)
		Expect(err).NotTo(HaveOccurred(), "should connect to gateway WebSocket")
		defer ws.Close()

		By("Waiting for connect.challenge event")
		var challengeMsg gwMessage
		Expect(ws.SetReadDeadline(time.Now().Add(15 * time.Second))).To(Succeed())
		err = ws.ReadJSON(&challengeMsg)
		Expect(err).NotTo(HaveOccurred(), "should receive challenge event")
		Expect(challengeMsg.Type).To(Equal("event"))
		Expect(challengeMsg.Event).To(Equal("connect.challenge"))
		GinkgoWriter.Printf("Received connect.challenge: %s\n", string(challengeMsg.Payload))

		By("Sending connect request with gateway token")
		connectID := randomHex()
		connectReq := map[string]interface{}{
			"type":   "req",
			"id":     connectID,
			"method": "connect",
			"params": map[string]interface{}{
				"minProtocol": 3,
				"maxProtocol": 3,
				"client": map[string]interface{}{
					// Use control-ui identity with dangerouslyDisableDeviceAuth to
					// preserve operator scopes without requiring device key pairing.
					"id":       "openclaw-control-ui",
					"version":  "1.0.0",
					"platform": "linux",
					"mode":     "ui",
				},
				"auth": map[string]interface{}{
					"token": gatewayToken,
				},
				"scopes": []string{
					"operator.admin",
					"operator.read",
					"operator.write",
					"operator.approvals",
					"operator.pairing",
				},
			},
		}
		Expect(ws.WriteJSON(connectReq)).To(Succeed())

		By("Waiting for connect response")
		Expect(ws.SetReadDeadline(time.Now().Add(15 * time.Second))).To(Succeed())
		_, connectRaw, err := ws.ReadMessage()
		Expect(err).NotTo(HaveOccurred(), "should receive connect response")
		GinkgoWriter.Printf("Connect response: %s\n", string(connectRaw))

		var connectResp gwMessage
		Expect(json.Unmarshal(connectRaw, &connectResp)).To(Succeed())
		Expect(connectResp.Type).To(Equal("res"))
		Expect(connectResp.ID).To(Equal(connectID))
		Expect(connectResp.OK).NotTo(BeNil())
		Expect(*connectResp.OK).To(BeTrue(),
			"connect response should be ok=true, got: %s", string(connectRaw))
		GinkgoWriter.Println("Gateway connection established")

		// Extract session key from connect response for chat.send
		var gwPayload gwConnectPayload
		Expect(json.Unmarshal(connectResp.Payload, &gwPayload)).To(Succeed())
		sessionKey := gwPayload.Snapshot.SessionDefaults.MainSessionKey
		Expect(sessionKey).NotTo(BeEmpty(), "connect response should contain mainSessionKey")
		GinkgoWriter.Printf("Session key: %s\n", sessionKey)

		By("Pre-approving device pairing via browser.request probe")
		// Trigger a browser.request to force any pending device pairing flow.
		// OpenClaw emits device.pair.requested when a control-ui client first
		// invokes a browser operation. Auto-approve it so the LLM does not see
		// a pairing gate when we send the actual chat message.
		probeID := randomHex()
		probeReq := map[string]interface{}{
			"type":   "req",
			"id":     probeID,
			"method": "browser.request",
			"params": map[string]interface{}{
				"method": "GET",
				"path":   "/json/version",
			},
		}
		Expect(ws.WriteJSON(probeReq)).To(Succeed())

		// Read events until the probe response comes back, auto-approving any
		// device pairing requests along the way.
		probeDeadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(probeDeadline) {
			Expect(ws.SetReadDeadline(probeDeadline)).To(Succeed())
			_, probeRaw, probeErr := ws.ReadMessage()
			if probeErr != nil {
				GinkgoWriter.Printf("Probe read error: %v\n", probeErr)
				break
			}
			var probeMsg gwMessage
			if jsonErr := json.Unmarshal(probeRaw, &probeMsg); jsonErr != nil {
				continue
			}
			if probeMsg.Type == "event" && probeMsg.Event == "device.pair.requested" {
				var pairReq gwDevicePairRequestPayload
				if jsonErr := json.Unmarshal(probeMsg.Payload, &pairReq); jsonErr == nil && pairReq.RequestID != "" {
					GinkgoWriter.Printf("Auto-approving device pair: %s\n", pairReq.RequestID)
					approveReq := map[string]interface{}{
						"type":   "req",
						"id":     randomHex(),
						"method": "device.pair.approve",
						"params": map[string]interface{}{
							"requestId": pairReq.RequestID,
						},
					}
					_ = ws.WriteJSON(approveReq)
				}
				continue
			}
			if probeMsg.Type == "res" && probeMsg.ID == probeID {
				if probeMsg.OK != nil && *probeMsg.OK {
					GinkgoWriter.Println("Browser probe succeeded - device pairing complete")
				} else {
					logMsg := string(probeRaw)
					if len(logMsg) > 200 {
						logMsg = logMsg[:200] + "..."
					}
					GinkgoWriter.Printf("Browser probe returned error (expected if pairing just completed): %s\n", logMsg)
				}
				break
			}
		}

		By("Sending message to take a screenshot of openclaw.rocks")
		sendID := randomHex()
		idempotencyKey := randomHex()
		sendReq := map[string]interface{}{
			"type":   "req",
			"id":     sendID,
			"method": "chat.send",
			"params": map[string]interface{}{
				"message":        "Navigate to https://openclaw.rocks and take a screenshot. Use the browser tool with the default profile.",
				"sessionKey":     sessionKey,
				"idempotencyKey": idempotencyKey,
			},
		}
		Expect(ws.WriteJSON(sendReq)).To(Succeed())
		GinkgoWriter.Println("Message sent via chat.send, waiting for agent response...")

		By("Reading events until screenshot data or completion")
		var screenshotData string
		agentCompleted := false
		pairApproved := false
		deadline := time.Now().Add(2 * time.Minute)

		for time.Now().Before(deadline) && !agentCompleted {
			Expect(ws.SetReadDeadline(deadline)).To(Succeed())

			_, rawMsg, readErr := ws.ReadMessage()
			if readErr != nil {
				GinkgoWriter.Printf("WebSocket read error: %v\n", readErr)
				break
			}

			var msg gwMessage
			if jsonErr := json.Unmarshal(rawMsg, &msg); jsonErr != nil {
				GinkgoWriter.Printf("Failed to unmarshal message: %v\n", jsonErr)
				continue
			}

			switch {
			case msg.Type == "event" && msg.Event == "device.pair.requested" && !pairApproved:
				// Auto-approve device pairing requests so the browser tool
				// can proceed without the LLM seeing a pairing gate.
				var pairReq gwDevicePairRequestPayload
				if jsonErr := json.Unmarshal(msg.Payload, &pairReq); jsonErr == nil && pairReq.RequestID != "" {
					GinkgoWriter.Printf("Auto-approving device pair request: %s\n", pairReq.RequestID)
					approveReq := map[string]interface{}{
						"type":   "req",
						"id":     randomHex(),
						"method": "device.pair.approve",
						"params": map[string]interface{}{
							"requestId": pairReq.RequestID,
						},
					}
					_ = ws.WriteJSON(approveReq)
					pairApproved = true
				}

			case msg.Type == "event" && msg.Event == "tool.result":
				var toolPayload gwToolResultPayload
				if jsonErr := json.Unmarshal(msg.Payload, &toolPayload); jsonErr == nil {
					GinkgoWriter.Printf("Tool result: tool=%s\n", toolPayload.Tool)
					if toolPayload.Result != nil {
						if data, ok := toolPayload.Result["data"].(string); ok && data != "" {
							GinkgoWriter.Printf("Screenshot data received: %d bytes\n", len(data))
							screenshotData = data
						}
					}
				}

			case msg.Type == "event" && msg.Event == "chat":
				var chatPayload gwChatPayload
				if jsonErr := json.Unmarshal(msg.Payload, &chatPayload); jsonErr == nil {
					if chatPayload.State == "final" {
						GinkgoWriter.Printf("Chat completed: stopReason=%s\n", chatPayload.StopReason)
						agentCompleted = true
					}
				}

			case msg.Type == "res" && msg.ID == sendID:
				GinkgoWriter.Printf("Received response for send request: %s\n", string(rawMsg))

			default:
				// Log other messages for debugging (truncate long payloads)
				logMsg := string(rawMsg)
				if len(logMsg) > 200 {
					logMsg = logMsg[:200] + "..."
				}
				GinkgoWriter.Printf("Event: type=%s event=%s - %s\n", msg.Type, msg.Event, logMsg)
			}
		}

		By("Verifying screenshot data was received")
		Expect(screenshotData).NotTo(BeEmpty(),
			"should have received screenshot data from the agent pipeline")

		// Verify the data is valid base64
		decoded, decodeErr := base64.StdEncoding.DecodeString(screenshotData)
		Expect(decodeErr).NotTo(HaveOccurred(),
			"screenshot data should be valid base64")
		Expect(len(decoded)).To(BeNumerically(">", 100),
			"decoded screenshot should be a non-trivial image (got %d bytes)", len(decoded))

		GinkgoWriter.Printf("Screenshot verified: %d bytes base64, %d bytes decoded PNG\n",
			len(screenshotData), len(decoded))
	})
})

// readCDPResponse reads CDP WebSocket messages until a response with the given
// ID is found or the timeout expires. Non-matching messages (events and
// responses for other IDs) are logged and discarded.
func readCDPResponse(ws *websocket.Conn, id int, timeout time.Duration) *cdpResponse {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ws.SetReadDeadline(deadline); err != nil {
			GinkgoWriter.Printf("Failed to set read deadline: %v\n", err)
			return nil
		}

		_, msg, err := ws.ReadMessage()
		if err != nil {
			GinkgoWriter.Printf("WebSocket read error: %v\n", err)
			return nil
		}

		var resp cdpResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			GinkgoWriter.Printf("Failed to unmarshal CDP message: %v\n", err)
			continue
		}

		if resp.ID == id {
			return &resp
		}

		// Log events and other responses for debugging
		if resp.Method != "" {
			GinkgoWriter.Printf("CDP event: %s\n", resp.Method)
		} else {
			GinkgoWriter.Printf("CDP response for id=%d (waiting for id=%d)\n", resp.ID, id)
		}
	}
	return nil
}

// waitForLoadEvent reads CDP messages looking for a Page.loadEventFired event.
// If the event is not received within the timeout, the function returns
// silently (the screenshot may still succeed on a partially loaded page).
func waitForLoadEvent(ws *websocket.Conn, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ws.SetReadDeadline(deadline); err != nil {
			return
		}

		_, msg, err := ws.ReadMessage()
		if err != nil {
			GinkgoWriter.Printf("waitForLoadEvent: read error: %v\n", err)
			return
		}

		var resp cdpResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}

		GinkgoWriter.Printf("waitForLoadEvent: %s (id=%d)\n", resp.Method, resp.ID)

		if resp.Method == "Page.loadEventFired" {
			GinkgoWriter.Println("Page load event received")
			return
		}
	}
	GinkgoWriter.Println("waitForLoadEvent: timed out waiting for Page.loadEventFired, continuing anyway")
}

// rewriteCDPWebSocketURL replaces the host:port in a Chrome DevTools WebSocket
// URL with localhost:<localPort> so it works through kubectl port-forward.
func rewriteCDPWebSocketURL(wsURL string, localPort int) string {
	wsURL = strings.Replace(wsURL,
		fmt.Sprintf("localhost:%d", resources.ChromiumPort),
		fmt.Sprintf("localhost:%d", localPort), 1)
	wsURL = strings.Replace(wsURL,
		fmt.Sprintf("127.0.0.1:%d", resources.ChromiumPort),
		fmt.Sprintf("localhost:%d", localPort), 1)
	wsURL = strings.Replace(wsURL,
		fmt.Sprintf("0.0.0.0:%d", resources.ChromiumPort),
		fmt.Sprintf("localhost:%d", localPort), 1)
	return wsURL
}
