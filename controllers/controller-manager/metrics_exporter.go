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

type MetricsExporter struct {
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
	workerScheduledProblemEnvironmentsGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcon_worker_scheduled_problem_environments",
			Help: "Number of ProblemEnvironment scheduled to the worker",
		},
		[]string{"name"},
	)
	schedulableWorkersGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "netcon_schedulable_workers",
			Help: "Number of ready workers",
		},
	)
	problemReplicasGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcon_problem_replicas",
			Help: "",
		},
		[]string{"namespace", "name", "status"},
	)
	problemDesiredAssignableReplicasGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcon_problem_desired_assignable_replicas",
			Help: "assignable replicas",
		},
		[]string{"namespace", "name"},
	)
)

var _ inject.Client = &MetricsExporter{}
var _ manager.Runnable = &MetricsExporter{}

func NewMetricsExporter(
	scrapeInterval time.Duration,
) *MetricsExporter {
	return &MetricsExporter{
		scrapeInterval: scrapeInterval,
	}
}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers/status,verbs=get

// InjectClient implements inject.Client
func (me *MetricsExporter) InjectClient(c client.Client) error {
	me.Client = c
	return nil
}

// Start implements manager.Runnable
func (me *MetricsExporter) Start(ctx context.Context) error {
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
			problemList := netconv1alpha1.ProblemList{}
			if err := me.List(ctx, &problemList); err != nil {
				log.Error(err, "failed to list workers")
			}

			problemEnvironmentList := netconv1alpha1.ProblemEnvironmentList{}
			if err := me.List(ctx, &problemEnvironmentList); err != nil {
				log.Error(err, "failed to list workers")
			}

			workerList := netconv1alpha1.WorkerList{}
			if err := me.List(ctx, &workerList); err != nil {
				log.Error(err, "failed to list workers")
			}

			if err := me.export(
				ctx,
				&problemList,
				&problemEnvironmentList,
				&workerList,
			); err != nil {
				log.Error(err, "failed to export metrics for workers")
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (wc *MetricsExporter) export(
	ctx context.Context,
	problemList *netconv1alpha1.ProblemList,
	problemEnvironmentList *netconv1alpha1.ProblemEnvironmentList,
	workerList *netconv1alpha1.WorkerList,
) error {
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

	for _, problem := range problemList.Items {
		problemReplicasGaugeVec.
			With(prometheus.Labels{
				"namespace": problem.Namespace,
				"name":      problem.Name,
				"status":    "scheduled",
			}).
			Set(float64(problem.Status.Replicas.Scheduled))

		problemReplicasGaugeVec.
			With(prometheus.Labels{
				"namespace": problem.Namespace,
				"name":      problem.Name,
				"status":    "assignable",
			}).
			Set(float64(problem.Status.Replicas.Assignable))

		problemReplicasGaugeVec.
			With(prometheus.Labels{
				"namespace": problem.Namespace,
				"name":      problem.Name,
				"status":    "assigned",
			}).
			Set(float64(problem.Status.Replicas.Assigned))

		problemDesiredAssignableReplicasGaugeVec.
			With(prometheus.Labels{
				"namespace": problem.Namespace,
				"name":      problem.Name,
			}).
			Set(float64(problem.Spec.AssignableReplicas))
	}

	workerCounterMap := map[string]int{}
	for _, problemEnvironment := range problemEnvironmentList.Items {
		workerName := ""
		if util.GetProblemEnvironmentCondition(
			&problemEnvironment,
			netconv1alpha1.ProblemEnvironmentConditionScheduled,
		) == metav1.ConditionTrue {
			workerName = problemEnvironment.Spec.WorkerName
		}

		if _, ok := workerCounterMap[workerName]; !ok {
			workerCounterMap[workerName] = 1
			continue
		}

		workerCounterMap[workerName] += 1
	}

	for key, value := range workerCounterMap {
		workerScheduledProblemEnvironmentsGaugeVec.
			With(prometheus.Labels{
				"name": key,
			}).
			Set(float64(value))
	}

	return nil
}
