package controllers

import (
	Myappv1 "github.com/20gu00/operator-sdk-demo/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//需要调谐的对象是deploy,传递的是指针 更改
//调谐函数(更改spec)
//根据自定义资源的spec来更改deployment
func MutateDeployment(app *Myappv1.Myapp, deploy *appsv1.Deployment) {
	labels := map[string]string{"app": app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: app.Spec.Size,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
		Selector: selector,
	}
}

func MutateSvc(app *Myappv1.Myapp, svc *corev1.Service) {
	svc.Spec = corev1.ServiceSpec{
		//clusterip
		ClusterIP: svc.Spec.ClusterIP,
		Type:      corev1.ServiceTypeNodePort,
		Ports:     app.Spec.Ports,
		Selector: map[string]string{
			"app": app.Name,
		},
	}
}
func NewDeploy(app *Myappv1.Myapp) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		//metadata中定义owner
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,

			//ownerreferences,关联到Myapp
			OwnerReferences: []metav1.OwnerReference{
				//owner GVK
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   Myappv1.GroupVersion.Group,
					Version: Myappv1.GroupVersion.Version,
					Kind:    Myappv1.Kind,
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
			Selector: selector,
		},
	}
}

func newContainers(app *Myappv1.Myapp) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts, cport)
	}
	return []corev1.Container{
		{
			Name:            app.Name,
			Image:           app.Spec.Image,
			Resources:       app.Spec.Resources,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             app.Spec.Envs,
		},
	}
}

func NewService(app *Myappv1.Myapp) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   Myappv1.GroupVersion.Group,
					Version: Myappv1.GroupVersion.Version,
					Kind:    Myappv1.Kind,
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeNodePort,
			Ports: app.Spec.Ports,
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}
}
