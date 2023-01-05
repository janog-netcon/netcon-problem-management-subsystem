package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
)

type Gateway struct {
	client.Client
}

type ProblemEnvironmentItem struct {
	ProblemEnvironment netconv1alpha1.ProblemEnvironment `json:"problemEnvironment"`
	Worker             netconv1alpha1.Worker             `json:"worker"`
}

type ProblemEnvironmentList struct {
	Items []ProblemEnvironmentItem `json:"items"`
}

type ProblemEnvironmentResponse struct {
	Response ProblemEnvironmentList `json:"response"`
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
	e.GET("/problem/:name", g.GetProblemEnvironmentHandlerFunc(ctx))
	e.POST("/problem", g.PostProblemEnvironmentHandlerFunc(ctx))
	e.DELETE("/problem/:name", g.DeleteProblemEnvironmentHandlerFunc(ctx))

	e.Logger.Fatal(e.Start(":8082"))

	return nil
}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Gateway for score server")
}

func (g *Gateway) GetProblemEnvironmentHandlerFunc(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := log.FromContext(ctx)
		problemEnvironmentName := c.Param("name")

		netconV1Alpha1ProblemEnvironment, err := g.GetProblemEnvironment(ctx, problemEnvironmentName)
		if err != nil {
			log.Error(err, "failed to get problem environment")
			return err
		}

		workerName := netconV1Alpha1ProblemEnvironment.Spec.WorkerName
		worker, err := g.GetWorker(ctx, workerName)
		if err != nil {
			log.Error(err, "failed to get worker")
			return err
		}

		problemEnvironmentItem := ProblemEnvironmentItem{}
		problemEnvironmentItem.ProblemEnvironment = netconV1Alpha1ProblemEnvironment
		problemEnvironmentItem.Worker = worker

		problemEnvironmentItems := []ProblemEnvironmentItem{}
		problemEnvironmentItems = append(problemEnvironmentItems, problemEnvironmentItem)

		problemEnvironmentList := ProblemEnvironmentList{}
		problemEnvironmentList.Items = problemEnvironmentItems

		problemEnvironmentResponse := ProblemEnvironmentResponse{}
		problemEnvironmentResponse.Response = problemEnvironmentList

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		encoder.Encode(problemEnvironmentResponse)

		return c.JSONBlob(http.StatusOK, b.Bytes())
	}
}

func (g *Gateway) PostProblemEnvironmentHandlerFunc(ctx context.Context) echo.HandlerFunc {
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
			log.Error(err, "could not list ProblemEnvironments")
			return err
		}

		selectedItems := []netconv1alpha1.ProblemEnvironment{}
		for _, pe := range problemEnvironments.Items {
			assignedCondition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionAssigned)
			readyCondition := util.GetProblemEnvironmentCondition(&pe, netconv1alpha1.ProblemEnvironmentConditionReady)
			if pe.Labels["problemName"] == problemName && assignedCondition == metav1.ConditionFalse && readyCondition == metav1.ConditionTrue {
				selectedItems = append(selectedItems, pe)

				message := "Assigned ProblemEnvironment " + pe.Name
				util.SetProblemEnvironmentCondition(
					&pe,
					netconv1alpha1.ProblemEnvironmentConditionAssigned,
					metav1.ConditionTrue,
					"AssignedProblemEnvironment",
					message,
				)
				g.updateStatus(ctx, &pe, ctrl.Result{})
				break
			}
		}
		if len(selectedItems) == 0 {
			log.Error(err, "no such applicable problem environment")
			return c.JSONBlob(http.StatusInternalServerError, nil)
		}

		workerName := selectedItems[0].Spec.WorkerName
		worker, err := g.GetWorker(ctx, workerName)
		if err != nil {
			log.Error(err, "failed to get worker")
			return err
		}
		problemEnvironmentItem := ProblemEnvironmentItem{}
		problemEnvironmentItem.ProblemEnvironment = selectedItems[0]
		problemEnvironmentItem.Worker = worker

		problemEnvironmentItems := []ProblemEnvironmentItem{}
		problemEnvironmentItems = append(problemEnvironmentItems, problemEnvironmentItem)

		problemEnvironmentList := ProblemEnvironmentList{}
		problemEnvironmentList.Items = problemEnvironmentItems

		problemEnvironmentResponse := ProblemEnvironmentResponse{}
		problemEnvironmentResponse.Response = problemEnvironmentList

		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		encoder.Encode(problemEnvironmentResponse)

		return c.JSONBlob(http.StatusOK, b.Bytes())
	}
}

func (g *Gateway) DeleteProblemEnvironmentHandlerFunc(ctx context.Context) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := log.FromContext(ctx)
		problemEnvironmentName := c.Param("name")

		problemEnvironment, err := g.GetProblemEnvironment(ctx, problemEnvironmentName)
		if err != nil {
			log.Error(err, "failed to get problem environment list")
			return err
		}

		if err := g.Client.Delete(ctx, &problemEnvironment); err != nil {
			log.Error(err, "failed to deletet")
			return err
		}

		return c.JSONBlob(http.StatusOK, nil)
	}
}

func (g *Gateway) updateStatus(
	ctx context.Context,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	res ctrl.Result,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if err := g.Status().Update(ctx, problemEnvironment); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}
	return res, nil
}

func (g *Gateway) GetProblemEnvironment(ctx context.Context, problemEnvironmentName string) (netconv1alpha1.ProblemEnvironment, error) {
	log := log.FromContext(ctx)
	problemEnvironment := netconv1alpha1.ProblemEnvironment{}
	if err := g.Client.Get(ctx, types.NamespacedName{Namespace: "netcon", Name: problemEnvironmentName}, &problemEnvironment); err != nil {
		log.Error(err, "could not get ProblemEnvironments")
		return problemEnvironment, err
	}
	return problemEnvironment, nil
}

func (g *Gateway) GetWorker(ctx context.Context, workerName string) (netconv1alpha1.Worker, error) {
	log := log.FromContext(ctx)
	worker := netconv1alpha1.Worker{}
	if err := g.Client.Get(ctx, types.NamespacedName{Name: workerName}, &worker); err != nil {
		log.Error(err, "could not get worker")
		return worker, err
	}
	return worker, nil
}
