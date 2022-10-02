/*
Copyright 2022 NETCON developers.

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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

// ProblemReconciler reconciles a ProblemEnvironment object
type ProblemReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProblemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	problem := netconv1alpha1.Problem{}
	if err := r.Get(ctx, req.NamespacedName, &problem); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.V(1).Info("could not get ProblemEnvironment")
		}
		return ctrl.Result{}, err
	}

	// TODO(proelbtn): the actual algorithm of scheduler

	return ctrl.Result{}, nil
}

func (r *ProblemReconciler) mapFromProblemEnvironment(o client.Object) []reconcile.Request {
	problemEnvironment := o.(*netconv1alpha1.ProblemEnvironment)
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: problemEnvironment.Namespace,
				Name:      problemEnvironment.Spec.ProblemRef.Name,
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.Problem{}).
		Watches(
			&source.Kind{Type: &netconv1alpha1.ProblemEnvironment{}},
			handler.EnqueueRequestsFromMapFunc(handler.MapFunc(r.mapFromProblemEnvironment)),
		).
		Complete(r)
}
