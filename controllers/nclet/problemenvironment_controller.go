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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet/drivers"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	"github.com/pkg/errors"
)

const ProblemEnvironmentFinalizer string = "problemenvironment.netcon.janog.gr.jp"
const StatusRefreshInterval = 5 * time.Second

// ProblemEnvironmentReconciler reconciles a ProblemEnvironment object
type ProblemEnvironmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	MaxConcurrentReconciles int

	// WorkerName is the name of worker where nclet places
	WorkerName string

	ProblemEnvironmentDriver drivers.ProblemEnvironmentDriver
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

	if problemEnvironment.Spec.WorkerName != r.WorkerName {
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

	deployed := util.GetProblemEnvironmentCondition(
		&problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionDeployed,
	) == metav1.ConditionTrue

	if !deployed {
		return r.deploy(ctx, &problemEnvironment)
	}

	return r.check(ctx, &problemEnvironment)
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
	if err := r.ProblemEnvironmentDriver.Destroy(ctx, r.Client, *problemEnvironment); err != nil {
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

func (r *ProblemEnvironmentReconciler) updateContainerStatus(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	containerStatuses []netconv1alpha1.ContainerStatus,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	needToUpdateStatus := false
	if !reflect.DeepEqual(containerStatuses, problemEnvironment.Status.Containers) {
		needToUpdateStatus = true
	}

	if needToUpdateStatus {
		log.V(1).Info("updating container statuses",
			"oldContainerStatuses", problemEnvironment.Status.Containers,
			"containerStatuses", containerStatuses,
		)
		problemEnvironment.Status.Containers = containerStatuses
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: StatusRefreshInterval})
	}

	return ctrl.Result{RequeueAfter: StatusRefreshInterval}, nil
}

func (r *ProblemEnvironmentReconciler) deploy(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	status, _ := r.ProblemEnvironmentDriver.Check(ctx, r.Client, *problemEnvironment)

	switch status {
	case drivers.StatusInit:
		r.Recorder.Eventf(
			problemEnvironment,
			corev1.EventTypeNormal,
			netconv1alpha1.ProblemEnvironmentEventDeploying,
			"Starting to deploy ProblemEnvironment on %s",
			r.WorkerName,
		)
		start := time.Now()
		r.ProblemEnvironmentDriver.Deploy(ctx, r.Client, *problemEnvironment)
		elapsed := time.Since(start)
		r.Recorder.Eventf(
			problemEnvironment,
			corev1.EventTypeNormal,
			netconv1alpha1.ProblemEnvironmentEventDeployed,
			"Deployed ProblemEnvironment in %s",
			r.WorkerName,
			elapsed.String(),
		)
		reason := "Deployed"
		message := "ProblemEnvironment is deployed"
		log.Info(message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionDeployed,
			metav1.ConditionTrue,
			reason,
			message,
		)
		return r.check(ctx, problemEnvironment)
	default: // StatusReady, StatusError
		err := errors.New("unexpected state")
		message := "internal error"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionDeployed,
			metav1.ConditionTrue,
			"DeployFailed",
			message,
		)
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
	}
}

func (r *ProblemEnvironmentReconciler) check(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	_, containerDetailStatuses := r.ProblemEnvironmentDriver.Check(ctx, r.Client, *problemEnvironment)

	return r.updateContainerStatus(
		ctx,
		problemEnvironment,
		containerDetailStatuses,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.MaxConcurrentReconciles == 0 {
		r.MaxConcurrentReconciles = 1
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.MaxConcurrentReconciles,
		}).
		Complete(r)
}
