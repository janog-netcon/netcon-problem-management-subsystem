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
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet/drivers"
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

	driver drivers.ProblemEnvironmentDriver
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

	// once if deploy is failed, nclet stop to handle it
	if util.GetProblemEnvironmentCondition(
		&problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
	) == metav1.ConditionFalse {
		log.Info("skipping to deploy because deploy was failed")
		return ctrl.Result{}, nil
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
		log.Error(err, "failed to update")
		return ctrl.Result{}, err
	}
	return res, nil
}

func (r *ProblemEnvironmentReconciler) updateStatus(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if err := r.Status().Update(ctx, problemEnvironment); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}
	return res, nil
}

func (r *ProblemEnvironmentReconciler) cleanup(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("ProblemEnvironment is cleaning up")
	if err := r.driver.Destroy(ctx, r.Client, *problemEnvironment); err != nil {
		message := "failed to destroy ProblemEnvironment"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionReady,
			metav1.ConditionFalse,
			"DestroyFailed",
			message,
		)
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
	}

	controllerutil.RemoveFinalizer(problemEnvironment, ProblemEnvironmentFinalizer)
	return r.update(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) ensureInstance(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(proelbtn): instantiate
	status, containerStatuses, err := r.driver.Check(ctx, r.Client, *problemEnvironment)
	if err != nil {
		message := "failed to check ProblemEnvironment"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionReady,
			metav1.ConditionFalse,
			"CheckFailed",
			message,
		)
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
	}

	log.V(1).Info("checked the status of ProblemEnvironment", "status", status)

	switch status {
	case drivers.StatusUp:
		if !reflect.DeepEqual(containerStatuses, problemEnvironment.Status.Containers) {
			log.V(1).Info("updating container statuses",
				"oldContainerStatuses", problemEnvironment.Status.Containers,
				"containerStatuses", containerStatuses,
			)
			problemEnvironment.Status.Containers = containerStatuses
			return r.updateStatus(ctx, problemEnvironment, ctrl.Result{
				// RequeueAfter: 1 * time.Second,
			})
		}
		return ctrl.Result{
			// RequeueAfter: 3 * time.Second,
		}, nil
	case drivers.StatusDown:
		if err := r.driver.Deploy(ctx, r.Client, *problemEnvironment); err != nil {
			message := "failed to deploy ProblemEnvironment"
			log.Error(err, message)
			util.SetProblemEnvironmentCondition(
				problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
				metav1.ConditionFalse,
				"DeployFailed",
				message,
			)
			return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
		}
	}

	message := "ProblemEnvironment is instantiated"
	log.Info(message)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionTrue,
		"Instantiated",
		message,
	)

	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 3 * time.Second})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager, driver drivers.ProblemEnvironmentDriver) error {
	r.name = WorkerName
	r.driver = driver

	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
