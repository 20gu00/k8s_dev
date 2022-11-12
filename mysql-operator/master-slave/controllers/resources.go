package controllers

import (
	v1 "github.com/20gu00/masterslave/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	MasterSlaveLabelKey       = "masterslave.cjq.io/masterslave"
	MasterSlaveCommonLabelKey = "app"
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
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "conf",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{}, //pod同生命周期,数据目录是kubelet目录
						},
					},
					corev1.Volume{
						Name: "config-map",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "mysql",
								},
							},
						},
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					//StorageClassName: "default"
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
				},
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
	return []corev1.Container{
		corev1.Container{
			Name:  "mysql",
			Image: masterSlave.Spec.Image,
			Env: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
					Value: masterSlave.Spec.MysqlPassword,
				},
			},
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "mysql",
					ContainerPort: 3306,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				//data目录
				corev1.VolumeMount{
					Name:      "data",
					MountPath: "/var/lib/mysql",
					SubPath:   "mysql",
				},
				//配置目录
				corev1.VolumeMount{
					Name:      "conf",
					MountPath: "/etc/mysql/conf.d",
					SubPath:   "mysql",
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			LivenessProbe: &corev1.Probe{
				//不是调用匿名结构体的成员而是定义
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							//会自动转换好
							"mysqladmin, ping",
						},
					},
				},
				InitialDelaySeconds: int32(30),
				PeriodSeconds:       int32(20),
				TimeoutSeconds:      int32(5),
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"mysql", "-h",
							"127.0.0.1 -e SELECT 1",
						},
					},
				},
				PeriodSeconds:       int32(2),
				InitialDelaySeconds: int32(30),
				TimeoutSeconds:      int32(2),
			},
		},
		corev1.Container{
			Name:  "xtrabackup",
			Image: "fxkjnj/xtrabackup:1.0",
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "xtrabackup",
					ContainerPort: 3307,
				},
			},
			Command: []string{
				"bash",
				"-c",
				"set -ex\n              cd /var/lib/mysql\n              \n              # Determine binlog position of cloned data, if any.\n              if [[ -f xtrabackup_slave_info && \"x$(<xtrabackup_slave_info)\" != \"x\" ]]; then\n              # XtraBackup already generated a partial \"CHANGE MASTER TO\" query\n              # because we're cloning from an existing slave. (Need to remove the tailing semicolon!)\n              cat xtrabackup_slave_info | sed -E 's/;$//g' > change_master_to.sql.in\n              # Ignore xtrabackup_binlog_info in this case (it's useless).\n              rm -f xtrabackup_slave_info xtrabackup_binlog_info\n              elif [[ -f xtrabackup_binlog_info ]]; then\n              # We're cloning directly from master. Parse binlog position.\n              [[ `cat xtrabackup_binlog_info` =~ ^(.*?)[[:space:]]+(.*?)$ ]] || exit 1\n              rm -f xtrabackup_binlog_info xtrabackup_slave_info\n              echo \"CHANGE MASTER TO MASTER_LOG_FILE='${BASH_REMATCH[1]}',\\\n              MASTER_LOG_POS=${BASH_REMATCH[2]}\" > change_master_to.sql.in\n              fi\n              \n              # Check if we need to complete a clone by starting replication.\n              if [[ -f change_master_to.sql.in ]]; then\n              echo \"Waiting for mysqld to be ready (accepting connections)\"\n              until mysql -h 127.0.0.1 -e \"SELECT 1\"; do sleep 1; done\n              \n              echo \"Initializing replication from clone position\"\n              mysql -h 127.0.0.1 \\\n              -e \"$(<change_master_to.sql.in), \\\n              MASTER_HOST='mysql-0.mysql', \\\n              MASTER_USER='root', \\\n              MASTER_PASSWORD='', \\\n              MASTER_CONNECT_RETRY=10; \\\n              START SLAVE;\" || exit 1\n              # In case of container restart, attempt this at-most-once.\n              mv change_master_to.sql.in change_master_to.sql.orig\n              fi\n              \n              # Start a server to send backups when requested by peers.\n              exec ncat --listen --keep-open --send-only --max-conns=1 3307 -c \\\n              \"xtrabackup --backup --slave-info --stream=xbstream --host=127.0.0.1 --user=root\"\n",
			},
			VolumeMounts: []corev1.VolumeMount{
				corev1.VolumeMount{
					Name:      "data",
					MountPath: "/var/lib/mysql",
					SubPath:   "mysql",
				},
				corev1.VolumeMount{
					Name:      "conf",
					MountPath: "/etc/mysql/conf.d",
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
	}
}
