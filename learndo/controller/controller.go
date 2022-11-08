package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

//自定义控制器(pod controller)
type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

//workqueue indexer informer(核心三部分)
func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

//处理下一个元素
func (c *Controller) processNextItem() bool {
	// 等到工作队列中有一个新元素,从队列取出一个元素,如果没有元素就阻塞
	key, quit := c.queue.Get() //拿到元素item和判断通道是否关闭的值
	if quit {                  //通道已经关闭,标记为关闭
		return false
	}

	//元素处理完成就要标记为done(队列是通过key来获取item的)(标记Object)
	// 告诉队列我们已经完成了处理此 key 的操作,这将为其他 worker 解锁该 key
	// 这将确保安全的并行处理，因为永远不会并行处理具有相同 key 的两个Pod
	defer c.queue.Done(key)

	//调用包含业务逻辑的方法
	err := c.syncToStdout(key.(string))
	//处理业务逻辑错误
	c.handleErr(err, key)
	return true
}

//控制器的业务逻辑
// 在此控制器中，它只是将有关 Pod 的信息打印到 stdout
// 如果发生错误，则简单地返回错误
// 此外重试逻辑不应成为业务逻辑的一部分。
func (c *Controller) syncToStdout(key string) error {
	// 从本地存储indexer中获取key对应的对象object
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("从缓存中获取%s对应的object出错%v", key, err)
		return err
	}

	//对象不存在,比如已经删除了,indexer中没有该元素
	if !exists {
		fmt.Printf("pod %s 不存在\n", key) //比如default/pod
	} else { //pod存在
		fmt.Printf("Sync/Add/Update for Pod %s\n", obj.(*v1.Pod).GetName())
	}

	return nil
}

// Object的逻辑处理出错,重新放入队列
func (c *Controller) handleErr(err error, key interface{}) {
	//没有错误
	if err == nil {
		// 忘记每次成功同步时 key 的#AddRateLimited历史记录。
		// 这样可以确保不会因过时的错误历史记录而延迟此 key 更新的以后处理。
		c.queue.Forget(key) //处理完成,队列忘记这个key
		return
	}
	//如果出错,控制其允许重试10次
	if c.queue.NumRequeues(key) < 10 {
		klog.Infof("重新入队列 %v: %v", key, err)
		// key重新加入到限速队列
		// 根据队列上的速率限制器(限速队列的限速器)和重新入队历史记录，稍后将再次处理该 key,优先级由队列自行判断
		c.queue.AddRateLimited(key)
		return
	}
	//超过重试次数
	c.queue.Forget(key)
	//处理错误
	runtime.HandleError(err)
	klog.Infof("从队列中删除%q这个pod: %v", key, err)
}

//控制循环Run开始watch和同步
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	//崩溃会执行defer
	defer runtime.HandleCrash() //捕获崩溃并记录

	//先调用,停止控制器后关掉工作队列
	defer c.queue.ShutDown()
	klog.Info("开启控制器") //日志记录

	// informer启动(list watch)
	go c.informer.Run(stopCh)

	// 等待所有相关的缓存同步，然后再开始处理队列中的项目
	//等待informer缓存同步,indexer中的数据和etcd保持一致
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("等待同步超时")) //k8s的runtime处理错误
		return
	}

	//控制器启动多个协程处理
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh) //不断循环直到通道关闭,每个一定时间执行一次c.runWorker
	}

	<-stopCh //阻塞
	//通道接受到信息,关闭(信号通道)
	klog.Info("关闭pod controller") //打印
}

//处理工作队列,处理元素
func (c *Controller) runWorker() {
	for c.processNextItem() { //不断循环
	}
}

func initClient() (*kubernetes.Clientset, error) {
	var err error
	var config *rest.Config
	// inCluster（Pod）、KubeConfig（kubectl）
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(可选) kubeconfig 文件的绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig 文件的绝对路径")
	}
	flag.Parse()

	// 首先使用 inCluster 模式(需要去配置对应的 RBAC 权限，默认的sa是default->是没有获取deployments的List权限)
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	// 已经获得了 rest.Config 对象
	// 创建 Clientset 对象
	return kubernetes.NewForConfig(config)
}

func main() {
	clientset, err := initClient()
	if err != nil {
		klog.Fatal(err)
	}

	//创建针对pod资源的ListWatcher(List watch)(reflector)(指定namespace)
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything()) //全部

	//创建工作队列(限速队列)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//工作队列 indexer通过informer连在一块,将pod的key添加到队列中
	// 注意，当我们最终从工作队列中处理元素时，我们可能会看到 Pod 的版本比响应触发更新的版本新(经过DeltaFifo事件处理资源的resource版本都会更新)
	//创建indexer informer,重新同步时间间隔(informer从本地缓存比如DeltaFifo同步数据,而不是apiserver,0不让重新同步,指定获取的资源类型
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		//第一次会出发add事件
		//直接在这里定义事件触发的函数
		//keyFunc用于生成资源对象的对象键,一般索引器是用namespace
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key) //向队列中添加key
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			//删除有两个状态
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{}) //indexer函数集合map,type IndexFunc func(obj interface{}) ([]string, error)

	controller := NewController(queue, indexer, informer)

	//向indexer中手动添加pod,实际不存在这个pod
	indexer.Add(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod",
			Namespace: v1.NamespaceDefault,
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	//启动controller
	go controller.Run(1, stopCh)

	select {} //阻塞
}
