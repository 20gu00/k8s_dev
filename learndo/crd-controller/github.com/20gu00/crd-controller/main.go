package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog"

	clientset "github.com/20gu00/crd-controller/pkg/client/clientset/versioned"
	informers "github.com/20gu00/crd-controller/pkg/client/informers/externalversions"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM} //关闭 强制退出
)

// SetupSignalHandler 注册 SIGTERM 和 SIGINT 信号
// 返回一个 stop channel，该通道在捕获到第一个信号时被关闭
// 如果捕捉到第二个信号，程序将直接退出
func setupSignalHandler() (stopCh <-chan struct{}) {
	// 当调用两次的时候 panics
	//关闭一个已经关闭的通道直接panic
	close(onlyOneSignalHandler)

	stop := make(chan struct{})
	c := make(chan os.Signal, 2) //缓冲2

	// Notify 函数让 signal 包将输入信号转发到 c
	// 如果没有列出要传递的信号，会将所有输入信号传递到 c；否则只传递列出的输入信号
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop) //关闭通道
		<-c
		os.Exit(1) // 第二个信号，直接退出
	}()

	return stop //传达信号的通道,可以用于优雅退出
}

//初始化客户端 clientset
func initClient() (*kubernetes.Clientset, *rest.Config, error) {
	var err error
	var config *rest.Config
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(可选)kubeconfig文件的绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig文件的绝对路径")
	}

	flag.Parse()
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	//操作kubernetes内置资源的clientset
	kubeclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, config, err
	}
	return kubeclient, config, nil
}

func main() {
	flag.Parse()

	//信号通道应用于优雅关闭
	stopCh := setupSignalHandler()

	_, cfg, err := initClient()
	if err != nil { //clientset获取内置资源
		klog.Fatalf("创建kubernetes的客户端clientset失败: %s", err.Error())
	}

	// 实例化一个CronTab的ClientSet
	crontabClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("创建crontab的clientset失败: %s", err.Error())
	}

	//informerFactory工厂类，这里注入通过代码生成的client,同一类资源共享一个informer
	//clent主要用于和APIServer进行通信，实现ListAndWatch(reflector informer)
	//resync重新同步时间间隔
	crontabInformerFactory := informers.NewSharedInformerFactory(crontabClient, time.Second*30)

	//实例化自定义控制器(和前面的内置资源的自定义控制器一样,实现一个控制循环)
	//CronTab实现了Informer和Lister方法也就是实现了informer接口
	//NewController中informer调用了Informer,相当于注册到工厂中,这个操作要在Start之前
	controller := NewController(crontabInformerFactory.Stable().V1().CronTabs())

	//启动 informer，开始List Watch
	//多个协程,使用的是同一个informer
	go crontabInformerFactory.Start(stopCh)

	//启动控制器,控制循环
	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("运行控制器失败: %s", err.Error())
	}
}
