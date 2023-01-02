package v1alpha1

import (
	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ClientsetInterface interface {
	Problem(namespace string) ProblemInterface
	ProblemEnvironment(namespace string) ProblemEnvironmentInterface
	Worker() WorkerInterface
}

type clientset struct {
	restClient rest.Interface
}

var _ ClientsetInterface = &clientset{}

func NewForConfig(c *rest.Config) (ClientsetInterface, error) {
	config := *c
	config.ContentConfig.GroupVersion = &v1alpha1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &clientset{restClient: client}, nil
}

func (c *clientset) Problem(namespace string) ProblemInterface {
	return &problemClient{
		restClient: c.restClient,
		namespace:  namespace,
	}
}

func (c *clientset) ProblemEnvironment(namespace string) ProblemEnvironmentInterface {
	return &problemEnvironmentClient{
		restClient: c.restClient,
		namespace:  namespace,
	}
}

func (c *clientset) Worker() WorkerInterface {
	return &workerClient{
		restClient: c.restClient,
	}
}
