package client_go

import (
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

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
	factory := informers.NewSharedInformerFactory(clientSet, 0)

	podInformer := factory.Core().V1().Pods().Informer()

	//添加处理事件
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj interface{}) {
		fmt.Println("ADD")
	}, UpdateFunc: func(oldObj, newObj interface{}) {
		fmt.Println("update")
	}, DeleteFunc: func(obj interface{}) {
		fmt.Println("delete")
	}})

	//start informer
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
}
