package client_go

import (
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// InformerTest informer->eventHander->workQueue（限速worker消费速度比较慢）->worker（对比obj的当前状态status和spec目标状态是否一致，不一致则处理）
func InformerTest() {
	//创建指定资源的client
	//获取kubeConfig,指定默认从home目录获取
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	//获取clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	//获取informer
	//factory := informers.NewSharedInformerFactory(clientSet, 0)

	//指定命名空间
	factory := informers.NewSharedInformerFactoryWithOptions(clientSet, 0, informers.WithNamespace("default"))
	podInformer := factory.Core().V1().Pods().Informer()

	//创建一个限速器
	rateLimitingQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "controller")

	//添加处理事件
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj interface{}) {
		//根据obj获取key
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			fmt.Println("找不到keyFuc")
		}
		//加入限速队列
		rateLimitingQueue.AddRateLimited(key)
		fmt.Println("ADD")
	}, UpdateFunc: func(oldObj, newObj interface{}) {
		//根据obj获取key
		key, err := cache.MetaNamespaceKeyFunc(newObj)
		if err != nil {
			fmt.Println("找不到keyFuc")
		}
		//加入限速队列
		rateLimitingQueue.AddRateLimited(key)
		fmt.Println("update")
	}, DeleteFunc: func(obj interface{}) {
		//根据obj获取key
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			fmt.Println("找不到keyFuc")
		}
		//加入限速队列
		rateLimitingQueue.AddRateLimited(key)
		fmt.Println("delete")
	}})

	//start informer
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	<-stopCh
}
