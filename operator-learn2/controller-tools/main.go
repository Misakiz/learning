package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	v1 "zqa.test/pkg/apis/zqa.test/v1"
)

func main() {
	//初始化config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}

	config.APIPath = "/apis/"

	config.NegotiatedSerializer = v1.Codecs.WithoutConversion()

	config.GroupVersion = &v1.GroupVersion
	//根据config初始化
	client, err := rest.RESTClientFor(config)
	if err != nil {
		log.Fatalln(err)
	}

	foo := v1.Foo{}
	err = client.Get().Namespace("default").Resource("foos").Name("test-crd").Do(context.TODO()).Into(&foo)
	if err != nil {
		log.Fatalln(err)
	}

	deepCopy := foo.DeepCopy()
	deepCopy.Spec.Name = "tet-2"
	fmt.Println(foo.Spec.Name)
	fmt.Println(foo.Spec.Replicas)
	fmt.Println(deepCopy.Spec.Name)
}
