package v1alpha1

import (
	"context"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// ref: https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/pod.go
type ProblemEnvironmentInterface interface {
	Create(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.CreateOptions) (*v1alpha1.ProblemEnvironment, error)
	Update(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.UpdateOptions) (*v1alpha1.ProblemEnvironment, error)
	UpdateStatus(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.UpdateOptions) (*v1alpha1.ProblemEnvironment, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.ProblemEnvironment, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.ProblemEnvironmentList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type problemEnvironmentClient struct {
	restClient rest.Interface
	namespace  string
}

var _ ProblemEnvironmentInterface = &problemEnvironmentClient{}

func (c *problemEnvironmentClient) Create(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.CreateOptions) (*v1alpha1.ProblemEnvironment, error) {
	result := v1alpha1.ProblemEnvironment{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("problemenvironments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(problemEnvironment).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) Update(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.UpdateOptions) (*v1alpha1.ProblemEnvironment, error) {
	result := v1alpha1.ProblemEnvironment{}
	err := c.restClient.
		Put().
		Namespace(c.namespace).
		Resource("problemenvironments").
		Name(problemEnvironment.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(problemEnvironment).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) UpdateStatus(ctx context.Context, problemEnvironment *v1alpha1.ProblemEnvironment, opts metav1.UpdateOptions) (*v1alpha1.ProblemEnvironment, error) {
	result := v1alpha1.ProblemEnvironment{}
	err := c.restClient.
		Put().
		Namespace(c.namespace).
		Resource("problemenvironments").
		Name(problemEnvironment.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(problemEnvironment).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *problemEnvironmentClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.
		Delete().
		Namespace(c.namespace).
		Resource("problemenvironments").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
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

func (c *problemEnvironmentClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("problemenvironments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
