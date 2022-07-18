package pkg

import (
	"k8s.io/api/networking/v1beta1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	informer "k8s.io/client-go/informers/core/v1"
	networkInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
)

type controller struct {
	cient kubernetes.Interface
	//主要用于与index交互，减轻与apiServer交互的压力
	ingressLister  v1.IngressLister
	servicesLister coreLister.ServiceLister
	queue          workqueue.RateLimitingInterface
}

func (c *controller) updateService(oldObj interface{}, newObj interface{}) {
	//todo 比较annotation是否相同
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)

}

func (c *controller) enqueue(obj interface{}) {
	//获取obj的key,后续worker中可以根据queue的key在index(本地存储)中找到对应资源
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	//放入queue
	c.queue.Add(key)
}
func (c *controller) deleteIngress(obj interface{}) {
	//获取ingress对象
	ingress := obj.(*v1beta1.Ingress)
	//根据ingress获取service
	service := v12.GetControllerOf(ingress)
	if service != nil {
		return
	}
	//判断service的类型是否是service
	if service.Kind != "Service" {
		return
	}
	//放入queue
	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	<-stopCh
}

func NewController(client kubernetes.Interface, serviceInformer informer.ServiceInformer, ingressInformer networkInformer.IngressInformer) controller {
	c := controller{cient: client,
		servicesLister: serviceInformer.Lister(),
		ingressLister:  ingressInformer.Lister(),
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"),
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
