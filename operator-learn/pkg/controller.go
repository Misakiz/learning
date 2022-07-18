package pkg

import (
	informer "k8s.io/client-go/informers/core/v1"
	networkInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
)

type controller struct {
	cient kubernetes.Interface
	//主要用于与index交互，减轻与apiServer交互的压力
	ingressLister  v1.IngressLister
	servicesLister coreLister.ServiceLister
}

func (c *controller) updateService(oldObj interface{}, newObj interface{}) {

}

func (*controller) addService(obj interface{}) {

}

func (c *controller) deleteIngress(obj interface{}) {

}

func (c *controller) Run(stopCh chan struct{}) {
	<-stopCh
}

func NewController(client kubernetes.Interface, serviceInformer informer.ServiceInformer, ingressInformer networkInformer.IngressInformer) controller {
	c := controller{cient: client,
		servicesLister: serviceInformer.Lister(),
		ingressLister:  ingressInformer.Lister(),
	}

	//创建service的事件处理handler
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	//创建ingress的事件处理handler
	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.deleteIngress,
	})

	return c
}
