# Ingress-manager-controller



## 需求

![image-20220925132333831](/Users/lxavier/Library/Application Support/typora-user-images/image-20220925132333831.png)

> 即用户可以通过在 SVC 上面的 annotation 字段上面添加 `ingress/http` 字段，自动化的创建相应的 `Nginx-Ingress`





## 环境

+ Minikube: v1.23.1
+ client-go: v0.23.3



## 运行

1、部署 ingress-controller

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.3.1/deploy/static/provider/cloud/deploy.yaml
```



2、部署 manager  和 测试的 pod 以及 svc

```shell
kubectl apply -f manifests 
```



3、测试 ingress-manager 功能

+ svc 上带有 annotation 为 ingress/http 的，Ingress-Manager 会自动拉起 Ingress
+ 删除 svc 上面的 ingress/http 的annotation 后， Ingress-Manager 也会自动的删除 Ingress
+ 删除 annotion为 ingress/http 的 svc 对应的 ingress 后， Ingress-Manager 会自动拉起 Ingress

