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

/*
    Response body structure from score-server
 */
type GetProblemEnvironment struct {
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

/*
	POST request body structure to score-server
*/
type PostProblemEnvironmentRequest struct {
	Status           string      `json:"status"`
	Host             string      `json:"host"`
	User             string      `json:"user"`
	Password         string      `json:"password"`
	ProblemID        uuid.UUID   `json:"problem_id"`
	// CreatedAt        time.Time   `json:"created_at"`
	Name             string      `json:"name"`
	MachineImageName string      `json:"machine_image_name"` // nullable
	Project          string      `json:"project"`            // nullable
	Zone             string      `json:"zone"`               // nullable
	Service          string      `json:"service"`	
	Port             int         `json:"port"`	
}

/*
    Response body structure for client
	Need to convert struct from ProblemEnvironment to this since vmms has different keys

	i.e.,
        "instance_name": content['name'],
        "machine_image_name": content['machine_image_name'],
        "domain": content['host'],
        "project": content['project'],
        "zone": content['zone'],
        "status": content['status'],
        "problem_id": content['problem_id'],
        "user_id": content['user'],
        "password": content['password'],
        "created_at": content['created_at']
 */
type GetInstances struct {
	Name             string      `json:"instance_name"`
	MachineImageName string      `json:"machine_image_name"` // nullable
	Host             string      `json:"domain"`
	Project          string      `json:"project"`            // nullable
	Zone             string      `json:"zone"`               // nullable
	Status           string      `json:"status"`
	ProblemID        uuid.UUID   `json:"problem_id"`
	User             string      `json:"user_id"`
	Password         string      `json:"password"`
	CreatedAt        time.Time   `json:"created_at"`
}

type GetInstancesResponse struct {
	Response         []GetInstances      `json:"response"`
}

type PostInstanceRequest struct {
	MachineImageName string      `json:"machine_image_name"` // nullable
	ProblemID        uuid.UUID   `json:"problem_id"`
	Project          string      `json:"project"`            // nullable
	Zone             string      `json:"zone"`               // nullable	
}

type PostInstance struct {
	Name             string      `json:"instance_name"`
	MachineImageName string      `json:"machine_image_name"` // nullable
	Host             string      `json:"domain"`
	Project          string      `json:"project"`            // nullable
	Zone             string      `json:"zone"`               // nullable
	Status           string      `json:"status"`
	ProblemID        uuid.UUID   `json:"problem_id"`
	User             string      `json:"user_id"`
	Password         string      `json:"password"`
}

type PostInstanceResponse struct {
	Response         []PostInstance      `json:"response"`	
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
	    return c.JSONBlob(http.StatusInternalServerError, nil)
	}
	defer r.Body.Close()

	pes := []GetProblemEnvironment{}
    decodedPes = json.NewDecoder(r.Body).Decode(&pes)

	getInstances := []GetInstances{}
	getInstances = convertProblemEnvironmentsToGetInstances(decodedPes)

	getInstancesResponse := GetInstancesResponse{}
	getInstancesResponse.Response = getInstances

	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.Encode(getInstancesResponse)

	return c.JSONBlob(http.StatusOK, b.Bytes())
}

func getInstance(c echo.Context, ctx context.Context) error {
	log := log.FromContext(ctx)
	instance_name := c.Param("name")

	url := vmdb_api_base_url + "/problem-environments/" + instance_name
	r, err := httpClient.Get(url)
	if err != nil {
		log.Error(err, "failed to call GET /problem-environments/" + instance_name)		
	    return c.JSONBlob(http.StatusInternalServerError, nil)
	}
	defer r.Body.Close()

	pes := []GetProblemEnvironment{}
    decodedPes = json.NewDecoder(r.Body).Decode(&pes)

	getInstances := []GetInstances{}
	getInstances = convertProblemEnvironmentsToGetInstances(decodedPes)

	getInstancesResponse := GetInstancesResponse{}
	getInstancesResponse.Response = getInstances

	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.Encode(getInstancesResponse)

	return c.JSONBlob(http.StatusOK, b.Bytes())
}

func postInstance(c echo.Context, ctx context.Context) error {
	log := log.FromContext(ctx)

	// validate params
	var reqBodyFromClient PostInstanceRequest
	if err := c.Bind(&reqBodyFromClient); err != nil {
		log.Error(err, "failed to parse request body")		
	    return c.JSONBlob(http.StatusBadRequest, nil)
	}

	// TODO: reqBodyFromClient.MachineImageName の 割り当て可能な VM 情報を取ってくる。
	// 必要なVM情報は Public IP address, password, 問題環境名 (e.g., pro001-blah) , 状態 の4つ
	// ここの時点で割り当て可能な状態のみに絞って、score-server側に作成依頼を投げる。

	var reqBodyToScoreServer PostProblemEnvironmentRequest
	// TODO: 以前は VM の GCE の status を埋めていた。
	// vmdb-apiだと external_status ってやつで、何に使ってるか確認する。
	// https://github.com/janog-netcon/netcon-score-server/blob/c0401cc8bdec06c71fa70b0b32db4ad1831c0f49/vmdb-api/main.go#L51
	// 基本的には不足してなければ、RUNNING入れとけば良い?
	reqBodyToScoreServer.Status = 

	// TODO: 以前は VM の Public IP addressだった。今回も同じ想定?
	reqBodyToScoreServer.Host = 

	// TODO: これは SSH する際に使ってたログインユーザか確認する
	// ssh nc_{{問題環境名}}@{{VMのPublic IP}} で接続するのであれば、User は nc_{{問題環境名}} が正しい?
	reqBodyToScoreServer.User = "janoger"

	// TODO: これは SSH する際に使ってたログインユーザか確認する
	// Password は 何か割り当てられてるのを取ってくるのか、あるいはなし?
	reqBodyToScoreServer.Password = 
	reqBodyToScoreServer.ProblemID = reqBodyFromClient.ProblemID

	// TODO: {{問題環境名}} で良いか確認する (e.g., pro001-blah)
	reqBodyToScoreServer.Name = 
	reqBodyToScoreServer.MachineImageName = reqBodyFromClient.MachineImageName
	reqBodyToScoreServer.Project = reqBodyFromClient.Project
	reqBodyToScoreServer.Zone = reqBodyFromClient.Zone
	reqBodyToScoreServer.Service = "SSH"
	reqBodyToScoreServer.Port = 50080

	url := vmdb_api_base_url + "/problem-environments/" + instance_name
	r, err := http.NewRequest("POST", url, reqBodyToScoreServer)
	if err != nil {
		log.Error(err, "failed to call POST /problem-environments/" + instance_name)		
	    return c.JSONBlob(http.StatusInternalServerError, nil)
	}
	defer r.Body.Close()

	var postInstance PostInstance
	postInstance.Name = reqBodyToScoreServer.Name
	postInstance.MachineImageName = reqBodyToScoreServer.MachineImageName
	postInstance.Host = reqBodyToScoreServer.Host
	postInstance.Project = reqBodyToScoreServer.Project
	postInstance.Zone = reqBodyToScoreServer.Zone
	postInstance.Status = reqBodyToScoreServer.Status
	postInstance.ProblemID = reqBodyToScoreServer.ProblemID
	postInstance.User = reqBodyToScoreServer.User
	postInstance.Password = reqBodyToScoreServer.Password

	var postInstanceResponse PostInstanceResponse
	postInstanceResponse.Response = postInstance

	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.Encode(peFromDb)

	return c.String(http.StatusOK, postInstanceResponse)
}

/*
    TODO: 実装する
*/
func deleteInstance(c echo.Context) error {
	return c.String(http.StatusOK, "{\"response\":{\"is_deleted\":\"true\"}}")
}

func convertProblemEnvironmentsToGetInstances(pes []GetInstances{}) []GetProblemEnvironment{} {
	getPes := []GetProblemEnvironment{}
	for _, v := range pes {
		getPe := GetProblemEnvironment{}
		getPe.Name = v.Name
		getPe.MachineImageName = v.MachineImageName
		getPe.Host = v.Host
		getPe.Name = v.Name
		getPe.Project = v.Project
		getPe.Zone = v.Zone
		getPe.Status = v.Status
		getPe.ProblemID = v.ProblemID
		getPe.User = v.User
		getPe.Password = v.Password
		getPe.CreatedAt = v.CreatedAt
		getPes = append(getPes, getPe)
	}
	return getPes
}