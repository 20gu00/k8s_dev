package main

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

func homeDir() string {
	//linux
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	//win
	return os.Getenv("USERPROFILE")
}

func NewMysqlSinglePvPvc() {
	var err error
	var config *rest.Config
	var kubeconfig string
	var storageName = "mysql-single"
	home := homeDir()
	//注意空指针
	kubeconfig = filepath.Join(home, ".kube", "config")
	//pod ca token或者kubeconfig->*rest.Config
	if config, err = rest.InClusterConfig(); err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	kubeClientSet.CoreV1().PersistentVolumes().Create(context.TODO(), &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql-pv-volume",
		},
		Spec: corev1.PersistentVolumeSpec{
			//匿名字段,类型名称
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/data/mysql-single",
				},
			},
			StorageClassName: storageName,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("20Gi"),
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			//切片
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
		},
	}, metav1.CreateOptions{})

	kubeClientSet.CoreV1().PersistentVolumeClaims("default").
		Create(context.TODO(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mysql-pv-claim",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageName,
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteMany,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						//string类型的常量
						corev1.ResourceStorage: resource.MustParse("20Gi"),
					},
				},
			},
		}, metav1.CreateOptions{})
}

func main() {
	NewMysqlSinglePvPvc()
}
