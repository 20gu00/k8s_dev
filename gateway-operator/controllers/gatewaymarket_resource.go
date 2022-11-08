package controllers

import (
	v1 "github.com/20gu00/gateway-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	GatewayMarketLableKey  = "gogateway.cjq.io/gatewaymarket"
	GatewayMarketCommonKey = "app"
)

func MutateDeploy(gatewayMarket *v1.GatewayMarket, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		GatewayMarketCommonKey: "gatewaymarket",
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: gatewayMarket.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				GatewayMarketLableKey: gatewayMarket.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				//任一匹配
				Labels: map[string]string{
					GatewayMarketLableKey:  gatewayMarket.Name,
					GatewayMarketCommonKey: "gatewaymarket",
				},
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(gatewayMarket),
			},
		},
	}
}

func newContainers(gatewayMarket *v1.GatewayMarket) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			Name:            "gateway-market-container",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Image:           gatewayMarket.Spec.Image,
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "marketport",
					ContainerPort: 8880,
				},
			},
		},
	}
}

func MutateSvc(gatewayMarket *v1.GatewayMarket, svc *corev1.Service) {
	svc.Labels = map[string]string{
		GatewayMarketCommonKey: "gatewaymarket",
	}
	oldClusterIp := svc.Spec.ClusterIP
	svc.Spec = corev1.ServiceSpec{
		ClusterIP: oldClusterIp, //新旧对比,资源创建出来部署了kube-proxy会分配个clientip,调谐过程新的资源和旧资源对比,但新的资源没有部署没有分配clientip
		Type:      corev1.ServiceTypeNodePort,
		Selector: map[string]string{
			GatewayMarketLableKey: gatewayMarket.Name,
		},
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Name:       "market",
				Port:       8880,
				TargetPort: intstr.FromInt(8880),
				Protocol:   corev1.ProtocolTCP,
				NodePort:   30088,
			},
		},
	}
}
