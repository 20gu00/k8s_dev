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

	appv1 "github.com/20gu00/redis-operator/api/v1"
)

// RedisSingleReconciler reconciles a RedisSingle object
type RedisSingleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rabc:group=apps,Resources=deployments,verbs=get;list;watch;patch;create;update;delete
// +kubebuilder:rabc:group=core,Resources=services,verbs=get;list;watch;patch;create;update;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=redissingles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=redissingles/status,verbs=get;update;patch

func (r *RedisSingleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("redissingle", req.NamespacedName)

	var redisSingle appv1.RedisSingle
	if err := r.Get(ctx, req.NamespacedName, &redisSingle); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var svc corev1.Service
	svc.Name = redisSingle.Name
	svc.Namespace = redisSingle.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			MutateSvc(&redisSingle, &svc)
			return controllerutil.SetControllerReference(&redisSingle, &svc, r.Scheme)
		})
		log.Info("createOrUpdata result", "service", or)
		return err
	}); err != nil {
		return ctrl.Result{}, nil
	}

	var deploy appsv1.Deployment
	deploy.Name = redisSingle.Name
	deploy.Namespace = redisSingle.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
			MutateDeployment(&redisSingle, &deploy)
			return controllerutil.SetControllerReference(&redisSingle, &deploy, r.Scheme)
		})
		log.Info("createorupdate", "deployment", or)
		return err
	}); err != nil {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *RedisSingleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.RedisSingle{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
