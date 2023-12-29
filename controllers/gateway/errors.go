package controllers

import "errors"

type ErrProblemNotFound struct {
	name string
}

func (e ErrProblemNotFound) Error() string {
	return "Problem not found: " + e.name
}

type ErrProblemEnvironmentNotFound struct {
	name string
}

func (e ErrProblemEnvironmentNotFound) Error() string {
	return "ProblemEnvironment not found: " + e.name
}

type ErrWorkerNotFound struct {
	name string
}

func (e ErrWorkerNotFound) Error() string {
	return "Worker not found: " + e.name
}

type ErrNoAvailableProblemEnvironment struct {
	problemName string
}

func (e ErrNoAvailableProblemEnvironment) Error() string {
	return "No available ProblemEnvironment for: " + e.problemName
}

func AsErrProblemNotFound(err error) (*ErrProblemNotFound, bool) {
	target := ErrProblemNotFound{}
	if errors.As(err, &target) {
		return &target, true
	}
	return nil, false
}

func AsErrProblemEnvironmentNotFound(err error) (*ErrProblemEnvironmentNotFound, bool) {
	target := ErrProblemEnvironmentNotFound{}
	if errors.As(err, &target) {
		return &target, true
	}
	return nil, false
}

func AsErrWorkerNotFound(err error) (*ErrWorkerNotFound, bool) {
	target := ErrWorkerNotFound{}
	if errors.As(err, &target) {
		return &target, true
	}
	return nil, false
}

func AsErrNoAvailableProblemEnvironment(err error) (*ErrNoAvailableProblemEnvironment, bool) {
	target := ErrNoAvailableProblemEnvironment{}
	if errors.As(err, &target) {
		return &target, true
	}
	return nil, false
}
