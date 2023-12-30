package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

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

func NewHeartbeatAgent(client client.Client, workerName string, externalIPaddr string, externalPort uint16, heartbeatInterval time.Duration, statusUpdateInterval time.Duration) *HeartbeatAgent {
	return &HeartbeatAgent{
		Client:             client,
		workerName:         workerName,
		externalIPAddr:     externalIPaddr,
		externalPort:       externalPort,
		heartbeatTicker:    time.NewTicker(heartbeatInterval),
		statusUpdateTicker: time.NewTicker(statusUpdateInterval),
	}
}

// Controller-manager can run manager.Runnable
var _ manager.Runnable = &HeartbeatAgent{}

// Start implements manager.Runnable
func (a *HeartbeatAgent) Start(ctx context.Context) error {
	log := log.FromContext(ctx)

	if err := a.initMetricsCollector(ctx); err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	metricsCollectTicker := time.NewTicker(1 * time.Second)

	worker := netconv1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{
			Name: a.workerName,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, a, &worker, func() error {
		return nil
	}); err != nil {
		return err
	}

	lease := coordv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "netcon",
			Name:      a.workerName,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, a, &lease, func() error {
		holderIdentity := worker.Name
		renewTime := metav1.NowMicro()
		leaseDurationSecond := int32(5)

		lease.Spec.HolderIdentity = &holderIdentity
		lease.Spec.RenewTime = &renewTime
		lease.Spec.LeaseDurationSeconds = &leaseDurationSecond

		return nil
	}); err != nil {
		return err
	}

	for {
		select {
		case <-metricsCollectTicker.C:
			log.V(1).Info("metrics collector ticker is fired, collect metrics")
			if err := a.collectMetrics(ctx); err != nil {
				log.Error(err, "failed to collect metrics")
				continue
			}
		case <-a.heartbeatTicker.C:
			log.V(1).Info("heartbeat ticker is fired, create Lease for heartbeat")

			if _, err := ctrl.CreateOrUpdate(ctx, a, &lease, func() error {
				holderIdentity := worker.Name
				renewTime := metav1.NowMicro()
				leaseDurationSecond := int32(5)

				lease.Spec.HolderIdentity = &holderIdentity
				lease.Spec.RenewTime = &renewTime
				lease.Spec.LeaseDurationSeconds = &leaseDurationSecond

				return nil
			}); err != nil {
				log.Error(err, "failed to update lease")
				continue
			}
		case <-a.statusUpdateTicker.C:
			log.V(1).Info("statusUpdate ticker is fired, create Worker or update status of Worker")

			worker := netconv1alpha1.Worker{}

			if err := a.Get(ctx, types.NamespacedName{
				Name: a.workerName,
			}, &worker); err != nil {
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
