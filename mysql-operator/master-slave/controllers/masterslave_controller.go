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

	mv1 "github.com/20gu00/masterslave/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cjqappv1 "github.com/20gu00/masterslave/api/v1"
)

// MasterSlaveReconciler reconciles a MasterSlave object
type MasterSlaveReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//resources plural
//group.domain

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cjqapp.cjq.io,resources=masterslaves,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cjqapp.cjq.io,resources=masterslaves/status,verbs=get;update;patch

func (r *MasterSlaveReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("masterslave", req.NamespacedName)

	//实例化(Object)
	var masterSlave mv1.MasterSlave
	//从缓存中获取
	if err := r.Get(ctx, req.NamespacedName, &masterSlave); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var svc corev1.Service
	svc.Name = masterSlave.Name
	svc.Namespace = masterSlave.Namespace
	//与众多资源一样
	//masterSlave.GetNamespace()

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		//crd_client svc func
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			MutateSvc(&masterSlave, &svc)
			return controllerutil.SetControllerReference(&masterSlave, &svc, r.Scheme)
		})
		log.Info("createOrUpdate mysql service", "service", or) //调谐结果
		return err
	}); err != nil {
		return ctrl.Result{}, nil
	}

	var readSvc corev1.Service
	readSvc.Name = masterSlave.Name
	readSvc.Namespace = masterSlave.Namespace
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &readSvc, func() error {
			MutateReadSvc(&masterSlave, &readSvc)
			return controllerutil.SetControllerReference(&masterSlave, &readSvc, r.Scheme)
		})
		log.Info("createOrUpdate mysql read-service", "Service", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	var sts appsv1.StatefulSet
	sts.Name = masterSlave.Name
	sts.Namespace = masterSlave.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &sts, func() error {
			MutateStatefulset(&masterSlave, &sts)
			return controllerutil.SetControllerReference(&masterSlave, &sts, r.Scheme)
		})
		log.Info("createOrUpdate statefulset", "Statefulset", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

//监听crd所控制的资源statefulset service
func (r *MasterSlaveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cjqappv1.MasterSlave{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
