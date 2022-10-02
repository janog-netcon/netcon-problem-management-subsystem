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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

// ProblemEnvironmentReconciler reconciles a ProblemEnvironment object
type ProblemEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problemenvironments/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProblemEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = r.setLoggerFor(ctx, req)
	log := log.FromContext(ctx)

	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := r.Get(ctx, req.NamespacedName, &problemEnvironment); err != nil {
		log.V(1).Info("could not get ProblemEnvironment")
		return ctrl.Result{}, err
	}

	if problemEnvironment.DeletionTimestamp != nil {
		// there is nothing to do on ProblemEnvironment controller
		return ctrl.Result{}, nil
	}

	if owners := problemEnvironment.GetOwnerReferences(); len(owners) == 0 {
		return r.setOwnerReference(ctx, &problemEnvironment)
	}

	if problemEnvironment.Spec.WorkerName == "" {
		//
		return r.schedule(ctx, &problemEnvironment)
	}

	// ProblemEnvironment is already scheduled
	// so, there is nothing to do on ProblemEnvironment controller
	return ctrl.Result{}, nil
}

func (r *ProblemEnvironmentReconciler) setLoggerFor(ctx context.Context, req ctrl.Request) context.Context {
	return log.IntoContext(ctx, log.FromContext(ctx).WithValues(
		"namespace", req.Namespace,
		"name", req.Name,
	))
}

func (r *ProblemEnvironmentReconciler) update(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if err := r.Update(ctx, problemEnvironment); err != nil {
		// TODO(proelbtn): try to adopt exponentialBackOff
		log.Error(err, "failed to update")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	return ctrl.Result{}, nil
}

func (r *ProblemEnvironmentReconciler) updateStatus(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if err := r.Status().Update(ctx, problemEnvironment); err != nil {
		// TODO(proelbtn): try to adopt exponentialBackOff
		log.Error(err, "failed to update status")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	return ctrl.Result{}, nil
}

func (r *ProblemEnvironmentReconciler) setOwnerReference(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	problem := netconv1alpha1.Problem{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: problemEnvironment.Namespace,
		Name:      problemEnvironment.Spec.ProblemRef.Name,
	}, &problem); err != nil {
		message := "failed to find problem referenced by .spec.problemRef.Name"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionInitialized,
			metav1.ConditionFalse,
			"ProblemMissing",
			message,
		)

		// TODO(proelbtn): try to adopt exponentialBackOff
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 5 * time.Second})
	}

	// TODO(proelbtn): is it really okay to use SetControllerReference?
	if err := ctrl.SetControllerReference(&problem, problemEnvironment, r.Scheme); err != nil {
		log.Error(err, "failed to set controller reference")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	if res, err := r.update(ctx, problemEnvironment, ctrl.Result{}); err != nil {
		return res, err
	}

	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) schedule(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	workers := netconv1alpha1.WorkerList{}
	if err := r.List(ctx, &workers); err != nil {
		message := "failed to list Workers"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
			metav1.ConditionFalse,
			"WorkersMissing",
			message,
		)

		// TODO(proelbtn): try to adopt exponentialBackOff
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 5 * time.Second})
	}

	if len(workers.Items) == 0 {
		message := "there are no schedulable Workers"
		log.Info(message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
			metav1.ConditionFalse,
			"WorkersNotSchedulable",
			message,
		)

		// TODO(proelbtn): try to adopt exponentialBackOff
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 5 * time.Second})
	}

	// TODO(proelbtn): more intelligent scheduling algorithm
	problemEnvironment.Spec.WorkerName = workers.Items[0].Name

	if res, err := r.update(ctx, problemEnvironment, ctrl.Result{}); err != nil {
		return res, err
	}

	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
