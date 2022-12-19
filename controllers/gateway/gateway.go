package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

const (
	vmdb_api_base_url = "http://vmdb-api:8080" // TODO: vmdb-apiに合わせて変える
)

type Gateway struct {
	client.Client
}

type ProblemResponse struct {
	Response netconv1alpha1.ProblemEnvironmentList `json:"response"`
}

var _ manager.Runnable = &Gateway{}
var _ inject.Client = &Gateway{}

func (g *Gateway) InjectClient(client client.Client) error {
	g.Client = client
	return nil
}

func (g *Gateway) GetProblem(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := log.FromContext(ctx)
		// e.g., pro-001-v8k24
		problemEnvironmentName := c.Param("name")

		problemEnvironmentNameParts := strings.Split(problemEnvironmentName, "-")
		// e.g., pro-001
		problemName := ""
		for i, v := range problemEnvironmentNameParts {
			problemName += v
			if i >= len(problemEnvironmentNameParts)-2 {
				break
			}
			problemName += "-"
		}
		problemNameLabel := client.MatchingLabels{"problemName": problemName}
		problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
		if err := g.Client.List(ctx, &problemEnvironments, problemNameLabel); err != nil {
			log.Error(err, "could not list ProblemEnvironments")
			return err
		}

		selectedItems := []netconv1alpha1.ProblemEnvironment{}
		for _, pe := range problemEnvironments.Items {
			log.Info(pe.Name)
			if pe.Name == problemEnvironmentName {
				selectedItems = append(selectedItems, pe)
			}
		}
		problemEnvironments.Items = selectedItems

		problemResponse := ProblemResponse{}
		problemResponse.Response = problemEnvironments

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		encoder.Encode(problemResponse)

		return c.JSONBlob(http.StatusOK, b.Bytes())
	}
}

func (g *Gateway) Start(ctx context.Context) error {
	_ = log.FromContext(ctx)

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", hello)
	e.GET("/problem/:name", g.GetProblem(ctx))
	// e.POST("/problem", postProblem)
	// e.DELETE("/problem/:name", deleteProblem)

	e.Logger.Fatal(e.Start(":8082"))

	return nil
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Gateway for score server")
}

func postProblem(c echo.Context, ctx context.Context) error {
	// log := log.FromContext(ctx)

	// var b bytes.Buffer
	// encoder := json.NewEncoder(&b)
	// encoder.SetEscapeHTML(false)
	// encoder.Encode(peFromDb)

	// return c.String(http.StatusOK, postInstanceResponse)
	return nil
}

func deleteProblem(c echo.Context, ctx context.Context) error {
	// log := log.FromContext(ctx)
	// instance_name := c.Param("name")

	// TODO: instance_name で 削除対象のPEを取得する

	//

	// return c.String(http.StatusOK, "{\"response\":{\"is_deleted\":\"true\"}}")
	return nil
}
