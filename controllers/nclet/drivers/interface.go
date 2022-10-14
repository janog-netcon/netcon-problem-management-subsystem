package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ProblemEnvironmentStatus string

const (
	StatusUnknown ProblemEnvironmentStatus = "Unknown"
	StatusUp      ProblemEnvironmentStatus = "Up"
	StatusDown    ProblemEnvironmentStatus = "Down"
)

type ContainerStatus struct {
	Name                string
	Image               string
	ContainerID         string
	Ready               bool
	ManagementIPAddress string
}

type ProblemEnvironmentDriver interface {
	Check(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) (ProblemEnvironmentStatus, []ContainerStatus, error)
	Deploy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
	Destroy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
}
