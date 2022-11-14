package controllers

import (
	v1 "github.com/20gu00/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	RedisSingleStorage  = "redis-single"
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

func NewPvCm() {
	var err error
	var config *rest.Config
	var kubeConfig string
	h := homeDir()
	kubeConfig = filepath.Join(h, ".kube", "config")
	//if语句可以使自己的局部变量,局部变量覆盖性
	if config, err = rest.InClusterConfig(); err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeConfig); err != nil {
			panic(err.Error())
		}

	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	kubeClient.CoreV1().PersistentVolumes().Create(&corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redis-pv-volume",
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: "redis-single",
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("2Gi"),
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/data/redis-single",
				},
			},
		},
	})
	kubeClient.CoreV1().PersistentVolumeClaims("default").Create(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redis-6379-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
			StorageClassName: &RedisSingleStorage,
		},
	})
	kubeClient.StorageV1().StorageClasses().Create(&storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redis-single",
		},
		Provisioner: "kubernetes.io/no-provisioner",
	})
	kubeClient.CoreV1().ConfigMaps("default").Create(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "redis-cm",
		},
		Data: map[string]string{
			"redis_6379.conf": "protected-mode no\n    port 6379\n    tcp-backlog 511\n    timeout 0\n    tcp-keepalive 300\n    daemonize no\n    supervised no\n    pidfile /var/run/redis_6379.pid\n    loglevel notice\n    logfile \"/data/redis_6379.log\"\n    databases 16\n    always-show-logo yes\n    # requirepass {{ $name.redis.env.passwd }}\n    save 900 1\n    save 300 10\n    save 60 10000\n    stop-writes-on-bgsave-error yes\n    rdbcompression yes\n    rdbchecksum yes\n    dbfilename dump_6379.rdb\n    dir  /data\n    replica-serve-stale-data yes\n    replica-read-only yes\n    repl-diskless-sync no\n    repl-diskless-sync-delay 5\n    repl-disable-tcp-nodelay no\n    replica-priority 100\n    lazyfree-lazy-eviction no\n    lazyfree-lazy-expire no\n    lazyfree-lazy-server-del no\n    replica-lazy-flush no\n    appendonly no\n    appendfilename \"appendonly.aof\"\n    appendfsync everysec\n    no-appendfsync-on-rewrite no\n    auto-aof-rewrite-percentage 100\n    auto-aof-rewrite-min-size 64mb\n    aof-load-truncated yes\n    aof-use-rdb-preamble yes\n    lua-time-limit 5000\n    slowlog-log-slower-than 10000\n    slowlog-max-len 128\n    latency-monitor-threshold 0\n    notify-keyspace-events \"\"\n    hash-max-ziplist-entries 512\n    hash-max-ziplist-value 64\n    list-max-ziplist-size -2\n    list-compress-depth 0\n    set-max-intset-entries 512\n    zset-max-ziplist-entries 128\n    zset-max-ziplist-value 64\n    hll-sparse-max-bytes 3000\n    stream-node-max-bytes 4096\n    stream-node-max-entries 100\n    activerehashing yes\n    client-output-buffer-limit normal 0 0 0\n    client-output-buffer-limit replica 256mb 64mb 60\n    client-output-buffer-limit pubsub 32mb 8mb 60\n    hz 10\n    dynamic-hz yes\n    aof-rewrite-incremental-fsync yes\n    rdb-save-incremental-fsync yes\n    rename-command FLUSHALL SAVEMORE16\n    rename-command FLUSHDB  SAVEDB16\n    rename-command CONFIG   UPDATEC16\n    rename-command KEYS     NOALL16",
		},
	})
}

func MutateSvc(redisSingle *v1.RedisSingle, svc *corev1.Service) {
	svc.Labels = map[string]string{
		RedisSingleCommonLabelKey: "redisSingle",
	}
	svc.Spec = corev1.ServiceSpec{

		Type: corev1.ServiceTypeNodePort,
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Port:       6379,
				TargetPort: intstr.FromInt(6379),
				Protocol:   corev1.ProtocolTCP,
				Name:       "redis-6379",
			},
		},
		Selector: map[string]string{
			RedisSingleLabelKey: redisSingle.Name,
		},
	}
}
