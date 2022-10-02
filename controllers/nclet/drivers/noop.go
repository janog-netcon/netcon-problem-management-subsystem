package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type NoopProblemEnvironmentDriver struct{}

var _ ProblemEnvironmentDriver = &NoopProblemEnvironmentDriver{}

// Check implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Check(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (bool, error) {
	return true, nil
}

// Deploy implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}

// Destroy implements ProblemEnvironmentDriver
func (*NoopProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}