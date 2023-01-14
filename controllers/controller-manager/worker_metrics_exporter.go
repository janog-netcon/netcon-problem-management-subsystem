package controllers

import (
	"context"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type WorkerMetricsExporter struct {
	client.Client

	scrapeInterval time.Duration
}

var (
	workersGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "netcon_workers",
			Help: "Number of workers",
		},
	)
	readyWorkersGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "netcon_ready_workers",
			Help: "Number of ready workers",
		},
	)
	schedulableWorkersGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "netcon_schedulable_workers",
			Help: "Number of ready workers",
		},
	)
)

var _ inject.Client = &WorkerMetricsExporter{}
var _ manager.Runnable = &WorkerMetricsExporter{}

func NewWorkerMetricsExporter(
	scrapeInterval time.Duration,
) *WorkerMetricsExporter {
	return &WorkerMetricsExporter{
		scrapeInterval: scrapeInterval,
	}
}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers/status,verbs=get

// InjectClient implements inject.Client
func (me *WorkerMetricsExporter) InjectClient(c client.Client) error {
	me.Client = c
	return nil
}

// Start implements manager.Runnable
func (me *WorkerMetricsExporter) Start(ctx context.Context) error {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(me.scrapeInterval)

	for _, collector := range []prometheus.Collector{
		workersGauge,
		readyWorkersGauge,
		schedulableWorkersGauge,
	} {
		if err := metrics.Registry.Register(collector); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ticker.C:
			workerList := netconv1alpha1.WorkerList{}
			if err := me.List(ctx, &workerList); err != nil {
				log.Error(err, "failed to list workers")
			}

			if err := me.export(ctx, &workerList); err != nil {
				log.Error(err, "failed to export metrics for workers")
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (wc *WorkerMetricsExporter) export(ctx context.Context, workerList *netconv1alpha1.WorkerList) error {
	total, ready, schedulable := 0.0, 0.0, 0.0

	for _, worker := range workerList.Items {
		total += 1

		if util.GetWorkerCondition(
			&worker,
			netconv1alpha1.WorkerConditionReady,
		) == metav1.ConditionTrue {
			ready += 1
		}

		if !worker.Spec.DisableSchedule {
			schedulable += 1
		}
	}

	workersGauge.Set(total)
	readyWorkersGauge.Set(ready)
	schedulableWorkersGauge.Set(schedulable)

	return nil
}
