package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type NoopProblemEnvironmentDriver struct{}

var _ ProblemEnvironmentDriver = &NoopProblemEnvironmentDriver{}

func NewNoopProblemEnvironmentDriver() *NoopProblemEnvironmentDriver {
	return &NoopProblemEnvironmentDriver{}
}

// Check implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Check(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerDetailStatus) {
	return StatusReady, nil
}

// Deploy implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}

// Destroy implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}
