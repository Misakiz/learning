package client_go

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetPods() {
	//获取kubeConfig,指定默认从home目录获取
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	//设置config的版本
	config.GroupVersion = &v1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	config.APIPath = "/api"
	if err != nil {
		panic(err)
	}
	//根据config初始化client
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}

	podList := v1.PodList{}
	//get data
	err = restClient.Get().Namespace("default").Resource("pods").Do(context.TODO()).Into(&podList)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(podList)
	}
}
