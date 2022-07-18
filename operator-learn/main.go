package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"operator-learn/pkg"
)

func main() {
	//创建指定资源的client

	//获取kubeConfig,指定默认从home目录获取
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		//根据/var/run目录下的K8s token和ca文件获取config
		clusterConfig, err := rest.InClusterConfig()
		if err != nil {
			logrus.Errorln("can't get config")
		}
		config = clusterConfig
	}
	//根据config初始化clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorln("can't create client")
	}

	//获取informer的Factory
	factory := informers.NewSharedInformerFactory(clientSet, 0)
	//获取services的informer
	servicesInformer := factory.Core().V1().Services()
	//获取ingress的informer
	ingressesInformer := factory.Networking().V1().Ingresses()
	controller := pkg.NewController(clientSet, servicesInformer, ingressesInformer)

	//start informer
	stopCh := make(chan struct{})
	//会将service和ingress资源存放在本地index
	factory.Start(stopCh)
	// 等待同步资源完成
	factory.WaitForCacheSync(stopCh)

	controller.Run(stopCh)

}
