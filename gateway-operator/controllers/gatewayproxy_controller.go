/*


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

	gogatewayv1 "github.com/20gu00/gateway-operator/api/v1"
)

// GatewayProxyReconciler reconciles a GatewayProxy object
type GatewayProxyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gogateway.cjq.io,resources=gatewayproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gogateway.cjq.io,resources=gatewayproxies/status,verbs=get;update;patch

func (r *GatewayProxyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("gatewayproxy", req.NamespacedName)

	var gatewayProxy gogatewayv1.GatewayProxy
	if err := r.Get(ctx, req.NamespacedName, &gatewayProxy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var svc corev1.Service
	svc.Name = gatewayProxy.Name
	svc.Namespace = gatewayProxy.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			MutateProxySvc(&gatewayProxy, &svc)
			return controllerutil.SetControllerReference(&gatewayProxy, &svc, r.Scheme)
		})
		log.Info("CreateOrUpdate的结果", "Service", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	var deploy appsv1.Deployment
	deploy.Name = gatewayProxy.Name
	deploy.Namespace = gatewayProxy.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
			MutateProxyDeploy(&gatewayProxy, &deploy)
			return controllerutil.SetControllerReference(&gatewayProxy, &deploy, r.Scheme)
		})
		log.Info("CreateOdUpdate结果", "Deployment", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GatewayProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gogatewayv1.GatewayProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
