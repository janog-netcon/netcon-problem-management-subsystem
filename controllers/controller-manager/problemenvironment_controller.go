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
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/crypto"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

// ProblemEnvironmentReconciler reconciles a ProblemEnvironment object
type ProblemEnvironmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	Parameters SchedulerParameters
}

type SchedulerParameters struct {
	CPUWeight       float64
	MemoryWeight    float64
	MemoryThreshold float64
	Temperature     float64
}

const MAX_USED_PERCENT float64 = 100.0
const DEFAULT_PASSWORD_LENGTH = 24

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

type CandidateWorker struct {
	// Name is the name of the worker.
	Name string

	// Score is the score of the worker (0..1).
	Score float64
}

func (r *ProblemEnvironmentReconciler) electWorker(
	ctx context.Context,
	workers netconv1alpha1.WorkerList,
	workerSelectors []metav1.LabelSelector,
) string {
	log := log.FromContext(ctx)

	workerLength := len(workers.Items)

	problemEnvironmentList := netconv1alpha1.ProblemEnvironmentList{}
	if err := r.List(ctx, &problemEnvironmentList); err != nil {
		log.Info("failed to ")
		return ""
	}

	candidates := make([]CandidateWorker, 0)
	for i := 0; i < workerLength; i++ {
		if util.GetWorkerCondition(
			&workers.Items[i],
			netconv1alpha1.WorkerConditionReady,
		) != metav1.ConditionTrue {
			continue
		}

		if workers.Items[i].Spec.DisableSchedule {
			continue
		}

		// check if worker matches workerSelectors
		if len(workerSelectors) > 0 {
			matched := false
			for _, selector := range workerSelectors {
				s, err := metav1.LabelSelectorAsSelector(&selector)
				if err != nil {
					log.Error(err, "failed to parse label selector")
					continue
				}
				if s.Matches(labels.Set(workers.Items[i].Labels)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		cpuUsedPercent, err := strconv.ParseFloat(workers.Items[i].Status.WorkerInfo.CPUUsedPercent, 64)
		if err != nil {
			log.Error(err, "failed to parse CPUUsedPercent for worker election")
			cpuUsedPercent = MAX_USED_PERCENT
		}

		memoryUsedPercent, err := strconv.ParseFloat(workers.Items[i].Status.WorkerInfo.MemoryUsedPercent, 64)
		if err != nil {
			log.Error(err, "failed to parse MemoryUsedPercent for worker election")
			memoryUsedPercent = MAX_USED_PERCENT
		}

		// If memory usage is above threshold, it's too danger to deploy to this worker.
		if memoryUsedPercent > r.Parameters.MemoryThreshold {
			continue
		}

		// Note: Score is scaled from 0 to 1
		cost := (cpuUsedPercent*r.Parameters.CPUWeight + memoryUsedPercent*r.Parameters.MemoryWeight)
		cost = cost / (r.Parameters.CPUWeight + r.Parameters.MemoryWeight) / 100
		score := 1 - cost

		candidates = append(candidates, CandidateWorker{
			Name:  workers.Items[i].Name,
			Score: score,
		})
	}
	if len(candidates) == 0 {
		return ""
	}

	return r.electWorkerFromCandidates(candidates)
}

// electWorkerFromCandidates elects a worker from candidates based on the score.
// It uses the Boltzmann distribution to select a worker.
func (r *ProblemEnvironmentReconciler) electWorkerFromCandidates(candidates []CandidateWorker) string {
	totalCandidates := len(candidates)

	tmps := make([]float64, totalCandidates)
	for i := range totalCandidates {
		tmps[i] = math.Exp(tmps[i] / r.Parameters.Temperature)
	}

	total := 0.0
	for i := range totalCandidates {
		total += tmps[i]
	}

	v := rand.Float64()
	for i := range totalCandidates {
		v -= tmps[i] / total
		if v < 0 {
			return candidates[i].Name
		}
	}

	return candidates[totalCandidates-1].Name
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
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 3 * time.Second})
	}

	electedWorkerName := r.electWorker(ctx, workers, problemEnvironment.Spec.WorkerSelectors)

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
		return r.updateStatus(ctx, problemEnvironment, ctrl.Result{RequeueAfter: 3 * time.Second})
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

	password, err := crypto.GeneratePassword(DEFAULT_PASSWORD_LENGTH)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to confirm schedule: %w", err)
	}

	r.Recorder.Eventf(
		problemEnvironment,
		corev1.EventTypeNormal,
		netconv1alpha1.ProblemEnvironmentEventScheduled,
		"ProblemEnvironment was scheduled on %s",
		problemEnvironment.Spec.WorkerName,
	)

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

func (r *ProblemEnvironmentReconciler) markNotReadyInit(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionFalse,
		"NotReady", "nclet haven't checked yet",
	)
	return r.updateStatus(ctx, problemEnvironment, res)
}

func (r *ProblemEnvironmentReconciler) markNotReady(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	// Suppress unneeded status updates
	if util.GetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
	) == metav1.ConditionFalse {
		return ctrl.Result{}, nil
	}

	r.Recorder.Event(
		problemEnvironment,
		corev1.EventTypeWarning,
		netconv1alpha1.ProblemEnvironmentEventNotReady,
		"Some containers went down",
	)

	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionFalse,
		"NotReady", "ProblemEnvironment is not ready",
	)
	return r.updateStatus(ctx, problemEnvironment, res)
}

func (r *ProblemEnvironmentReconciler) markReady(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	// Suppress unneeded status updates
	if util.GetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
	) == metav1.ConditionTrue {
		return ctrl.Result{}, nil
	}

	r.Recorder.Event(
		problemEnvironment,
		corev1.EventTypeNormal,
		netconv1alpha1.ProblemEnvironmentEventReady,
		"All containers are ready now",
	)

	util.SetProblemEnvironmentCondition(
		problemEnvironment,
		netconv1alpha1.ProblemEnvironmentConditionReady,
		metav1.ConditionTrue,
		"Ready", "ProblemEnvironment is ready",
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
		return r.markNotReadyInit(ctx, problemEnvironment, ctrl.Result{})
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
