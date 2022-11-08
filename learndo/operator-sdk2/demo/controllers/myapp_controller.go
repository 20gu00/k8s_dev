/*
Copyright 2022 cjq.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1 "github.com/20gu00/operator-sdk-demo/api/v1"
)

var (
	oldSpecAnnotation = "old/spec"
)

// MyappReconciler reconciles a Myapp object
//每一个调谐器器都需要记录日志，并且能够获取对象，所以可以直接使用
type MyappReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//生成对应的RBAC,集群使用,主要是watch

// +kubebuilder:rbac:groups=apps,resources=Deploymets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=Services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=Myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=Myapps/status,verbs=get;update;patch

//不断的 watch 资源的状态，然后根据状态的不同去实现各种操作逻辑
//对单个对象进行调谐
func (r *MyappReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//大多数控制器需要一个日志句柄和一个上下文，所以在Reconcile中将他们初始化.上下文用来允许取消请求
	ctx := context.Background()
	log := r.Log.WithValues("Myapp", req.NamespacedName)

	//业务逻辑实现

	//获取Myapp实例
	//GV
	var Myapp appv1.Myapp
	//获取Myapp
	//controller-runtime资源如队列是request,这里拿到Object key
	err := r.Get(ctx, req.NamespacedName, &Myapp) //r get
	if err != nil {
		//Myapp被删除的时候,这种也是错误,可以忽略
		//crd资源找不到不是通过重试能够修复的
		//获取myapp资源出错
		if client.IgnoreNotFound(err) != nil { //IgnoreNotFound notfound时返回nil,其他错误照常返回错误值
			return ctrl.Result{}, err //错误,重试
		}

		//not found
		return ctrl.Result{}, nil
	}

	//调谐,获取当前状态,和期望状态比较
	var deploy appsv1.Deployment
	deploy.Name = Myapp.Name
	deploy.Namespace = Myapp.Namespace
	//处理deploy
	//CreateOrUpdate逻辑和前一个版本的一样,创建更新
	//前一个版本是通过实现个annotation来判断更新的,这个是用equality.Semantic.DeepEqual比较判断,更加语义化
	or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
		//调谐函数(之前的NewDeployment等)
		MutateDeployment(&Myapp, &deploy)
		//owner 被控制的 注册表
		return controllerutil.SetControllerReference(&Myapp, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err //重试
	}

	//打印调谐结果
	log.Info("CreateOrUpdate", "Deployment", or)

	var svc corev1.Service
	svc.Name = Myapp.Name
	svc.Namespace = Myapp.Namespace
	//处理deploy
	//CreateOrUpdate逻辑和前一个版本的一样,创建更新
	//前一个版本是通过实现个annotation来判断更新的,这个是用equality.Semantic.DeepEqual比较判断,更加语义化
	or, err = ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
		//调谐函数(之前的NewDeployment等)
		MutateSvc(&Myapp, &svc)
		//owner 被控制的 注册表
		//删除owner其他资源也会被删除
		return controllerutil.SetControllerReference(&Myapp, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err //重试
	}

	//打印调谐结果
	log.Info("CreateOrUpdate", "Service", or)

	return ctrl.Result{}, nil
}

//将Reconcile添加到manager中，这样当manager启动时它就会被启动
//watch所管理的资源 deploy service 删除重建
func (r *MyappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Myapp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
