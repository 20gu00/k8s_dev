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
	gogatewayv1 "github.com/20gu00/gateway-operator/api/v1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GatewayMarketReconciler reconciles a GatewayMarket object
type GatewayMarketReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps,resources=deploymrnts,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=gogateway.cjq.io,resources=gatewaymarkets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gogateway.cjq.io,resources=gatewaymarkets/status,verbs=get;update;patch

func (r *GatewayMarketReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("gatewaymarket", req.NamespacedName)

	var gatewayMarket gogatewayv1.GatewayMarket
	if err := r.Get(ctx, req.NamespacedName, &gatewayMarket); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var svc corev1.Service
	svc.Name = gatewayMarket.Name
	svc.Namespace = gatewayMarket.Namespace

	//重试多次之后还不行就报错
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			MutateSvc(&gatewayMarket, &svc)
			return controllerutil.SetControllerReference(&gatewayMarket, &svc, r.Scheme)
		})
		log.Info("CreateOrUpdate的结果", "Service", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	var deploy appsv1.Deployment
	deploy.Name = gatewayMarket.Name
	deploy.Namespace = gatewayMarket.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
			MutateDeploy(&gatewayMarket, &deploy)
			return controllerutil.SetControllerReference(&gatewayMarket, &deploy, r.Scheme)
		})
		log.Info("CreateOdUpdate结果", "Deployment", or)
		return err
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GatewayMarketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gogatewayv1.GatewayMarket{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
