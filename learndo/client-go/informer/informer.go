package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//通过informer获取资源对象
func main() {
	var err error
	var config *rest.Config //clienset或者informer都需要访问apiserver
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "[可选] kubeconfig 绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig 绝对路径")
	}

	//获取rest.Config
	if config, err = rest.InClusterConfig(); err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	//实例化Clientset对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	//通过clientset初始化informer工厂(30分钟重新list一次数据到informer中)(注意是informer或者它的工厂去本地缓存list而不是下边的Lister方法,就是一个resync的过程,informer自身维护一套缓存)
	//数据的同步是缓存和apiserver之间完成)
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Minute*30)

	//确切的informer,指定要监听的资源
	deployInformer := informerFactory.Apps().V1().Deployments()

	//实例化informer,informer的操作就是list watch(第一次list,重新全面同步同步的时候list,拉去相应的全部资源对象)
	//注册到工厂中
	informer := deployInformer.Informer()

	//创建Lister,Lister从informer当中(informer自身维护的缓存)去获取全部的资源对象数据而不是从apiserver
	deployLister := deployInformer.Lister()

	// 注册事件处理程序
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,    //第一次执行的时候,全面list,算是add
		UpdateFunc: onUpdate, //重新list的时候,新旧对比,也是update
		DeleteFunc: onDelete,
	})

	//信号通道
	stopper := make(chan struct{})
	defer close(stopper)

	//启动 informer，List  Watch
	//将数据拉去到缓存当中
	//Start之前要Informer()和Lister(),不然缓存中没有资源,获取不到任何资源进行操作
	informerFactory.Start(stopper)

	//等待所有启动的 Informer 的缓存被同步(上面可以写多个informer)
	informerFactory.WaitForCacheSync(stopper)

	//Lister从本地缓存informer中获取default中的所有deployment列表
	deployments, err := deployLister.Deployments("default").List(labels.Everything()) //全部
	if err != nil {
		panic(err)
	}
	for idx, deploy := range deployments {
		fmt.Printf("%d -> %s\n", idx+1, deploy.Name)
	}
	<-stopper //阻塞
}

func onAdd(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	fmt.Println("add a deployment:", deploy.Name)
}

func onUpdate(old, new interface{}) {
	oldDeploy := old.(*v1.Deployment) //断言
	newDeploy := new.(*v1.Deployment)
	fmt.Println("update deployment:", oldDeploy.Name, newDeploy.Name)
}

func onDelete(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	fmt.Println("delete a deployment:", deploy.Name)
}
