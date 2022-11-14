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

	appv1 "github.com/20gu00/redis-sts/api/v1"
)

// RedisStsReconciler reconciles a RedisSts object
type RedisStsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=redissts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.cjq.io,resources=redissts/status,verbs=get;update;patch

func (r *RedisStsReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("redissts", req.NamespacedName)

	var redisSts appv1.RedisSts
	if err := r.Get(ctx, req.NamespacedName, &redisSts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var svc corev1.Service
	svc.Name = redisSts.Name
	svc.Namespace = redisSts.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
			MutateSvc(&redisSts, &svc)
			return controllerutil.SetControllerReference(&redisSts, &svc, r.Scheme)
		})
		log.Info("createOrUpdata result", "service", or)
		return err
	}); err != nil {
		return ctrl.Result{}, nil
	}

	var sts appsv1.StatefulSet
	sts.Name = redisSts.Name
	sts.Namespace = redisSts.Namespace

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		or, err := ctrl.CreateOrUpdate(ctx, r, &sts, func() error {
			MutateStatefulset(&redisSts, &sts)
			return controllerutil.SetControllerReference(&redisSts, &sts, r.Scheme)
		})
		log.Info("createorupdate", "statefulset", or)
		return err
	}); err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RedisStsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.RedisSts{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
