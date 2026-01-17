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

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	controllers "github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet"
	"github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet/drivers"
	"github.com/janog-netcon/netcon-problem-management-subsystem/internal/tracing"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/crypto"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	metricsAddr string
	probeAddr   string
	sshAddr     string

	externalIPAddr string
	configDir      string

	adminPass string

	heartbeatInterval    string
	statusUpdateInterval string

	workerClass string

	maxWorkers int
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netconv1alpha1.AddToScheme(scheme))
}

func main() {
	ctx := ctrl.SetupSignalHandler()

	controllers.RegisterMetrics(metrics.Registry)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&sshAddr, "ssh-bind-address", ":2222", "The address SSH server binds to.")

	flag.StringVar(&externalIPAddr, "external-ip-address", "127.0.0.1", "The IP address user connect to.")
	flag.StringVar(&configDir, "config-directory", "/data", "Path ContainerLab files are placed")

	flag.StringVar(&adminPass, "admin-password", "", "The address SSH server binds to.")

	flag.StringVar(&workerClass, "worker-class", "", "Class of the Worker")

	flag.StringVar(&heartbeatInterval, "heartbeat-interval", "1s", "Heartbeat interval")
	flag.StringVar(&statusUpdateInterval, "status-update-interval", "10s", "Status update interval")

	flag.IntVar(&maxWorkers, "max-workers", 0, "Max workers for ProblemEnvironment")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if workerClass == "" {
		setupLog.Error(fmt.Errorf("worker-class is required"), "worker-class is required")
		os.Exit(1)
	}

	shutdown, err := tracing.SetupOpenTelemetry(ctx, "nclet")
	if err != nil {
		setupLog.Error(err, "failed to setup OpenTelemetry")
		os.Exit(1)
	}
	defer shutdown(ctx)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		// nclet run on each Worker, so LeaderElection isn't needed
		LeaderElection: false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		setupLog.Error(err, "failed to create docker client")
	}

	driver := drivers.NewContainerLabProblemEnvironmentDriver(configDir, dockerClient)

	workerName, err := os.Hostname()
	if err != nil {
		setupLog.Error(err, "failed to get hostname")
		os.Exit(1)
	}

	if adminPass == "" {
		password, err := crypto.GeneratePassword(64)
		if err != nil {
			setupLog.Error(err, "failed to generate password for admin")
		}
		adminPass = password
	}

	heartbeatInterval, err := time.ParseDuration(heartbeatInterval)
	if err != nil {
		setupLog.Error(err, "failed to parse heartbeat interval")
	}

	statusUpdateInterval, err := time.ParseDuration(statusUpdateInterval)
	if err != nil {
		setupLog.Error(err, "failed to status update interval")
	}

	idx := strings.LastIndex(sshAddr, ":")
	if idx == -1 {
		setupLog.Error(fmt.Errorf("invalid format"), "failed to parse sshAddr")
	}
	sshPort, err := strconv.ParseUint(sshAddr[idx+1:], 10, 16)
	if err != nil {
		setupLog.Error(err, "failed to parse sshAddr")
	}

	if maxWorkers == 0 {
		cores, err := cpu.Counts(true)
		if err != nil {
			setupLog.Error(err, "failed to get the number of cpu cores")
		}
		workers := cores / 8
		if workers != 0 {
			maxWorkers = workers
		}
	}

	if err = (&controllers.ProblemEnvironmentReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		Recorder:                 mgr.GetEventRecorderFor("problemenvironment-controller"),
		MaxConcurrentReconciles:  maxWorkers,
		WorkerName:               workerName,
		ProblemEnvironmentDriver: driver,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProblemEnvironment")
		os.Exit(1)
	}

	if err = mgr.Add(controllers.NewSSHServer(mgr.GetClient(), sshAddr, adminPass)); err != nil {
		setupLog.Error(err, "unable to create ssh server")
		os.Exit(1)
	}

	if err = mgr.Add(controllers.NewHeartbeatAgent(
		mgr.GetClient(),
		workerName,
		workerClass,
		externalIPAddr,
		uint16(sshPort),
		heartbeatInterval,
		statusUpdateInterval,
	)); err != nil {
		setupLog.Error(err, "unable to add heartbeat agent")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
