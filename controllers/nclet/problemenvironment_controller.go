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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

// TODO: fetch Worker name dynamically
const WorkerName string = "worker001"

// ProblemEnvironmentReconciler reconciles a ProblemEnvironment object
type ProblemEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	name          string
	finalizerName string
}

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

	// check whether ProblemEnvironment is being deleted or not
	if problemEnvironment.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&problemEnvironment, r.finalizerName) {
			// ProblemEnvironment is assigned to this Worker, cleaning up
			log.Info("ProblemEnvironment is being deleted, cleaning up instance")
			return r.cleanup(ctx, &problemEnvironment)
		}

		// ProblemEnvironment is assigned to other Worker, ignoring
		return ctrl.Result{}, nil
	}

	// now, we are reconciling for **alive** ProblemEnvironments
	// check whether ProblemEnvironment is already assigned to any Worker
	if problemEnvironment.Spec.WorkerName == "" {
		log.V(1).Info("ProblemEnvironment isn't assigned to any Worker yet")
		return ctrl.Result{}, nil
	}

	// now, we are reconciling for **scheduled** ProblemEnvironments
	if problemEnvironment.Spec.WorkerName != r.name {
		// in case that ProblemEnvironment is assigned to other Worker

		if controllerutil.ContainsFinalizer(&problemEnvironment, r.finalizerName) {
			log.Info(
				"reassigned ProblemEnvironment to other Worker",
				"newWorkerName",
				problemEnvironment.Spec.WorkerName,
			)
			util.SetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
				metav1.ConditionFalse,
				"Rescheduling", "",
			)

			r.cleanup(ctx, &problemEnvironment)

		}

		return ctrl.Result{}, nil
	}

	switch len(problemEnvironment.Finalizers) {
	case 0:
		log.Info("scheduled ProblemEnvironment to this Worker")
		return r.instantiate(ctx, &problemEnvironment)
	case 1:
		if !controllerutil.ContainsFinalizer(&problemEnvironment, r.finalizerName) {
			log.Info("waiting old Worker for cleaning up ProblemEnvironment")
			return ctrl.Result{}, nil
		}
		// TODO(proelbtn): watch instance
		return ctrl.Result{}, nil
	default:
		// unknown state, it may just bug
		err := fmt.Errorf("multiple finalizers are registered to ProblemEnvironment")
		log.Error(err, "failed to instantiate ProblemEnvironment")
		return ctrl.Result{}, err
	}
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

func (r *ProblemEnvironmentReconciler) updateBoth(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	if res, err := r.update(ctx, problemEnvironment, ctrl.Result{}); err != nil {
		return res, err
	}
	return r.updateStatus(ctx, problemEnvironment, res)
}

func (r *ProblemEnvironmentReconciler) cleanup(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(proelbtn): cleaning up instance

	controllerutil.RemoveFinalizer(problemEnvironment, r.finalizerName)
	return r.updateBoth(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) instantiate(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(proelbtn): instantiate

	controllerutil.AddFinalizer(problemEnvironment, r.finalizerName)
	return r.updateBoth(ctx, problemEnvironment, ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.name = WorkerName
	r.finalizerName = fmt.Sprintf("%s.problemenvironment.netcon.janog.gr.jp", WorkerName)

	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
