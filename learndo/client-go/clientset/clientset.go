package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var err error
	var config *rest.Config
	var kubeconfig *string

	//参数
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "输入kubeconfig文件绝对路径,默认/root/.kube/config") //可选
	} else { //未获取HOME
		kubeconfig = flag.String("kubeconfig", "", "输入kubeconfig的绝对路径") //必填
	}
	flag.Parse()

	//获取kubeconfig集群信息(clusters,users,context),两种模式,集群内集群外(kubectl pod),pod内注意权限
	//pod有secret,里边有ca证书和token,进行身份验证,还有对应的sa,权限(默认default)
	if config, err = rest.InClusterConfig(); err != nil { //token ca身份验证权限验证
		// 使用KubeConfig文件创建集群配置*rest.Config对象(文件在本地)
		//接引用
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	//实例化clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	//clientset获取schema类型资源
	//curd
	deployments, err := clientset.AppsV1().Deployments("default").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=v1",
	}) //小写别名
	if err != nil {
		panic(err)
	}

	for idx, deploy := range deployments.Items {
		fmt.Printf("%d -> %s\n", idx+1, deploy.Name)
	}

}

//获取程序所处的系统的家目录
func homeDir() string {
	//linux
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	//win
	return os.Getenv("USERPROFILE")
}
