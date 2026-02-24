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
	"k8s.io/apimachinery/pkg/util/intstr"

	openclawv1alpha1 "github.com/openclawrocks/k8s-operator/api/v1alpha1"
)

// BuildService creates a Service for the OpenClawInstance
func BuildService(instance *openclawv1alpha1.OpenClawInstance) *corev1.Service {
	labels := Labels(instance)
	selectorLabels := SelectorLabels(instance)

	serviceType := instance.Spec.Networking.Service.Type
	if serviceType == "" {
		serviceType = corev1.ServiceTypeClusterIP
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ServiceName(instance),
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Spec.Networking.Service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:            serviceType,
			Selector:        selectorLabels,
			SessionAffinity: corev1.ServiceAffinityNone,
			Ports:           buildServicePorts(instance),
		},
	}

	return service
}

// buildServicePorts returns custom ports if specified, otherwise default ports.
func buildServicePorts(instance *openclawv1alpha1.OpenClawInstance) []corev1.ServicePort {
	if len(instance.Spec.Networking.Service.Ports) > 0 {
		ports := make([]corev1.ServicePort, 0, len(instance.Spec.Networking.Service.Ports))
		for _, p := range instance.Spec.Networking.Service.Ports {
			protocol := p.Protocol
			if protocol == "" {
				protocol = corev1.ProtocolTCP
			}
			tp := intstr.FromInt32(p.Port)
			if p.TargetPort != nil {
				tp = intstr.FromInt32(*p.TargetPort)
			}
			ports = append(ports, corev1.ServicePort{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: tp,
				Protocol:   protocol,
			})
		}
		return ports
	}

	ports := []corev1.ServicePort{
		{
			Name:       "gateway",
			Port:       int32(GatewayPort),
			TargetPort: intstr.FromInt32(int32(GatewayProxyPort)),
			Protocol:   corev1.ProtocolTCP,
		},
		{
			Name:       "canvas",
			Port:       int32(CanvasPort),
			TargetPort: intstr.FromInt32(int32(CanvasProxyPort)),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	if instance.Spec.Chromium.Enabled {
		ports = append(ports, corev1.ServicePort{
			Name:       "chromium",
			Port:       int32(ChromiumPort),
			TargetPort: intstr.FromInt32(int32(ChromiumPort)),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	if instance.Spec.WebTerminal.Enabled {
		ports = append(ports, corev1.ServicePort{
			Name:       "web-terminal",
			Port:       int32(WebTerminalPort),
			TargetPort: intstr.FromInt32(int32(WebTerminalPort)),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	if IsMetricsEnabled(instance) {
		metricsPort := MetricsPort(instance)
		ports = append(ports, corev1.ServicePort{
			Name:       "metrics",
			Port:       metricsPort,
			TargetPort: intstr.FromInt32(metricsPort),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	return ports
}
