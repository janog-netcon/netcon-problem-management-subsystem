package controllers

import (
	"context"
	"fmt"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getProblemEnvironmentFor(problemEnvironment netconv1alpha1.ProblemEnvironment, worker netconv1alpha1.Worker) *ProblemEnvironment {
	return &ProblemEnvironment{
		Name:     problemEnvironment.Name,
		Host:     worker.Status.WorkerInfo.ExternalIPAddress,
		Port:     worker.Status.WorkerInfo.ExternalPort,
		User:     fmt.Sprintf("nc_%s", problemEnvironment.Name),
		Password: problemEnvironment.Status.Password,
	}
}

func (g *Gateway) GetProblemEnvironment(ctx context.Context, name string) (*ProblemEnvironment, error) {
	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := g.Get(ctx, types.NamespacedName{Namespace: "netcon", Name: name}, &problemEnvironment); err != nil {
		if errors.IsNotFound(err) {
			return nil, ErrProblemEnvironmentNotFound{name}
		}
		return nil, err
	}

	worker := netconv1alpha1.Worker{}
	if err := g.Get(ctx, types.NamespacedName{Name: problemEnvironment.Spec.WorkerName}, &worker); err != nil {
		if errors.IsNotFound(err) {
			return nil, ErrWorkerNotFound{problemEnvironment.Spec.WorkerName}
		}
		return nil, err
	}

	return getProblemEnvironmentFor(problemEnvironment, worker), nil
}

func (g *Gateway) AcquireProblemEnvironment(ctx context.Context, problemName string) (*ProblemEnvironment, error) {
	problem := netconv1alpha1.Problem{}
	if err := g.Get(ctx, types.NamespacedName{Namespace: "netcon", Name: problemName}, &problem); err != nil {
		if errors.IsNotFound(err) {
			return nil, ErrProblemNotFound{problemName}
		}
		return nil, err
	}

	problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
	if err := g.List(ctx, &problemEnvironments, client.MatchingLabels{
		"problemName": problemName,
	}); err != nil {
		return nil, err
	}

	var selected *netconv1alpha1.ProblemEnvironment
	for _, problemEnvironment := range problemEnvironments.Items {
		isAssigned := util.GetProblemEnvironmentCondition(&problemEnvironment, netconv1alpha1.ProblemEnvironmentConditionAssigned)
		isReady := util.GetProblemEnvironmentCondition(&problemEnvironment, netconv1alpha1.ProblemEnvironmentConditionReady)
		if isAssigned == metav1.ConditionFalse && isReady == metav1.ConditionTrue {
			selected = &problemEnvironment
			break
		}
	}
	if selected == nil {
		return nil, ErrNoAvailableProblemEnvironment{problemName}
	}

	util.SetProblemEnvironmentCondition(
		selected,
		netconv1alpha1.ProblemEnvironmentConditionAssigned,
		metav1.ConditionTrue,
		"Assigned",
		"Assigned ProblemEnvironment",
	)
	if err := g.Status().Update(ctx, selected); err != nil {
		return nil, err
	}

	g.Recorder.Event(
		selected,
		corev1.EventTypeNormal,
		netconv1alpha1.ProblemEnvironmentEventAssigned,
		"ProblemEnvironment assigned",
	)

	worker := netconv1alpha1.Worker{}
	if err := g.Get(ctx, types.NamespacedName{Name: selected.Spec.WorkerName}, &worker); err != nil {
		if errors.IsNotFound(err) {
			return nil, ErrWorkerNotFound{selected.Spec.WorkerName}
		}
		return nil, err
	}

	return getProblemEnvironmentFor(*selected, worker), nil
}

func (g *Gateway) ReleaseProblemEnvironment(ctx context.Context, name string) error {
	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := g.Get(ctx, types.NamespacedName{Namespace: "netcon", Name: name}, &problemEnvironment); err != nil {
		if errors.IsNotFound(err) {
			return ErrProblemEnvironmentNotFound{name}
		}
		return err
	}

	if err := g.Delete(ctx, &problemEnvironment); err != nil {
		return err
	}

	return nil
}
