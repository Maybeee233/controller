package main

import (
	"awesomeProject10/pkg"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"time"
)

func main() {
	// 外部生成 config
	// 这边是用 homedir/.kube/config 生成 Config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// 尝试从集群内部挂载的 token 中获取 config对象
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("can't create config")
			return
		}
		config = inClusterConfig
	}

	// 生成 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("can't create clientset")
		return
	}
	// 创建 informer的工厂函数
	factory := informers.NewSharedInformerFactory(clientset, 0)
	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()
	controller := pkg.NewController(clientset, serviceInformer, ingressInformer)
	stopCh := make(chan struct{})
	// factory 的 start 方法就是启动 map 中的 inforemer, informer启动之后才会讲数据存储在本地
	factory.Start(stopCh)
	controller.Run(stopCh)

	time.Sleep(1000 * time.Second)
}
