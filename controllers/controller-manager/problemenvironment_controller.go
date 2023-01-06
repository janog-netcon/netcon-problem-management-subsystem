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
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
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

const MAX_USED_PERCENT float64 = 100.0
const DEFAULT_PASSWORD_LENGTH = 24

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problemenvironments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problems,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch

func generatePassword(length int) (string, error) {
	if length < 0 {
		return "", errors.New("invalid length")
	}

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	var buf = make([]rune, length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		buf[i] = chars[idx.Int64()]
	}

	return string(buf), nil
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

	if problemEnvironment.DeletionTimestamp != nil {
		log.Info("being deleted, ignoring")
		// there is nothing to do when deleting ProblemEnvironment
		return ctrl.Result{}, nil
	}

	scheduled := util.GetProblemEnvironmentCondition(
		&problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionScheduled,
	) == metav1.ConditionTrue

	if !scheduled {
		if problemEnvironment.Spec.WorkerName == "" {
			log.Info("not scheduled yet, scheduling")
			return r.schedule(ctx, &problemEnvironment)
		}

		return r.confirmSchedule(ctx, &problemEnvironment)
	}

	return r.checkStatus(ctx, &problemEnvironment)
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

func (r *ProblemEnvironmentReconciler) electWorker(
	ctx context.Context,
	workers netconv1alpha1.WorkerList,
	problemEnvironments netconv1alpha1.ProblemEnvironmentList,
) string {
	log := log.FromContext(ctx)

	workerLength := len(workers.Items)
	if workerLength == 0 {
		return ""
	} else if workerLength < 2 {
		return workers.Items[0].Name
	}

	type WorkerResource struct {
		Name string
		// CPUUsedPercent            float64
		// MemoryUsedPercent         float64
		SumOfResourcesUsedPercent float64
	}
	arr := make([]WorkerResource, 0, workerLength)
	for i := 0; i < workerLength; i++ {
		cpuUsedPct, err := strconv.ParseFloat(workers.Items[i].Status.WorkerInfo.CPUUsedPercent, 64)
		if err != nil {
			log.Error(err, "failed to parse CPUUsedPercent for worker election")
			cpuUsedPct = MAX_USED_PERCENT
		}
		memoryUsedPercent, err := strconv.ParseFloat(workers.Items[i].Status.WorkerInfo.MemoryUsedPercent, 64)
		if err != nil {
			log.Error(err, "failed to parse MemoryUsedPercent for worker election")
			memoryUsedPercent = MAX_USED_PERCENT
		}
		sumOfResourcesUsedPercent := cpuUsedPct + memoryUsedPercent

		arr = append(arr, WorkerResource{workers.Items[i].Name, sumOfResourcesUsedPercent})
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].SumOfResourcesUsedPercent < arr[j].SumOfResourcesUsedPercent
	})

	log.Info("electWorker : " + arr[0].Name)

	return arr[0].Name
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
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 5 * time.Second})
	}

	problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
	if err := r.List(ctx, &problemEnvironments); err != nil {
		message := "failed to list ProblemEnvironments"
		log.Error(err, message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
			metav1.ConditionFalse,
			"WorkersNotSchedulable",
			message,
		)
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 5 * time.Second})
	}

	electedWorkerName := r.electWorker(ctx, workers, problemEnvironments)

	if electedWorkerName != "" {
		problemEnvironment.Spec.WorkerName = electedWorkerName
		return r.update(ctx, problemEnvironment, ctrl.Result{})
	} else {
		reason := "ScheduleFailed"
		message := "failed to elect worker for scheduling"
		log.Info(message)
		util.SetProblemEnvironmentCondition(
			problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
			metav1.ConditionFalse,
			reason,
			message,
		)
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
	}
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
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 3 * time.Second})
	}

	password, err := generatePassword(DEFAULT_PASSWORD_LENGTH)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to confirm schedule: %w", err)
	}

	log.Info("confirmed", "workerName", problemEnvironment.Spec.WorkerName)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionScheduled,
		metav1.ConditionTrue,
		"Scheduled", "ProblemEnvironment is assigned to Worker",
	)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionDeployed,
		metav1.ConditionFalse,
		"NotDeployed", "ProblemEnvironment is not deployed to Worker",
	)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionFalse,
		"NotReady", "ProblemEnvironment is not ready",
	)
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionAssigned,
		metav1.ConditionFalse,
		"NotAssigned", "ProblemEnvironment is not assigned",
	)
	problemEnvironment.Status.Password = password
	return r.updateStatus(ctx, problemEnvironment, ctrl.Result{})
}

func (r *ProblemEnvironmentReconciler) markReady(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionTrue,
		"Ready", "ProblemEnvironment is ready",
	)
	return r.updateStatus(ctx, problemEnvironment, res)
}

func (r *ProblemEnvironmentReconciler) markNotReady(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionFalse,
		"NotReady", "ProblemEnvironment is not ready",
	)
	return r.updateStatus(ctx, problemEnvironment, res)
}

func (r *ProblemEnvironmentReconciler) checkStatus(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.V(1).Info("checking status")

	if problemEnvironment.Status.Containers == nil {
		return r.markNotReady(ctx, problemEnvironment, ctrl.Result{})
	}

	if len(problemEnvironment.Status.Containers) == 0 {
		return r.markNotReady(ctx, problemEnvironment, ctrl.Result{})
	}

	for _, containerStatus := range problemEnvironment.Status.Containers {
		if !containerStatus.Ready {
			return r.markNotReady(ctx, problemEnvironment, ctrl.Result{})
		}
	}

	return r.markReady(ctx, problemEnvironment, ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
