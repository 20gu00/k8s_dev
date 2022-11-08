package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	crdv1 "github.com/20gu00/crd-controller/pkg/apis/stableexamplecom/v1"
	informers "github.com/20gu00/crd-controller/pkg/client/informers/externalversions/stableexamplecom/v1"
)

//informer workqueue
type Controller struct {
	informer  informers.CronTabInformer //informer lister
	workqueue workqueue.RateLimitingInterface
}

func NewController(informer informers.CronTabInformer) *Controller {
	//使用client(list watch)和Informer，初始化自定义控制器
	controller := &Controller{
		informer: informer,
		// WorkQueue 的实现，负责同步 Informer 和控制循环之间的数据
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CronTab"), //指定资源类型
	}

	klog.Info("建立crontab事件处理器")

	// informer 注册了三个 Handler,处理API对象（AddFunc、UpdateFunc 和 DeleteFunc）
	// 将该事件对应的 API 对象(Object key)加入到工作队列中
	//informer的事件处理器,注册事件监听函数
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueCronTab,
		//监听到更新事件,先判断资源版本是否不一致
		UpdateFunc: func(old, new interface{}) {
			//指针 update
			oldObj := old.(*crdv1.CronTab)
			newObj := new.(*crdv1.CronTab)
			//每一次操作,都会改变资源的版本(DeltaFifo)
			// 如果资源版本相同则不处理
			if oldObj.ResourceVersion == newObj.ResourceVersion {
				return
			}
			controller.enqueueCronTab(new)
		},
		DeleteFunc: controller.enqueueCronTabForDelete,
	})
	return controller
}

//运行
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash() //捕获崩溃
	defer c.workqueue.ShutDown()

	// 记录开始日志
	klog.Info("Starting CronTab control loop")
	klog.Info("Waiting for informer caches to sync")
	//等待缓存同步,informer 本地缓存 etcd(使用时从本地缓存获取数据)
	if ok := cache.WaitForCacheSync(stopCh, c.informer.Informer().HasSynced); !ok { //是否同步,至少一次完整的list
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh) //每一秒调用一次runWorker,知道通道关闭
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")
	return nil
}

// runWorker 是一个不断运行的方法，并且一直会调用 c.processNextWorkItem 从workqueue读取和读取消息 item
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// 从workqueue读取和读取消息
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get() //从通道中获取item
	if shutdown {                      //如果工作队列关闭了
		return false
	}

	//处理队列的Object
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj) //标记该资源对象已经处理
		var key string
		var ok bool
		//放入队列中的是Object的key,也就是Object的key形式期望从队列中获取的是key,是字符串
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj) //object断言成string失败,工作队列忘记该object
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		//正确获取key进行业务处理
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.workqueue.Forget(obj) //处理完成
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	//处理队列的Object失败
	if err != nil {
		runtime.HandleError(err)
		return true
	}
	return true
}

//处理逻辑 比如说创建pod,那就需要kubernetes的clientset
// 尝试从 Informer 维护的缓存中拿到key所对应的 CronTab 对象(从indexer中获取,或者通过informer的lister获取缓存信息)
func (c *Controller) syncHandler(key string) error {
	//key默认是namespace的索引器,类似default/pod,获取namespace和name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil { //切割key失败
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	//使用informer的lister获取资源(从indexer中获取)
	crontab, err := c.informer.Lister().CronTabs(namespace).Get(name)

	//从缓存中拿不到这个对象,那就意味着这个 CronTab 对象的 Key 是通过前面的“删除”事件添加进工作队列的。
	if err != nil {
		if errors.IsNotFound(err) { //找不到
			// 对应的 crontab 对象已经被删除了
			klog.Warningf("[CronTabCRD] %s/%s does not exist in local cache, will delete it from CronTab ...",
				namespace, name)
			klog.Infof("[CronTabCRD] deleting crontab: %s/%s ...", namespace, name)
			return nil
		}

		runtime.HandleError(fmt.Errorf("failed to get crontab by: %s/%s", namespace, name))
		return err
	}
	klog.Infof("[CronTabCRD] try to process crontab: %#v ...", crontab)
	return nil
}

//添加到队列 informer和workqueue之间
//将获取的对象转换成key
func (c *Controller) enqueueCronTab(obj interface{}) {
	var key string
	var err error
	//生成key的函数,默认的索引器是namespace
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	c.workqueue.AddRateLimited(key) //添加到队列
}

func (c *Controller) enqueueCronTabForDelete(obj interface{}) {
	var key string
	var err error
	//删除有两个状态
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	c.workqueue.AddRateLimited(key) //添加进队列
}
