package printers

import (
	"io"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

type PrintOptions printers.PrintOptions

func PrintObject(writer io.Writer, obj runtime.Object, options PrintOptions) error {
	generateOptions := GenerateOptions{
		Wide: options.Wide,
	}

	var table *metav1.Table
	var err error

	switch obj := obj.(type) {
	case *v1alpha1.Problem:
		panic(nil)
	case *v1alpha1.ProblemList:
		panic(nil)
	case *v1alpha1.ProblemEnvironment:
		table, err = generateTableForProblemEnvironment(obj, generateOptions)
	case *v1alpha1.ProblemEnvironmentList:
		table, err = generateTableForProblemEnvironmentList(obj, generateOptions)
	case *v1alpha1.Worker:
		panic(nil)
	case *v1alpha1.WorkerList:
		panic(nil)
	default:
		return ErrUnsupporedType
	}

	if err != nil {
		return err
	}

	printers.NewTablePrinter(printers.PrintOptions(options)).PrintObj(table, writer)
	return nil
}
