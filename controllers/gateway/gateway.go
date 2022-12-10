package controllers

import (
	"context"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", health)

	e.Logger.Fatal(e.Start(":8080"))

	return nil
}

// ハンドラーを定義
func health(c echo.Context) error {
	return c.String(http.StatusOK, "health")
}
