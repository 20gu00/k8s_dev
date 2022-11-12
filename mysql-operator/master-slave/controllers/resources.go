package controllers

import (
	v1 "github.com/20gu00/masterslave/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	MasterSlaveLabelKey       = "masterslave.cjq.io/masterslave"
	MasterSlaveCommonLabelKey = "app"
	EtcdDataDirName           = "datadir"
)

func MutateStatefulset(masterSlave *v1.MasterSlave, sts *appsv1.StatefulSet) {
	sts.Labels = map[string]string{
		MasterSlaveCommonLabelKey: "masterSlave",
	}
	sts.Spec = appsv1.StatefulSetSpec{
		ServiceName: masterSlave.Name,
		Replicas:    masterSlave.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				MasterSlaveLabelKey: masterSlave.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					MasterSlaveCommonLabelKey: "masterSlave",
					MasterSlaveLabelKey:       masterSlave.Name,
				},
			},
			Spec: corev1.PodSpec{
				//InitContainers: []corev1.Container{},
				InitContainers: newInitContainer(masterSlave),
				Containers:     newContainers(masterSlave),
			},
		},
	}
}

func newInitContainer(masterSlave *v1.MasterSlave) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			//初始化
			Name:  "init-mysql",
			Image: masterSlave.Spec.Image,
			Command: []string{
				"bash", "-c",
				"set -ex\n              # Generate mysql server-id from pod ordinal index.\n              [[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n              ordinal=${BASH_REMATCH[1]}\n              echo [mysqld] > /mnt/conf.d/server-id.cnf\n              # Add an offset to avoid reserved server-id=0 value.\n              echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf\n              # Copy appropriate conf.d files from config-map to emptyDir.\n              if [[ $ordinal -eq 0 ]]; then\n                cp /mnt/config-map/master.cnf /mnt/conf.d/\n              else\n                cp /mnt/config-map/slave.cnf /mnt/conf.d/\n              fi",
			},
			VolumeMounts: []corev1.VolumeMount{
				corev1.VolumeMount{
					Name:      "conf",
					MountPath: "/mnt/conf.d",
				},
				corev1.VolumeMount{
					Name:      "config-map",
					MountPath: "/mnt/config-map",
				},
			},
		},
		corev1.Container{
			//同步数据
			Name:  "clone-mysql",
			Image: "fxkjnj/xtrabackup:1.0",
			Command: []string{
				"bash", "-c",
				"set -ex\n              # Skip the clone if data already exists.\n              [[ -d /var/lib/mysql/mysql ]] && exit 0\n              # Skip the clone on master (ordinal index 0).\n              [[ `hostname` =~ -([0-9]+)$ ]] || exit 1\n              ordinal=${BASH_REMATCH[1]}\n              [[ $ordinal -eq 0 ]] && exit 0\n              # Clone data from previous peer.\n              ncat --recv-only mysql-$(($ordinal-1)).mysql 3307 | xbstream -x -C /var/lib/mysql\n              # Prepare the backup.\n              xtrabackup --prepare --target-dir=/var/lib/mysql",
			},
			VolumeMounts: []corev1.VolumeMount{
				corev1.VolumeMount{
					Name:      "data",
					MountPath: "/var/lib/mysql",
					SubPath:   "mysql", //会自动给volume创建子路径mysql(挂载类似文件系统操作,目录是入口,会隐藏原来的文件)
					//volume的子路径的,比如cm secret的key
				},
				corev1.VolumeMount{
					Name:      "conf",
					MountPath: "/etc/mysql/conf.d",
				},
			},
		},
	}
}

func newContainers(masterSlave *v1.MasterSlave) []corev1.Container {

}
