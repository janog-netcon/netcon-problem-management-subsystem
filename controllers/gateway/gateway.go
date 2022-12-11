package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type Gateway struct {
	client.Client
}

var _ manager.Runnable = &Gateway{}
var _ inject.Client = &Gateway{}

func (g *Gateway) InjectClient(client client.Client) error {
	g.Client = client
	return nil
}

func (g *Gateway) Start(ctx context.Context) error {
	_ = log.FromContext(ctx)

	// TODO: spawn HTTP server

	return nil
}
