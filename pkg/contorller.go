package pkg

import (
	"context"
	"fmt"
	v13 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informer "k8s.io/client-go/informers/core/v1"
	netInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v15 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

const (
	workNum  = 5
	maxRetry = 10
)

// 小写不暴露出去 通过 New方法返回实例
type controller struct {
	// 通过 client 可以创建 inforemer, 后续操作集群资源也是通过 client来做的
	client kubernetes.Interface
	// 提供 List 来获取对象的数据
	ingressList v15.IngressLister
	serviceList coreLister.ServiceLister

	// workque
	queue workqueue.RateLimitingInterface
}

// ingress 操作
func (c controller) deleteIngrees(obj interface{}) {
	// 通过 ingress 获取对应的 svc
	ingress := obj.(*v1.Ingress)
	ownerReference := v12.GetControllerOf(ingress)
	if ownerReference != nil {
		if ownerReference.Kind != "service" {
			return
		}
	} else {
		return
	}
	c.enqueue(obj)
}

// service 操作
func (c controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c controller) updataService(obj interface{}, obj2 interface{}) {
	if reflect.DeepEqual(obj, obj2) {
		return
	}
	c.enqueue(obj)
}

// 将对象对象增加到 workqueue 中
func (c controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	// workqueue 中存放的是obj 的 key
	c.queue.Add(key)
}

func (c controller) Run(stopCh chan struct{}) {
	// work 的逻辑
	for i := 0; i < workNum; i++ {
		// 启动一个协程 每隔一段时间就去调用方法，直到收到了 stopCh的消息
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

// 控制器协调的主要参与者
// 不断的从 workqueue中获取obj 并进行调谐操作
func (c controller) worker() {
	for c.processNextItem() {

	}
}

func (c controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		//如果 workqueue 已经关闭
		return false
	}
	defer c.queue.Done(item)
	key := item.(string)
	err := c.syncService(key)
	if err != nil {
		c.handlerError(key, err)
	}
	return true
}

func (c controller) syncService(key string) error {
	// 从key中读取信息
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Println("i am 1")
		return err
	}
	// 删除情况
	service, err := c.serviceList.Services(namespaceKey).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			fmt.Println("i am 2")
			return err
		}
	}
	// 新增情况 和 更新
	_, ok := service.GetAnnotations()["ingrees/http"]
	ingress, err := c.ingressList.Ingresses(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		fmt.Println("i am 3")
		return err
	}

	if ok && errors.IsNotFound(err) {
		// 创建  ingress  通过 clinet 操作
		ig := c.constructIngress(service)
		_, err := c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), ig, v12.CreateOptions{})
		if err != nil {
			fmt.Println("i am 4")
			return err
		}
	} else if !ok && ingress != nil {
		// 删除 ingress
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v12.DeleteOptions{})
		if err != nil {
			fmt.Println("i am 5")
			return err
		}
	}
	return nil
}

// 把 key 放到 workqueue 中，重试几次
func (c controller) handlerError(key string, err error) {
	if c.queue.NumRequeues(key) <= maxRetry {
		c.queue.AddRateLimited(key)
	}
	runtime.HandleError(err)
	c.queue.Forget(key)
	return
}

// 构造ingress，用来在集群中创建
func (c controller) constructIngress(service *v13.Service) *v1.Ingress {
	ingress := v1.Ingress{}
	name := service.Name
	namespace := service.Namespace

	ingress.OwnerReferences = []v12.OwnerReference{
		*v12.NewControllerRef(service, v13.SchemeGroupVersion.WithKind("service")),
	}
	ingress.Name = name
	ingress.Namespace = namespace
	pathType := v1.PathTypePrefix
	icn := "nginx"
	ingress.Spec = v1.IngressSpec{
		IngressClassName: &icn,
		Rules: []v1.IngressRule{
			{Host: "example.com",
				IngressRuleValue: v1.IngressRuleValue{
					HTTP: &v1.HTTPIngressRuleValue{
						Paths: []v1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: v1.IngressBackend{
								Service: &v1.IngressServiceBackend{
									Name: name,
									Port: v1.ServiceBackendPort{
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

// Informer 会提供 informer 和 lister
func NewController(client kubernetes.Interface, serviceInformer informer.ServiceInformer, ingressInformer netInformer.IngressInformer) controller {
	c := controller{
		client:      client,
		ingressList: ingressInformer.Lister(),
		serviceList: serviceInformer.Lister(),
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManger"),
	}
	// 为 informer 添加事件处理方法
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updataService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngrees,
	})
	return c
}
