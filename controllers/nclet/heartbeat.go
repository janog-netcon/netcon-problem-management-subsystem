package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	cpu "github.com/shirou/gopsutil/v3/cpu"
	mem "github.com/shirou/gopsutil/v3/mem"
)

const (
	CPU_USED_HISTORY_SIZE = 20
	MEM_USED_HISTORY_SIZE = 20
)

type HeartbeatAgent struct {
	client.Client

	// workerName is the name of Worker that nclet runs on
	workerName string

	// workerName is the name of Worker that nclet runs on
	externalIPAddr string
	externalPort   uint16

	heartbeatTicker    *time.Ticker
	statusUpdateTicker *time.Ticker

	cpuUsedHistory [CPU_USED_HISTORY_SIZE]float64
	memUsedHistory [MEM_USED_HISTORY_SIZE]float64
}

func NewHeartbeatAgent(workerName string, externalIPaddr string, externalPort uint16, heartbeatInterval time.Duration, statusUpdateInterval time.Duration) *HeartbeatAgent {
	return &HeartbeatAgent{
		workerName:         workerName,
		externalIPAddr:     externalIPaddr,
		externalPort:       externalPort,
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

	if err := a.initMetricsCollector(ctx); err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	metricsCollectTicker := time.NewTicker(1 * time.Second)

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
		case <-metricsCollectTicker.C:
			log.V(1).Info("metrics collector ticker is fired, collect metrics")
			if err := a.collectMetrics(ctx); err != nil {
				return fmt.Errorf("failed to collect metrics: %w", err)
			}
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

			cpuUsed, memUsed := a.getMetrics()

			worker.Status.WorkerInfo = netconv1alpha1.WorkerInfo{
				Hostname:          hostname,
				ExternalIPAddress: a.externalIPAddr,
				ExternalPort:      a.externalPort,
				CPUUsedPercent:    strconv.FormatFloat(cpuUsed, 'f', -1, 64),
				MemoryUsedPercent: strconv.FormatFloat(memUsed, 'f', -1, 64),
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

func (a *HeartbeatAgent) initMetricsCollector(ctx context.Context) error {
	if _, err := cpu.PercentWithContext(ctx, 0, false); err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	for i := range a.cpuUsedHistory {
		a.cpuUsedHistory[i] = 0
	}

	for i := range a.memUsedHistory {
		a.memUsedHistory[i] = 0
	}

	return nil
}

func (a *HeartbeatAgent) collectMetrics(ctx context.Context) error {
	log := log.FromContext(ctx)

	cpuUsed, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return err
	}

	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < CPU_USED_HISTORY_SIZE-1; i++ {
		a.cpuUsedHistory[i+1] = a.cpuUsedHistory[i]
	}

	for i := 0; i < MEM_USED_HISTORY_SIZE-1; i++ {
		a.memUsedHistory[i+1] = a.memUsedHistory[i]
	}

	a.cpuUsedHistory[0] = cpuUsed[0]
	a.memUsedHistory[0] = memInfo.UsedPercent

	log.V(1).Info("collected metrics", "cpuUsed", cpuUsed[0], "memUsed", memInfo.UsedPercent)

	return nil
}

func (a *HeartbeatAgent) getMetrics() (float64, float64) {
	var cpuUsed, memUsed float64

	// most old metrics != 0 => we can collect enough metrics
	if a.cpuUsedHistory[MEM_USED_HISTORY_SIZE-1] != 0 {
		for i := 0; i < MEM_USED_HISTORY_SIZE; i++ {
			cpuUsed += a.cpuUsedHistory[i]
		}
		cpuUsed /= MEM_USED_HISTORY_SIZE
	}

	// most old metrics != 0 => we can collect enough metrics
	if a.memUsedHistory[MEM_USED_HISTORY_SIZE-1] != 0 {
		for i := 0; i < MEM_USED_HISTORY_SIZE; i++ {
			memUsed += a.memUsedHistory[i]
		}
		memUsed /= MEM_USED_HISTORY_SIZE
	}

	return cpuUsed, memUsed
}
