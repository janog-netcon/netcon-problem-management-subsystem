package controllers

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type HeartbeatAgent struct {
	client.Client

	// workerName is the name of Worker that nclet runs on
	workerName string

	heartbeatTicker    *time.Ticker
	statusUpdateTicker *time.Ticker
}

func NewHeartbeatAgent(workerName string, heartbeatInterval time.Duration, statusUpdateInterval time.Duration) *HeartbeatAgent {
	return &HeartbeatAgent{
		workerName:         workerName,
		heartbeatTicker:    time.NewTicker(heartbeatInterval),
		statusUpdateTicker: time.NewTicker(statusUpdateInterval),
	}
}

// Controller-manager will inject Kubernetes API client to inject.Client
var _ inject.Client = &HeartbeatAgent{}

// Controller-manager can run manager.Runnable
var _ manager.Runnable = &HeartbeatAgent{}

// InjectClient implements inject.Client
func (a *HeartbeatAgent) InjectClient(c client.Client) error {
	a.Client = c
	return nil
}

// Start implements manager.Runnable
func (a *HeartbeatAgent) Start(ctx context.Context) error {
	// TODO: implement the mechanism for heartbeat
	// There are two forms of heartbeat for Node
	// 1. Update status field (DONE)
	// 2. Heartbeat with Lease
	// Worker also need to implement such heartbeat mechanism
	// ref: https://kubernetes.io/docs/concepts/architecture/nodes/#heartbeats

	log := log.FromContext(ctx)

	target := types.NamespacedName{
		// TODO: fix hardcode
		Namespace: "netcon",
		Name:      a.workerName,
	}

	worker := netconv1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: target.Namespace,
			Name:      target.Name,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, a, &worker, func() error {
		// Without following code block, CreateOrUpdate will fail
		worker.Namespace = target.Namespace
		worker.Name = target.Name
		return nil
	}); err != nil {
		return err
	}

	for {
		select {
		case <-a.heartbeatTicker.C:
			log.V(1).Info("heartbeat ticker is fired, create Lease for heartbeat")
			// TODO: implement the mechanism to update Lease for this Worker
		case <-a.statusUpdateTicker.C:
			log.V(1).Info("statusUpdate ticker is fired, create Worker or update status of Worker")

			worker := netconv1alpha1.Worker{}

			if err := a.Get(ctx, target, &worker); err != nil {
				log.Error(err, "failed to get Worker")
				continue
			}

			hostname, err := os.Hostname()
			if err != nil {
				log.Error(err, "failed to get hostname")
				continue
			}

			// TODO: resolve external IP address used by users to access
			externalIPAddress := "..."

			worker.Status.WorkerInfo = netconv1alpha1.WorkerInfo{
				Hostname:          hostname,
				ExternalIPAddress: externalIPAddress,
			}

			if err := a.Status().Update(ctx, &worker); err != nil {
				log.Error(err, "failed to update status")
			}
		case <-ctx.Done():
			log.Info("context done, quitting...")
			return nil
		}
	}
}
