package pkg

import (
	"context"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informer "k8s.io/client-go/informers/core/v1"
	networkInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

const workNum = 5
const maxTry = 10

type controller struct {
	client kubernetes.Interface
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
	ingress := obj.(*v14.Ingress)
	//根据ingress获取service
	ownerReference := v12.GetControllerOf(ingress)
	if ownerReference == nil {
		return
	}
	//判断service的类型是否是service
	if ownerReference.Kind != "Service" {
		return
	}
	//放入queue
	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	for i := 0; i < workNum; i++ {
		//保证5个goroutine一直执行
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

func (c *controller) worker() {
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	//获取queue中的key
	item, shutdown := c.queue.Get()
	//如果该队列状态为shutdown
	if shutdown {
		return false
	}
	//处理完该key后,从queue中移除
	defer c.queue.Done(item)
	key := item.(string)
	err := c.syncService(key)
	if err != nil {
		c.handlerError(key, err)
		return false
	}
	return true
}

func (c *controller) syncService(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	//删除
	//从servicesLister获取service资源
	service, err := c.servicesLister.Services(namespaceKey).Get(name)
	//如果资源不存在会返回NotFound错误,获取错误不做处理
	if errors.IsNotFound(err) {
		return nil
	}
	//报错返回错误
	if err != nil {
		return err
	}

	//新增或者删除
	//todo 先判断service下有没有Annotations,再判断service资源下的annotation是否含有"ingress/http"的key
	_, ok := service.GetAnnotations()["ingress/http"]
	ingress, err := c.ingressLister.Ingresses(namespaceKey).Get(name)
	//缓存中有ingress资源但是获取报错
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	//有service资源,但是没有ingress资源
	if ok && errors.IsNotFound(err) {
		//创建ingress对象
		ig := c.constructIngress(service)
		//创建ingress
		_, err := c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), ig, v12.CreateOptions{})
		if err != nil {
			return err
		}
		//没有service,但是有ingress资源
	} else if !ok && ingress != nil {
		//删除ingress
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v12.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

//处理错误
func (c *controller) handlerError(item string, err error) {
	//重试小于10次
	if c.queue.NumRequeues(item) <= maxTry {
		//worker中报错则加回限速队列重试
		c.queue.AddRateLimited(item)
		return
	}
	runtime.HandleError(err)
	c.queue.Forget(item)
}

//创建ingress对象
func (c *controller) constructIngress(service *v13.Service) *v14.Ingress {
	ingress := v14.Ingress{}
	ingress.ObjectMeta.OwnerReferences = []v12.OwnerReference{
		*v12.NewControllerRef(service, v13.SchemeGroupVersion.WithKind("Service")),
	}
	ingress.Name = service.Name
	ingress.Namespace = service.Namespace
	pathType := v14.PathTypePrefix
	inc := "nginx"
	ingress.Spec = v14.IngressSpec{
		IngressClassName: &inc,
		Rules: []v14.IngressRule{
			v14.IngressRule{
				Host: "zqa.com",
				IngressRuleValue: v14.IngressRuleValue{
					HTTP: &v14.HTTPIngressRuleValue{
						Paths: []v14.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: v14.IngressBackend{
									Service: &v14.IngressServiceBackend{
										Name: service.Name,
										Port: v14.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &ingress
}

func NewController(client kubernetes.Interface, serviceInformer informer.ServiceInformer, ingressInformer networkInformer.IngressInformer) controller {
	c := controller{client: client,
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
		DeleteFunc: c.deleteIngress,
	})

	return c
}
