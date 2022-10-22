package controllers

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

// ref: https://kubernetes.io/ja/docs/concepts/architecture/_print/#%E3%83%8F%E3%83%BC%E3%83%88%E3%83%93%E3%83%BC%E3%83%88

type HeartbeatAgent struct {
	client.Client
}

// Controller-manager will inject Kubernetes API client to inject.Client
var _ inject.Client = &HeartbeatAgent{}

// Controller-manager can run manager.Runnable
var _ manager.Runnable = &HeartbeatAgent{}

// InjectClient implements inject.Client
func (a *HeartbeatAgent) InjectClient(c client.Client) error {
	a.Client = c
	return nil
}

// Start implements manager.Runnable
func (a *HeartbeatAgent) Start(c context.Context) error {
	return nil
}
