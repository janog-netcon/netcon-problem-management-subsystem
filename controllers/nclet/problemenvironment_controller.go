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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

// TODO: fetch Worker name dynamically
const WorkerName string = "worker001"

const ProblemEnvironmentFinalizer string = "problemenvironment.netcon.janog.gr.jp"

// ProblemEnvironmentReconciler reconciles a ProblemEnvironment object
type ProblemEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	name string
}

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

	if scheduled := util.GetProblemEnvironmentCondition(
		&problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionScheduled,
	); scheduled != metav1.ConditionTrue {
		log.V(1).Info("ProblemEnvironment isn't scheduled yet")
		return ctrl.Result{}, nil
	}

	if problemEnvironment.Spec.WorkerName != r.name {
		log.V(1).Info("ProblemEnvironment isn't assigned to me")
		return ctrl.Result{}, nil
	}

	// check whether ProblemEnvironment is being deleted or not
	if problemEnvironment.DeletionTimestamp != nil {
		if !controllerutil.ContainsFinalizer(&problemEnvironment, ProblemEnvironmentFinalizer) {
			log.Info("being deleted, but not instantiated, ignoring")
			return ctrl.Result{}, nil
		}

		log.Info("being deleted, cleaning up instance")
		return r.cleanup(ctx, &problemEnvironment)
	}

	if !controllerutil.ContainsFinalizer(&problemEnvironment, ProblemEnvironmentFinalizer) {
		controllerutil.AddFinalizer(&problemEnvironment, ProblemEnvironmentFinalizer)
		return r.update(ctx, &problemEnvironment, ctrl.Result{})
	}

	log.Info("ensuring instance")
	return r.ensureInstance(ctx, &problemEnvironment)
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

func (r *ProblemEnvironmentReconciler) cleanup(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(proelbtn): cleaning up instance
	log.Info("ProblemEnvironment is cleaning up")

	controllerutil.RemoveFinalizer(problemEnvironment, ProblemEnvironmentFinalizer)
	return r.update(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) ensureInstance(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(proelbtn): instantiate
	message := "ProblemEnvironment is instantiated"
	log.Info(message)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionTrue,
		"Instantiated",
		message,
	)

	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.name = WorkerName

	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
