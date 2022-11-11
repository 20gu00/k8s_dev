package controllers

import (
	v1 "github.com/20gu00/mysql-single-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	MysqlSingleCommonLabelKey = "cjqapp"
	//group domain
	MysqlSingleLabelKey = "cjqapp.cjq.io/mysqlSingle"
)

//处理的是指针
func MutateDeployment(mysqlSingle *v1.MysqlSingle, deploy *appsv1.Deployment) {
	deploy.Labels = map[string]string{
		//deploy的label
		MysqlSingleCommonLabelKey: "mysqlsingle",
	}

	//定义spec
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: mysqlSingle.Spec.Replicas,
		//deploy的selector
		//MatchLabels MatchExpressions
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
			MysqlSingleLabelKey: mysqlSingle.Name,
		}},
		//pod
		Template: corev1.PodTemplateSpec{
			//metadata
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					MysqlSingleLabelKey:       mysqlSingle.Name, //用于select的标签
					MysqlSingleCommonLabelKey: "mysqlsingle",    //普通身份标示标签
				},
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(mysqlSingle),
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "mysqlVolume",
						//使用的pvc作为volume
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "mysql-pvc-claim", //pvc名称
								//ReadOnly: true,
							},
						},
					},
				},
			},
		},
	}
}

func newContainers(mysqlSingle *v1.MysqlSingle) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			Name:  "mysqlSingle",
			Image: mysqlSingle.Spec.Image,
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					ContainerPort: 3306,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				corev1.VolumeMount{
					Name:      "mysqlVolume",
					MountPath: "/var/lib/mysql",
				},
			},
			//设置容器的环境变量
			Env: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "MYSQL_ROOT_PASSWORD",
					Value: mysqlSingle.Spec.MysqlPassword,
					//ValueFrom: &corev1.EnvVarSource{
					//	FieldRef: &corev1.ObjectFieldSelector{
					//		FieldPath:
					//	},
				},
			},
		},
	}
}

func MutateSvc(mysqlSingle *v1.MysqlSingle, svc *corev1.Service) {
	svc.Labels = map[string]string{
		MysqlSingleCommonLabelKey: "mysqlsingle",
	}
	svc.Spec = corev1.ServiceSpec{
		Selector: map[string]string{
			MysqlSingleLabelKey: mysqlSingle.Name,
		},
		Type: corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Port:       3306,
				Protocol:   corev1.ProtocolTCP,   //mysql
				TargetPort: intstr.FromInt(3306), //intstr包
				//NodePort: 31234,
			},
		},
	}
}
