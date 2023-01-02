package v1alpha1

import (
	"context"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type WorkerInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.WorkerList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Worker, error)
	Create(ctx context.Context, worker *v1alpha1.Worker) (*v1alpha1.Worker, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type workerClient struct {
	restClient rest.Interface
}

var _ WorkerInterface = &workerClient{}

func (c *workerClient) List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.WorkerList, error) {
	result := v1alpha1.WorkerList{}
	err := c.restClient.
		Get().
		Resource("workers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *workerClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Worker, error) {
	result := v1alpha1.Worker{}
	err := c.restClient.
		Get().
		Resource("workers").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *workerClient) Create(ctx context.Context, worker *v1alpha1.Worker) (*v1alpha1.Worker, error) {
	result := v1alpha1.Worker{}
	err := c.restClient.
		Post().
		Resource("workers").
		Body(worker).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *workerClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Resource("workers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
