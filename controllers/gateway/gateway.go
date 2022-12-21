package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

const (
	vmdb_api_base_url = "http://vmdb-api:8080" // TODO: vmdb-apiに合わせて変える
)

type Gateway struct {
	client.Client
}

type ProblemEnvironmentResponse struct {
	Response netconv1alpha1.ProblemEnvironmentList `json:"response"`
}

type PostProblemEnvironmentRequest struct {
	ProblemName string `json:"problem_name"`
}

var _ manager.Runnable = &Gateway{}
var _ inject.Client = &Gateway{}

func (g *Gateway) InjectClient(client client.Client) error {
	g.Client = client
	return nil
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
	e.POST("/problem", g.PostProblem(ctx))
	e.DELETE("/problem/:name", g.DeleteProblem(ctx))

	e.Logger.Fatal(e.Start(":8082"))

	return nil
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Gateway for score server")
}

func (g *Gateway) GetProblemEnvironmentHandlerFunc(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		problemEnvironmentName := c.Param("name")

		problemEnvironments, err := g.GetProblemEnvironmentList(ctx, problemEnvironmentName)
		if err != nil {
			c.Echo().Logger.Errorf("failed to get problem environment list", err)
			return err
		}

		selectedItems := []netconv1alpha1.ProblemEnvironment{}
		for _, pe := range problemEnvironments.Items {
			if pe.Name == problemEnvironmentName {
				selectedItems = append(selectedItems, pe)
			}
		}
		problemEnvironments.Items = selectedItems

		problemEnvironmentResponse := ProblemEnvironmentResponse{}
		problemEnvironmentResponse.Response = problemEnvironments

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		encoder.Encode(problemEnvironmentResponse)

		return c.JSONBlob(http.StatusOK, b.Bytes())
	}
}

func (g *Gateway) PostProblem(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := log.FromContext(ctx)
		postProblemEnvironmentRequest := PostProblemEnvironmentRequest{}

		err := c.Bind(&postProblemEnvironmentRequest)
		if err != nil {
			log.Error(err, "failed to bind to PostProblemEnvironmentRequest")
			return err
		}

		problemName := postProblemEnvironmentRequest.ProblemName
		problemNameLabel := client.MatchingLabels{"problemName": problemName}
		problemEnvironments := netconv1alpha1.ProblemEnvironmentList{}
		if err := g.Client.List(ctx, &problemEnvironments, problemNameLabel); err != nil {
			c.Echo().Logger.Errorf("could not list ProblemEnvironments", err)
			log.Error(err, "could not list ProblemEnvironments")
			return err
		}

		selectedItems := []netconv1alpha1.ProblemEnvironment{}
		for _, pe := range problemEnvironments.Items {
			assignedCondition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionAssigned)
			readyCondition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionReady)
			if pe.Labels["problemName"] == problemName && assignedCondition == metav1.ConditionFalse && readyCondition == metav1.ConditionTrue {
				selectedItems = append(selectedItems, pe)

				message := "Assigned ProblemEnvironemnt " + pe.Name
				util.SetProblemEnvironmentCondition(
					&pe,
					netconv1alpha1.ProblemEnvironmentConditionAssigned,
					metav1.ConditionTrue,
					"AssignedProblemEnvironemnt",
					message,
				)
				break
			}
		}
		if len(selectedItems) == 0 {
			return c.JSONBlob(http.StatusBadRequest, nil)
		}

		problemEnvironments.Items = selectedItems

		problemEnvironmentResponse := ProblemEnvironmentResponse{}
		problemEnvironmentResponse.Response = problemEnvironments

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		encoder.Encode(problemEnvironmentResponse)

		return c.JSONBlob(http.StatusOK, b.Bytes())
	}
}

func (g *Gateway) DeleteProblem(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		problemEnvironmentName := c.Param("name")

		problemEnvironments, err := g.GetProblemEnvironmentList(ctx, problemEnvironmentName)
		if err != nil {
			c.Echo().Logger.Errorf("failed to get problem environment list", err)
			return err
		}

		for _, pe := range problemEnvironments.Items {
			// assignedCondition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionAssigned)
			if pe.Name == problemEnvironmentName {
				if err := g.Client.Delete(ctx, &pe); err != nil {
					c.Echo().Logger.Errorf("failed to delete", err)
					return err
				}
				break
			}
		}

		return c.JSONBlob(http.StatusOK, nil)
	}
}

func (g *Gateway) GetProblemEnvironmentList(ctx context.Context, problemEnvironmentName string) (netconv1alpha1.ProblemEnvironmentList, error) {
	log := log.FromContext(ctx)
	problemEnvironmentNameParts := strings.Split(problemEnvironmentName, "-")
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
		return problemEnvironments, err
	}
	return problemEnvironments, nil
}
