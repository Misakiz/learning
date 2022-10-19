package pkg

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsLister "k8s.io/client-go/listers/apps/v1"
	coreLister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"strings"
	"time"
)

const workNum = 3
const maxRetry = 10

type controller struct {
	client           kubernetes.Interface
	serviceLister    coreLister.ServiceLister
	deploymentLister appsLister.DeploymentLister
	queue            workqueue.RateLimitingInterface
}

//更新deployment事件
func (c *controller) updateDeployment(oldObj interface{}, newObj interface{}) {

	//比较旧deploy和新deploy是否相同，不相同return
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	//相同加入缓冲队列
	c.enqueue(newObj)
}

//新增deployment事件
func (c *controller) addDeployment(obj interface{}) {

	//将该对象加到缓冲队列
	c.enqueue(obj)
}

//加到workQueue 缓冲 生产消费速率
func (c *controller) enqueue(obj interface{}) {

	//获取对象的key,返回nameSpace/name
	key, err := cache.MetaNamespaceKeyFunc(obj)
	namespaceKey, _, err := cache.SplitMetaNamespaceKey(key)
	if C7nNameSpace == "" {
		logrus.Error("获取不到命名空间环境变量")
		return
	}
	if namespaceKey != C7nNameSpace {
		return
	}

	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

// Run Controller启动
func (c *controller) Run(stopCh chan struct{}) {
	//启动3个worker处理
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

// worker处理
func (c *controller) worker() {
	for c.processNextItem() {

	}
}

//
func (c *controller) processNextItem() bool {
	//从workerQueue获取对象
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	//处理完后从workerQueue去除该对象

	defer c.queue.Done(item)
	//interface -> string
	itemKey := item.(string)

	err := c.syncSrmService(itemKey)
	if err != nil {
		c.handlerError(itemKey, err)
	}
	return true
}

func (c *controller) syncSrmService(key string) error {
	//根据指定命名空间同步Service
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	deployment, err := c.deploymentLister.Deployments(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		logrus.Infoln(err)
		return err
	}
	//判断是否获取到deploy
	if errors.IsNotFound(err) {
		logrus.Error("找不到对应Deployment,Deployment名字为: ", deployment)
		return nil
	}

	deploymentName := deployment.GetName()
	//判断deploy名字是否以srm开头
	if strings.HasPrefix(deploymentName, "srm") {
		//返回 '-' 最后出现的index
		index := strings.LastIndex(deploymentName, "-")

		if index == -1 {
			logrus.Error("字符串搜索失败")
			return nil
		}

		//获取对应service名字,用于创建service
		serviceName := deploymentName[0:index]
		service, err := c.client.CoreV1().Services(namespaceKey).Get(context.TODO(), serviceName, metav1.GetOptions{})
		//获取service报错，且报错为非notFound报错
		if err != nil && !errors.IsNotFound(err) {
			logrus.Error(err)
			return err
		} else if err == nil {
			logrus.Infoln("找到对应Service,Service名字为: " + service.GetName())
			c.checkEPIsBind(namespaceKey, serviceName, deploymentName)
			return nil
		}
		//判断是否获取到deploy
		if errors.IsNotFound(err) {
			logrus.Infoln("找不到对应Service,Service名字为: ", serviceName)
			logrus.Infoln("开始创建Service")
			//判断deployment的labels的app值是否为空,
			_, ok := deployment.GetLabels()["app"]
			//deployment存在labels为app的值且找不到对应service
			if ok && errors.IsNotFound(err) {
				//创建对应service
				//根据deploy的模板获取deploy
				serviceCreate := c.constructService(deployment, serviceName)
				//创建Service
				_, err := c.client.CoreV1().Services(namespaceKey).Create(context.TODO(), serviceCreate, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				logrus.Infoln("创建Service成功,service名字为" + serviceName + "对应Deployment名字为" + deploymentName)
				c.checkEPIsBind(namespaceKey, serviceName, deploymentName)
			}
		}

	}
	return nil
}

func (c *controller) checkEPIsBind(namespaceKey string, serviceName string, deploymentName string) {
	//判断ep对象是否绑定成功
	endpoints, _ := c.client.CoreV1().Endpoints(namespaceKey).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if len(endpoints.Subsets) == 0 {
		logrus.Error("ep绑定失败！！！请检查对应配置！。service名字为" + serviceName + "对应Deployment名字为" + deploymentName)
	}
	logrus.Infoln("ep绑定pod成功！！！请检查对应配置！。service名字为" + serviceName + "对应Deployment名字为" + deploymentName)
}

func (c *controller) handlerError(key string, err error) {
	//最大重试次数
	if c.queue.NumRequeues(key) <= maxRetry {
		c.queue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	c.queue.Forget(key)
}

// NewController 初始化controller
func NewController(client kubernetes.Interface, deploymentInformer appsinformer.DeploymentInformer) controller {
	c := controller{
		client:           client,
		deploymentLister: deploymentInformer.Lister(),
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deploymentManager"),
	}
	//添加deployment资源事件处理方法
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDeployment,
		UpdateFunc: c.updateDeployment,
	})
	return c
}

//创建ingress对象
func (c *controller) constructService(deployment *appv1.Deployment, serviceName string) *corev1.Service {
	SelectorMap := make(map[string]string, 3)

	SelectorMap["app"] = deployment.Spec.Selector.MatchLabels["app"]
	service := corev1.Service{}
	service.Name = serviceName
	service.Namespace = deployment.Namespace
	service.Spec.Type = corev1.ServiceTypeClusterIP
	var ServicePortMap = make([]corev1.ServicePort, 1)
	if len(deployment.Spec.Template.Spec.Containers) != 0 {
		containers := deployment.Spec.Template.Spec.Containers
		for _, container := range containers {
			if len(container.Ports) != 0 {

				for portNum, ContainerPort := range container.Ports {
					fmt.Println(portNum, ContainerPort)
					ServicePortMap[portNum] = corev1.ServicePort{
						Name:     ContainerPort.Name,
						Protocol: ContainerPort.Protocol,
						TargetPort: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: int32(ContainerPort.ContainerPort),
						},
						Port: ContainerPort.ContainerPort,
					}

				}
			}

		}
	}

	service.Spec = corev1.ServiceSpec{
		Selector: SelectorMap,
		Ports:    ServicePortMap,
	}

	return &service
}
