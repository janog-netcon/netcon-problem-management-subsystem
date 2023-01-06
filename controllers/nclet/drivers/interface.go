package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ProblemEnvironmentStatus string

const (
	// StatusInit indicates ProblemEnvironment has not been deployed yet
	StatusInit ProblemEnvironmentStatus = "Init"

	// StatusInit indicates ProblemEnvironment was deployed successfully
	StatusDeployed ProblemEnvironmentStatus = "Deployed"

	// StatusInit indicates ProblemEnvironment was not deployed successfully
	StatusError ProblemEnvironmentStatus = "Error"
)

type ProblemEnvironmentDriver interface {
	// Check whether ProblemEnvironment is deployed or not and return ContainerStatus
	// []ContainerDetailStatus should be nil if ProblemEnvironment is not deployed successfully
	Check(ctx context.Context, reader client.Client, problemEnvironment netconv1alpha1.ProblemEnvironment) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerStatus)

	// Deploy ProblemEnvironment
	Deploy(ctx context.Context, reader client.Client, problemEnvironment netconv1alpha1.ProblemEnvironment) error

	// Destroy ProblemEnvironment
	Destroy(ctx context.Context, reader client.Client, problemEnvironment netconv1alpha1.ProblemEnvironment) error
}
