package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ContainerLabProblemEnvironmentDriver struct{}

var _ ProblemEnvironmentDriver = &ContainerLabProblemEnvironmentDriver{}

func NewContainerLabProblemEnvironmentDriver() *ContainerLabProblemEnvironmentDriver {
	return &ContainerLabProblemEnvironmentDriver{}
}

// Check implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Check(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, error) {
	return StatusUp, nil
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}

// Destroy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	return nil
}
