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
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problems,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProblemEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := r.Get(ctx, req.NamespacedName, &problemEnvironment); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.V(1).Info("could not get ProblemEnvironment")
		}
		return ctrl.Result{}, err
	}

	if problemEnvironment.DeletionTimestamp != nil {
		log.Info("being deleted, ignoring")
		// there is nothing to do when deleting ProblemEnvironment
		return ctrl.Result{}, nil
	}

	if problemEnvironment.Spec.WorkerName == "" {
		log.Info("not scheduled yet, scheduling")
		return r.schedule(ctx, &problemEnvironment)
	}

	log.Info("already scheduled, confirming schedule")
	return r.confirmSchedule(ctx, &problemEnvironment)
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
		return ctrl.Result{RequeueAfter: 3 * time.Second}, err
	}
	return ctrl.Result{}, nil
}

func (r *ProblemEnvironmentReconciler) schedule(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.V(1).Info("fetching Worker list for scheduling")
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

	log.Info("scheduled", "newWorkerName", problemEnvironment.Spec.WorkerName)
	return r.update(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) confirmSchedule(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.V(1).Info("fetching Worker scheduled")
	worker := netconv1alpha1.Worker{}
	if err := r.Get(ctx, types.NamespacedName{
		Name: problemEnvironment.Spec.WorkerName,
	}, &worker); err != nil {
		message := "failed to get Worker"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
			metav1.ConditionFalse,
			"WorkersMissing",
			message,
		)

		// TODO(proelbtn): try to adopt exponentialBackOff
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 3 * time.Second})
	}

	log.Info("confirmed", "workerName", problemEnvironment.Spec.WorkerName)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionScheduled,
		metav1.ConditionTrue,
		"Scheduled", "ProblemEnvironment is assigned to Worker",
	)
	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}