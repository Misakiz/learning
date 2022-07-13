package client_go

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetPodsByCs() {
	//创建指定资源的client
	//获取kubeConfig,指定默认从home目录获取
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	//获取clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	//获取pod资源的的client
	get, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "busy-box-6f976f5dd-7m5rd", v1.GetOptions{})
	if err != nil {
		panic(err)
	} else {
		fmt.Println(get.Namespace)
	}
}
