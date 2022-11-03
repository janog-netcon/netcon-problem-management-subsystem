package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ProblemEnvironmentStatus string

const (
	StatusReady    ProblemEnvironmentStatus = "Ready"
	StatusNotReady ProblemEnvironmentStatus = "NotReady"
)

type ProblemEnvironmentDriver interface {
	// Check checks whether problemEnvironment is working or not
	// Note that even if status returned from Check is *StatusReady*, it doesn't mean all containers are running successfully.
	// So, you need to check ContainerDetailStatus to ensure all containers are running.
	Check(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerDetailStatus)
	Deploy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
	Destroy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
}
