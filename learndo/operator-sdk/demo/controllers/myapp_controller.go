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
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	//成功获取该资源对象
	log.Info("fetch Myapp objects", "Myapp", Myapp)

	//调谐 存在->更新
	// 如果不存在，则创建关联资源 如果存在，判断是否需要更新 如果需要更新，则直接更新 如果不需要更新，则正常返回
	deploy := &appsv1.Deployment{}
	//获取deploy
	if err := r.Get(ctx, req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) { //not found返回true,没有该资源->创建
		//关联 Annotations
		//将spec放到annotation中
		//第一次进来,拿到spec编码  比较
		data, _ := json.Marshal(Myapp.Spec) //编码
		if Myapp.Annotations != nil {
			Myapp.Annotations[oldSpecAnnotation] = string(data)
		} else {
			Myapp.Annotations = map[string]string{oldSpecAnnotation: string(data)}
		}

		if err := r.Client.Update(ctx, &Myapp); err != nil {
			return ctrl.Result{}, err
		}

		// 创建关联资源

		//创建Deployment
		deploy := NewDeploy(&Myapp)
		if err := r.Client.Create(ctx, deploy); err != nil {
			return ctrl.Result{}, err
		}

		//创建Service
		service := NewService(&Myapp)
		if err := r.Create(ctx, service); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	oldspec := appv1.MyappSpec{}
	if err := json.Unmarshal([]byte(Myapp.Annotations[oldSpecAnnotation]), &oldspec); err != nil {
		return ctrl.Result{}, err
	}
	// 当前规范与旧的对象不一致，则需要更新
	if !reflect.DeepEqual(Myapp.Spec, oldspec) { //比较 slice map map可以顺序不一致
		// 更新关联资源
		newDeploy := NewDeploy(&Myapp)
		oldDeploy := &appsv1.Deployment{}
		if err := r.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
			return ctrl.Result{}, err
		}
		oldDeploy.Spec = newDeploy.Spec
		if err := r.Client.Update(ctx, oldDeploy); err != nil {
			return ctrl.Result{}, err
		}

		newService := NewService(&Myapp)
		oldService := &corev1.Service{}
		if err := r.Get(ctx, req.NamespacedName, oldService); err != nil {
			return ctrl.Result{}, err
		}
		// 需要指定 ClusterIP 为之前的，不然更新会报错
		//更新了资源,但该资源对象没有被创建,kube-proxy没有分配VIP给service,无法比较
		newService.Spec.ClusterIP = oldService.Spec.ClusterIP
		oldService.Spec = newService.Spec
		if err := r.Client.Update(ctx, oldService); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	//返回一个空的结果，没有错误，成功地对这个对象进行了调谐,在有一些变化之前不需要再尝试调谐
	return ctrl.Result{}, nil
}

//将Reconcile添加到manager中，这样当manager启动时它就会被启动
func (r *MyappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Myapp{}).
		Complete(r)
}
