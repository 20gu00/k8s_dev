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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cjqappv1 "github.com/20gu00/mysql-single-operator/api/v1"
)

// MysqlSingleReconciler reconciles a MysqlSingle object
type MysqlSingleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cjqapp.cjq.io,resources=mysqlsingles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cjqapp.cjq.io,resources=mysqlsingles/status,verbs=get;update;patch

func (r *MysqlSingleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mysqlsingle", req.NamespacedName)

	//实例化crd资源
	var mysqlSingle cjqappv1.MysqlSingle
	//从缓存中获取该资源对象
	//ctx key object(一般使用namespace索引器作为索引)
	if err := r.Get(ctx, req.NamespacedName, &mysqlSingle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err) //忽略从缓存中找不到的情况,比如该资源对象被删除了,缓存和etcd保持一致)
	}

	//CreateOrUpdate

	//service
	var svc corev1.Service
	svc.Name = mysqlSingle.Name
	svc.Namespace = mysqlSingle.Namespace

	//重试
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		//ctx crd资源的client 处理的资源对象 调谐函数
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			//调谐函数
			MutateSvc(&mysqlSingle, &svc)
			//设置关系,crd资源控制的svc
			//owner 被控制的资源对象 注册表
			return controllerutil.SetControllerReference(&mysqlSingle, &svc, r.Scheme)
		})
		log.Info("调谐结果Result", "Service", or)
		//CreateOrUpdate出错
		return err
	}); err != nil {
		return ctrl.Result{}, err //出错重试
	}

	//deployment

	var deploy appsv1.Deployment
	deploy.Name = mysqlSingle.Name
	deploy.Namespace = mysqlSingle.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, deploy, func() error {
			MutateDeployment(&mysqlSingle, &deploy)
			return controllerutil.SetControllerReference(&mysqlSingle, &deploy, r.Scheme)
		})
		return err
	}); err != nil {
		//调谐失败
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

//对 deployment 和 Service 这两种资源进行 Watch，因为当这两个资源出现变化的时候我们也需要去重新进行调谐
//只需要 Watch 被 mysqlSingle 控制的这部分对象
//将 Service 或者 deployment 删除了也会自动重新调谐然后重建出来
func (r *MysqlSingleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cjqappv1.MysqlSingle{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
