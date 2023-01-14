package controllers

import (
	"context"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	"go.uber.org/multierr"
	coordv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type WorkerController struct {
	client.Client

	workerMonitorPeriod time.Duration
}

var _ inject.Client = &WorkerController{}
var _ manager.Runnable = &WorkerController{}

func NewWorkerController(
	workerMonitorPeriod time.Duration,
) *WorkerController {
	return &WorkerController{
		workerMonitorPeriod: workerMonitorPeriod,
	}
}

// InjectClient implements inject.Client
func (wc *WorkerController) InjectClient(c client.Client) error {
	wc.Client = c
	return nil
}

// Start implements manager.Runnable
func (wc *WorkerController) Start(ctx context.Context) error {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(wc.workerMonitorPeriod)

	for {
		select {
		case <-ticker.C:
			if err := wc.monitorWorkerHealth(ctx); err != nil {
				log.Error(err, "failed to monitor some workers")
			}
		case <-ctx.Done():
			return nil
		}
		ctx.Done()

	}
}

func (wc *WorkerController) monitorWorkerHealth(ctx context.Context) error {
	workerList := netconv1alpha1.WorkerList{}

	if err := wc.List(ctx, &workerList); err != nil {
		return err
	}

	errList := []error{}
	for _, worker := range workerList.Items {
		lease := coordv1.Lease{}
		if err := wc.Get(ctx, types.NamespacedName{
			Namespace: "netcon",
			Name:      worker.Name,
		}, &lease); err != nil {
			errList = append(errList, err)
			continue
		}

		ready := false
		if lease.Spec.RenewTime != nil {
			renewTime := *lease.Spec.RenewTime

			leaseDurationSeconds := int32(5)
			if lease.Spec.LeaseDurationSeconds != nil {
				leaseDurationSeconds = *lease.Spec.LeaseDurationSeconds
			}

			expireTime := renewTime.Add(time.Duration(leaseDurationSeconds) * time.Second)

			if expireTime.After(time.Now()) {
				ready = true
			}
		}

		current := util.GetWorkerCondition(
			&worker,
			netconv1alpha1.WorkerConditionReady,
		)

		if current == metav1.ConditionTrue && !ready {
			util.SetWorkerCondition(
				&worker,
				netconv1alpha1.WorkerConditionReady,
				metav1.ConditionFalse,
				"HealthCheckFail",
				"failed to check health",
			)
		} else if current != metav1.ConditionTrue && ready {
			util.SetWorkerCondition(
				&worker,
				netconv1alpha1.WorkerConditionReady,
				metav1.ConditionTrue,
				"HealthCheck",
				"checked health",
			)
		} else {
			continue
		}

		if err := wc.Status().Update(ctx, &worker); err != nil {
			errList = append(errList, err)
		}
	}

	return multierr.Combine(errList...)
}
