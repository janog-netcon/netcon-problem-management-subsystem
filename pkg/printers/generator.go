package printers

import (
	"errors"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ref: https://pkg.go.dev/k8s.io/kubernetes/pkg/printers

var ErrUnsupporedType = errors.New("unsupported type")

type GenerateOptions struct {
	Wide bool
}

func GenerateTable(obj runtime.Object, options GenerateOptions) (*metav1.Table, error) {
	switch obj := obj.(type) {
	case *v1alpha1.Problem:
		panic(nil)
	case *v1alpha1.ProblemList:
		panic(nil)
	case *v1alpha1.ProblemEnvironment:
		return generateTableForProblemEnvironment(obj, options)
	case *v1alpha1.ProblemEnvironmentList:
		return generateTableForProblemEnvironmentList(obj, options)
	case *v1alpha1.Worker:
		panic(nil)
	case *v1alpha1.WorkerList:
		panic(nil)
	default:
		return nil, ErrUnsupporedType
	}
}
