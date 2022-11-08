package controllers

import (
	v1 "github.com/20gu00/gateway-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	GatewayProxyLableKey  = "gogateway.cjq.io/gatewayproxy"
	GatewayProxyCommonKey = "app"
)

func MutateProxyDeploy(gatewayProxy *v1.GatewayProxy, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		GatewayMarketCommonKey: "gatewayproxy",
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: gatewayProxy.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				GatewayProxyLableKey: gatewayProxy.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				//任一匹配
				Labels: map[string]string{
					GatewayProxyLableKey:  gatewayProxy.Name,
					GatewayProxyCommonKey: "gatewayproxy",
				},
			},
			Spec: corev1.PodSpec{
				Containers: newProxyContainers(gatewayProxy),
			},
		},
	}
}

func newProxyContainers(gatewayProxy *v1.GatewayProxy) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			Name:            "gateway-proxy-container",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Image:           gatewayProxy.Spec.Image,
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "proxyhttp",
					ContainerPort: 8080,
				},
				corev1.ContainerPort{
					Name:          "proxyhttps",
					ContainerPort: 4433,
				},
			},
		},
	}
}

func MutateProxySvc(gatewayProxy *v1.GatewayProxy, svc *corev1.Service) {
	svc.Labels = map[string]string{
		GatewayProxyCommonKey: "gatewayproxy",
	}
	oldClusterIp := svc.Spec.ClusterIP
	svc.Spec = corev1.ServiceSpec{
		ClusterIP: oldClusterIp,
		Type:      corev1.ServiceTypeNodePort,
		Selector: map[string]string{
			GatewayProxyLableKey: gatewayProxy.Name,
		},
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Name:       "proxyhttp",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
				Protocol:   corev1.ProtocolTCP,
				NodePort:   30080,
			},
			corev1.ServicePort{
				Name:       "proxyhttps",
				Port:       4433,
				TargetPort: intstr.FromInt(4433),
				Protocol:   corev1.ProtocolTCP,
				NodePort:   30443,
			},
		},
	}
}
