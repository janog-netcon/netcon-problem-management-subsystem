package controllers

type ProblemEnvironment struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type GetProblemEnvironmentResponse ProblemEnvironment

type AcquireProblemEnvironmentRequest struct {
	ProblemName string `json:"problemName"`
}

type AcquireProblemEnvironmentResponse ProblemEnvironment
