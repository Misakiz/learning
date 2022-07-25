package main

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	clientset "operator-crd/pkg/generated/clientset/versioned"
	"operator-crd/pkg/generated/informers/externalversions"
)

func main() {
	//获取config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}
	//创建clientset
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}
	//通过clientset获取default命名空间下的foo资源
	list, err := clientset.CrdV1().Foos("default").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	//遍历foo资源的名字
	for _, foo := range list.Items {
		println(foo.Name)
	}
	//创建informer的Factory
	factory := externalversions.NewSharedInformerFactory(clientset, 0)

	factory.Crd().V1().Foos().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {

		}})
}
