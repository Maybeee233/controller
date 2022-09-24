package main

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {
	// 外部生成 config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// 尝试从集群内部挂载的 token 中获取 config对象
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("can't create config")
			return
		}
		config =  inClusterConfig
	}


}
