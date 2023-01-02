package v1alpha1

import (
	"context"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ProblemInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.ProblemList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Problem, error)
	Create(ctx context.Context, problem *v1alpha1.Problem) (*v1alpha1.Problem, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type problemClient struct {
	restClient rest.Interface
	namespace  string
}

var _ ProblemInterface = &problemClient{}

func (c *problemClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.ProblemList, error) {
	result := v1alpha1.ProblemList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problems").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Problem, error) {
	result := v1alpha1.Problem{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problems").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemClient) Create(ctx context.Context, problem *v1alpha1.Problem) (*v1alpha1.Problem, error) {
	result := v1alpha1.Problem{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("problems").
		Body(problem).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problems").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
