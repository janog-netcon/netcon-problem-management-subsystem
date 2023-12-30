package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type MetricsExporter struct {
	client.Client

	Log logr.Logger

	ScrapeInterval time.Duration
}

const namespace = "netcon_pms"

// Workers-related metrics
var (
	workersLabels = []string{"namespace", "name"}

	workersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers_total",
		},
	)
	workersReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers_ready",
		},
		workersLabels,
	)
	workersSchedulable = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers_schedulable",
		},
		workersLabels,
	)
	workersScheduledProblemEnvironmentsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers_scheduled_problem_environments_total",
		},
		workersLabels,
	)
)

// Problems-related metrics
var (
	problemsMetricsLabels = []string{"namespace", "name"}

	problemsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "problems_total",
		},
	)
	problemsAssignableReplicasTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "problems_assignable_replicas_total",
		},
		problemsMetricsLabels,
	)
)

// ProblemEnvironments-related metrics
var (
	problemEnvironmentsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "problem_environments_total",
		},
		problemsMetricsLabels,
	)
	problemEnvironmentsReadyTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "problem_environments_ready_total",
		},
		problemsMetricsLabels,
	)
	problemEnvironmentsAssignedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "problem_environments_assigned_total",
		},
		problemsMetricsLabels,
	)
)

var _ manager.Runnable = &MetricsExporter{}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers/status,verbs=get

// Start implements manager.Runnable
func (me *MetricsExporter) Start(ctx context.Context) error {
	for _, collector := range []prometheus.Collector{
		workersTotal,
		workersReady,
		workersSchedulable,
		workersScheduledProblemEnvironmentsTotal,
		problemsTotal,
		problemsAssignableReplicasTotal,
		problemEnvironmentsTotal,
		problemEnvironmentsReadyTotal,
		problemEnvironmentsAssignedTotal,
	} {
		if err := metrics.Registry.Register(collector); err != nil {
			return err
		}
	}

	ticker := time.NewTicker(me.ScrapeInterval)
	for {
		select {
		case <-ticker.C:
			me.Log.V(1).Info("ticker is expired, starting to collect metrics")
			if err := me.collect(ctx); err != nil {
				me.Log.Error(err, "failed to collect metrics")
			}
			me.Log.V(1).Info("collected metrics")
		case <-ctx.Done():
			return nil
		}
	}
}

func (me *MetricsExporter) collect(ctx context.Context) error {
	workers := netconv1alpha1.WorkerList{}
	if err := me.List(ctx, &workers); err != nil {
		return err
	}

	problems := netconv1alpha1.ProblemList{}
	if err := me.List(ctx, &problems); err != nil {
		return err
	}

	problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
	if err := me.List(ctx, &problemEnvironments); err != nil {
		return err
	}

	// Workers-related metrics
	workersTotal.Set(float64(len(workers.Items)))

	workersReady.Reset()
	workersSchedulable.Reset()
	for _, worker := range workers.Items {
		labels := []string{worker.Namespace, worker.Name}

		total := 0
		for _, problemEnvironment := range problemEnvironments.Items {
			if problemEnvironment.Spec.WorkerName == worker.Name {
				total += 1
			}
		}

		schedulableValue := 0
		if !worker.Spec.DisableSchedule {
			schedulableValue = 1
		}

		readyValue := 0
		if util.GetWorkerCondition(
			&worker,
			netconv1alpha1.WorkerConditionReady,
		) == metav1.ConditionTrue {
			readyValue = 1
		}

		workersReady.WithLabelValues(labels...).Set(float64(readyValue))
		workersSchedulable.WithLabelValues(labels...).Set(float64(schedulableValue))
		workersScheduledProblemEnvironmentsTotal.WithLabelValues(labels...).Set(float64(total))
	}

	// Problems-related metrics
	problemsTotal.Set(float64(len(problems.Items)))
	for _, problem := range problems.Items {
		labels := []string{problem.Namespace, problem.Name}
		problemsAssignableReplicasTotal.WithLabelValues(labels...).Set(float64(problem.Spec.AssignableReplicas))
	}

	// ProblemEnvironments-related metrics
	for _, problem := range problems.Items {
		labels := []string{problem.Namespace, problem.Name}

		total, readyTotal, assignedTotal := 0, 0, 0
		for _, problemEnvironment := range problemEnvironments.Items {
			if len(problemEnvironment.OwnerReferences) == 0 {
				continue
			}
			if problemEnvironment.OwnerReferences[0].Name != problem.Name {
				continue
			}

			total += 1

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionReady,
			) == metav1.ConditionTrue {
				readyTotal += 1
			}

			if util.GetProblemEnvironmentCondition(
				&problemEnvironment,
				netconv1alpha1.ProblemEnvironmentConditionAssigned,
			) == metav1.ConditionTrue {
				assignedTotal += 1
			}
		}

		problemEnvironmentsTotal.WithLabelValues(labels...).Set(float64(total))
		problemEnvironmentsReadyTotal.WithLabelValues(labels...).Set(float64(readyTotal))
		problemEnvironmentsAssignedTotal.WithLabelValues(labels...).Set(float64(assignedTotal))
	}

	return nil
}
