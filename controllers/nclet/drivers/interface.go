package drivers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ProblemEnvironmentDriver interface {
	Check(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) (bool, error)
	Deploy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
	Destroy(ctx context.Context, reader client.Reader, problemEnvironment netconv1alpha1.ProblemEnvironment) error
}
