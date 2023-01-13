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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	util "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

const (
	KeyProblemName = "problemName"
)

// ProblemReconciler reconciles a Problem object
type ProblemReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problems,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=problems/status,verbs=get;update;patch

func (r *ProblemReconciler) listChildProblemEnvironments(
	ctx context.Context,
	problem *netconv1alpha1.Problem,
) (netconv1alpha1.ProblemEnvironmentList, error) {
	problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}

	return problemEnvironments, r.List(
		ctx,
		&problemEnvironments,
		client.InNamespace(problem.Namespace),
		client.MatchingLabels{KeyProblemName: problem.Name},
	)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProblemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	problem := netconv1alpha1.Problem{}
	if err := r.Get(ctx, req.NamespacedName, &problem); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "could not get Problem")
		}
		return ctrl.Result{}, err
	}

	if !problem.DeletionTimestamp.IsZero() {
		return r.updateStatus(ctx, &problem)
	}

	problemEnvironments, err := r.listChildProblemEnvironments(ctx, &problem)
	if err != nil {
		log.Error(err, "failed to list child ProblemEnvironments")
		return ctrl.Result{}, nil
	}

	assignableProblemEnvironments := 0
	for _, problemEnvironment := range problemEnvironments.Items {
		// problemEnvironment being deleted is not assignable
		if problemEnvironment.DeletionTimestamp != nil {
			continue
		}

		// problemEnvironment already assigned is not assignable
		if util.GetProblemEnvironmentCondition(
			&problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionAssigned,
		) == metav1.ConditionTrue {
			continue
		}

		// otherwise problemEnvironment is assignable
		assignableProblemEnvironments = assignableProblemEnvironments + 1
	}

	if problem.Spec.AssignableReplicas > assignableProblemEnvironments {
		newProbEnv := netconv1alpha1.ProblemEnvironment{}

		template := *problem.Spec.Template.DeepCopy()

		newProbEnv.Labels = template.Labels
		newProbEnv.Annotations = template.Annotations
		newProbEnv.Namespace = problem.Namespace
		newProbEnv.GenerateName = problem.Name + "-"
		newProbEnv.Spec = template.Spec

		if newProbEnv.Labels == nil {
			newProbEnv.Labels = make(map[string]string)
		}
		newProbEnv.Labels[KeyProblemName] = problem.Name

		if err := controllerutil.SetControllerReference(&problem, &newProbEnv, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &newProbEnv); err != nil {
			log.Error(err, "could not create new ProblemEnvironment")
			return ctrl.Result{}, err
		}

		log.Info("created ProblemEnvironment")
	} else if problem.Spec.AssignableReplicas < assignableProblemEnvironments {
		diff := assignableProblemEnvironments - problem.Spec.AssignableReplicas
		delete_count := 0
		for _, pe := range problemEnvironments.Items {
			if delete_count >= diff {
				break
			}
			condition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionAssigned)
			if condition != metav1.ConditionTrue {
				if err := r.Delete(ctx, &pe); err != nil {
					return ctrl.Result{}, err
				}
				delete_count += 1
				log.Info("deleted ProblemEnvironment")
			}
		}
	}

	return r.updateStatus(ctx, &problem)
}

func (r *ProblemReconciler) updateStatus(ctx context.Context, problem *netconv1alpha1.Problem) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	problemEnvironments, err := r.listChildProblemEnvironments(ctx, problem)
	if err != nil {
		log.Error(err, "failed to list child ProblemEnvironments")
		return ctrl.Result{}, nil
	}

	scheduled, assignable, assigned := 0, 0, 0
	for _, pe := range problemEnvironments.Items {
		isScheduled := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionScheduled)
		isReady := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionReady)
		isAssigned := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionAssigned)

		if isScheduled == metav1.ConditionTrue {
			scheduled++
		}

		if isReady == metav1.ConditionTrue {
			if isAssigned != metav1.ConditionTrue {
				assignable++
			} else {
				assigned++
			}
		}
	}

	problem.Status.Replicas.Total = len(problemEnvironments.Items)
	problem.Status.Replicas.Scheduled = scheduled
	problem.Status.Replicas.Assignable = assignable
	problem.Status.Replicas.Assigned = assigned

	return ctrl.Result{}, r.Status().Update(ctx, problem)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProblemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netconv1alpha1.Problem{}).
		Owns(&netconv1alpha1.ProblemEnvironment{}).
		Complete(r)
}
