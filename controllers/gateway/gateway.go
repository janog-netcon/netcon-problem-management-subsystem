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

const (
	vmdb_api_base_url = "http://vmdb-api:8080" // TODO: vmdb-apiに合わせて変える
)

type Gateway struct {
	client.Client
}

type ProblemEnvironment struct {
	Status           string      `json:"status"`
	Host             string      `json:"host"`
	User             string      `json:"user"`
	Password         string      `json:"password"`
	ProblemID        uuid.UUID   `json:"problem_id"`
	CreatedAt        time.Time   `json:"created_at"`
	Name             string      `json:"name"`
	MachineImageName string      `json:"machine_image_name"` // nullable
	Project          string      `json:"project"`            // nullable
	Zone             string      `json:"zone"`               // nullable
}

type GetProblemEnvironmentResponse struct {
	Response         []ProblemEnvironment      `json:"response"`
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
	e.GET("/instance", getInstances)
	e.GET("/instance/:name", getInstance)
	e.POST("/instance", postInstance)
	e.DELETE("/instance/:name", deleteInstance)

	e.Logger.Fatal(e.Start(":8082"))

	return nil
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Gateway for score server")
}

func getInstances(c echo.Context, ctx context.Context) error {
	log := log.FromContext(ctx)

	url := vmdb_api_base_url + "/problem-environments"
	r, err := httpClient.Get(url)
	if err != nil {
		log.Error(err, "failed to call GET /problem-environments")		
	    // return c.JSONBlob(http.StatusInternalServerError, b.Bytes())
		return nil
	}
	defer r.Body.Close()

	pes := []ProblemEnvironment{}
    jsonResponse = json.NewDecoder(r.Body).Decode(&pes)

	getProblemEnvironmentResponse := GetProblemEnvironmentResponse{}
	getProblemEnvironmentResponse.Response = jsonResponse

	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.Encode(getProblemEnvironmentResponse)

	return c.JSONBlob(http.StatusOK, b.Bytes())
}

func getInstance(c echo.Context) error {
	return c.String(http.StatusOK, "health")
}

func postInstance(c echo.Context) error {
	return c.String(http.StatusOK, "health")
}

func deleteInstance(c echo.Context) error {
	return c.String(http.StatusOK, "health")
}