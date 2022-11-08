# gateway-operator
go语言基于kubebuilder实现一个gateway的operator，gateway是我开发的一款网关，项目链接https://github.com/20gu00/gateway  
operator使gateway网关更贴合云原生环境，自动检测调谐gateway的deploy和service等  

## 运行
make  
make install  
make run
kubectl apply -f config/sample/

## 其他
我将crd和operator一块生成了个镜像，可以用容器方式运行operator  
make deploy IMG=010101010007/gatewayoperator:v1  


其他命令参考：  
make docker-build IMG=010101010007/gatewayoperator:v1  
make docker-push IMG=010101010007/gatewayoperator:v1  
