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
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/docker/docker/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet"
	"github.com/janog-netcon/netcon-problem-management-subsystem/controllers/nclet/drivers"
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

	heartbeatInterval    string
	statusUpdateInterval string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netconv1alpha1.AddToScheme(scheme))
}

func main() {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&sshAddr, "ssh-bind-address", ":2222", "The address SSH server binds to.")

	flag.StringVar(&externalIPAddr, "external-ip-address", "127.0.0.1", "The IP address user connect to.")
	flag.StringVar(&configDir, "config-directory", "/data", "Path ContainerLab files are placed")

	flag.StringVar(&heartbeatInterval, "heartbeat-interval", "3s", "Heartbeat interval")
	flag.StringVar(&statusUpdateInterval, "status-update-interval", "10s", "Status update interval")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
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

	heartbeatInterval, err := time.ParseDuration(heartbeatInterval)
	if err != nil {
		setupLog.Error(err, "failed to parse heartbeat interval")
	}

	statusUpdateInterval, err := time.ParseDuration(statusUpdateInterval)
	if err != nil {
		setupLog.Error(err, "failed to status update interval")
	}

	if err = (&controllers.ProblemEnvironmentReconciler{
		Client:                   mgr.GetClient(),
		Scheme:                   mgr.GetScheme(),
		WorkerName:               workerName,
		ProblemEnvironmentDriver: driver,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProblemEnvironment")
		os.Exit(1)
	}

	if err = mgr.Add(controllers.NewSSHServer(sshAddr)); err != nil {
		setupLog.Error(err, "unable to create ssh server")
		os.Exit(1)
	}

	if err = mgr.Add(controllers.NewHeartbeatAgent(
		workerName,
		externalIPAddr,
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
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
