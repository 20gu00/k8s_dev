package controllers

import (
	v1 "github.com/20gu00/redis-sts/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	RedisStsCommonKey = "app.cjq.io/redisSts"
	RedisStsLabelKey  = "app"
)

func MutateStatefulset(redisSts *v1.RedisSts, deploy *appsv1.StatefulSet) {
	deploy.Labels = map[string]string{
		RedisStsCommonKey: "redisSts",
	}
	deploy.Spec = appsv1.StatefulSetSpec{
		ServiceName: "redis",
		Replicas:    redisSts.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				RedisStsLabelKey: redisSts.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					RedisStsCommonKey: "redisSts",
					RedisStsLabelKey:  redisSts.Name,
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					corev1.Container{
						Name:            "install",
						Image:           "bprashanth/redis-install-3.2.0:e2e",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Args: []string{
							"--install-into=/opt",
							"--work-dir=/work-dir",
						},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "opt",
								MountPath: "/opt",
							},
							corev1.VolumeMount{
								Name:      "workdir",
								MountPath: "/work-dir",
							},
						},
					},
					corev1.Container{
						Name:  "bootstrap",
						Image: "debian:jessie",
						Command: []string{
							"/work-dir/peer-finder",
							"-on-start=\"/work-dir/on-start.sh\"",
							"-service=redis",
						},
						//downwardAPI
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name: "POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										APIVersion: "v1",
										FieldPath:  deploy.Namespace,
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "/opt",
								MountPath: "/opt",
							},
							corev1.VolumeMount{
								Name:      "workdir",
								MountPath: "/work-dir",
							},
						},
					},
				},
				Containers: []corev1.Container{
					corev1.Container{
						Name:  "redis",
						Image: "debian:jessie",
						Ports: []corev1.ContainerPort{
							corev1.ContainerPort{
								Name:          "peer",
								ContainerPort: 6379,
							},
						},
						Command: []string{
							"/opt/redis/redis-server",
						},
						Args: []string{
							"/opt/redis/redis.conf",
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"sh", "-c",
										"/opt/redis/redis-cli -h $(hostname) ping",
									},
								},
							},
							InitialDelaySeconds: 20,
							PeriodSeconds:       5,
							TimeoutSeconds:      10,
						},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "datadir",
								MountPath: "/data",
							},
							corev1.VolumeMount{
								Name:      "opt",
								MountPath: "/opt",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "opt",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					corev1.Volume{
						Name: "workdir",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "datadir",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce, //rwo rwx
					},
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
}

func MutateSvc(redisSingle *v1.RedisSts, svc *corev1.Service) {
	svc.Labels = map[string]string{
		RedisStsCommonKey: "redisSts",
	}
	svc.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Name: "peer",
				Port: 6379, //clusterip
			},
		},
		ClusterIP: corev1.ClusterIPNone,
		Selector: map[string]string{
			RedisStsLabelKey: redisSingle.Name,
		},
		PublishNotReadyAddresses: "true",
	}
}

