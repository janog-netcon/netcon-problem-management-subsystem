package cmd

import (
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/clientset/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func newConfigMapClientForConfig(c *rest.Config, namespace string) (v1.ConfigMapInterface, error) {
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1().ConfigMaps(namespace), nil
}

func newProblemEnvironmentClientForConfig(c *rest.Config, namespace string) (v1alpha1.ProblemEnvironmentInterface, error) {
	clientset, err := v1alpha1.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return clientset.ProblemEnvironment(namespace), nil
}
