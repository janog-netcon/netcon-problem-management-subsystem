package v1alpha1

import (
	"context"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ProblemEnvironmentInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.ProblemEnvironmentList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.ProblemEnvironment, error)
	Create(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment) (*v1alpha1.ProblemEnvironment, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type problemEnvironmentClient struct {
	restClient rest.Interface
	namespace  string
}

var _ ProblemEnvironmentInterface = &problemEnvironmentClient{}

func (c *problemEnvironmentClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.ProblemEnvironmentList, error) {
	result := v1alpha1.ProblemEnvironmentList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problemenvironments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.ProblemEnvironment, error) {
	result := v1alpha1.ProblemEnvironment{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problemenvironments").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) Create(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment) (*v1alpha1.ProblemEnvironment, error) {
	result := v1alpha1.ProblemEnvironment{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("problemenvironments").
		Body(problemEnvironment).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problemenvironments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
