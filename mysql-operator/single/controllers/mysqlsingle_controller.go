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

	// your logic here

	return ctrl.Result{}, nil
}

func (r *MysqlSingleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cjqappv1.MysqlSingle{}).
		Complete(r)
}
