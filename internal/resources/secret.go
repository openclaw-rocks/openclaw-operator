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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// BuildGatewayTokenSecret creates a Secret containing the gateway authentication token.
// The token is used to configure gateway.auth.mode=token so that Bonjour/mDNS pairing
// (which is unusable in Kubernetes) is bypassed automatically.
func BuildGatewayTokenSecret(instance *openclawv1alpha1.OpenClawInstance, tokenHex string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayTokenSecretName(instance),
			Namespace: instance.Namespace,
			Labels:    Labels(instance),
		},
		Data: map[string][]byte{
			GatewayTokenSecretKey: []byte(tokenHex),
		},
	}
}
