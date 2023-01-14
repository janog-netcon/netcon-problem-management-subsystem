package controllers

import (
	"context"
	"time"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type ProblemMetricsExporter struct {
	client.Client

	scrapeInterval time.Duration
}

var (
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

var _ inject.Client = &ProblemMetricsExporter{}
var _ manager.Runnable = &ProblemMetricsExporter{}

func NewProblemMetricsExporter(
	scrapeInterval time.Duration,
) *ProblemMetricsExporter {
	return &ProblemMetricsExporter{
		scrapeInterval: scrapeInterval,
	}
}

//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=netcon.janog.gr.jp,resources=workers/status,verbs=get

// InjectClient implements inject.Client
func (me *ProblemMetricsExporter) InjectClient(c client.Client) error {
	me.Client = c
	return nil
}

// Start implements manager.Runnable
func (me *ProblemMetricsExporter) Start(ctx context.Context) error {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(me.scrapeInterval)

	for _, collector := range []prometheus.Collector{
		problemReplicasGaugeVec,
		problemDesiredAssignableReplicasGaugeVec,
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

			if err := me.export(ctx, &problemList, &problemEnvironmentList); err != nil {
				log.Error(err, "failed to export metrics for workers")
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (wc *ProblemMetricsExporter) export(
	ctx context.Context,
	problemList *netconv1alpha1.ProblemList,
	problemEnvironmentList *netconv1alpha1.ProblemEnvironmentList,
) error {

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

	return nil
}
