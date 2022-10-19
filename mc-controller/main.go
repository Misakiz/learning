package main

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"mc-operator/pkg"
)

func main() {

	//1. config
	//2. client
	//3. informer
	//4. add event handler
	//5. informer.Start

	//根据kubeConfig文件文件连接集群
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		//如果找不到则通过pod目录下到token和ca证书连接
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("can't get config")
		}
		config = inClusterConfig
	}

	//创建默认资源的clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("can't create client")
	}

	//创建factory，工厂模式
	factory := informers.NewSharedInformerFactory(clientSet, 0)
	//创建deploymentInformer
	deploymentInformer := factory.Apps().V1().Deployments()
	controller := pkg.NewController(clientSet, deploymentInformer)

	stopCh := make(chan struct{})
	//启动informer
	factory.Start(stopCh)
	//等待informer同步完成后
	factory.WaitForCacheSync(stopCh)

	controller.Run(stopCh)

}
