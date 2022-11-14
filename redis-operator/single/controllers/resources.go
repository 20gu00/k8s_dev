package controllers

import (
	v1 "github.com/20gu00/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

var (
	RedisSingleCommonLabelKey = "app"
	//group domain
	RedisSingleLabelKey = "app.cjq.io/redisSingle"
)

func MutateDeployment(redisSingle *v1.RedisSingle, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		RedisSingleCommonLabelKey: "redisSingle",
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: redisSingle.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				RedisSingleLabelKey: redisSingle.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					RedisSingleCommonLabelKey: "redisSingle",
					RedisSingleLabelKey:       redisSingle.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Name:  "redis-6379",
						Image: redisSingle.Spec.Image,
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "configmap-volume",
								MountPath: "/usr/local/etc/redis/redis_6379.conf",
								SubPath:   "redis_6379.conf",
							},
							corev1.VolumeMount{
								Name:      "redis-6379",
								MountPath: "/data",
							},
						},
						Command: []string{
							"redis-server",
						},
						Args: []string{
							"/usr/local/etc/redis/redis_6379.conf",
						},
					},
				},
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "configmap-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "redis-cm",
								},
								Items: []corev1.KeyToPath{
									corev1.KeyToPath{
										Key:  "redis_6379.conf",
										Path: "redis_6379.conf",
									},
								},
							},
						},
					},
					corev1.Volume{
						Name: "redis-6379",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "redis-6379-pvc",
							},
						},
					},
				},
			},
		},
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	return os.Getenv("USERPROFILE")
}
func NewPv() {
	var err error
	var config *rest.Config
	var kubeConfig *string
	h := homeDir()
	*kubeConfig = filepath.Join(h, ".kube", "config")
	if config, err := rest.InClusterConfig(); err != nil {
		if config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig); err != nil {
			panic(err.Error())
		}

	}

	kubeClinet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	kubeClinet.CoreV1().PersistentVolumes().Create(&corev1.PersistentVolume{})
}
