package main

import (
	"context"
	"fmt"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

type AccessMethod string

type AccessHelper interface {
	access(
		ctx context.Context,
		nodeDefinition containerlab.NodeDefinition,
		containerDetails containerlab.ContainerDetails,
	) error
}

var accessHelpers map[AccessMethod]AccessHelper = make(map[AccessMethod]AccessHelper)

func registerAccessHelper(method AccessMethod, helper AccessHelper) {
	if _, ok := accessHelpers[method]; ok {
		panic(fmt.Sprintf("failed to register access helper: method `%s` is already registered", method))
	}
	accessHelpers[method] = helper
}

func findAccessHelper(method AccessMethod) AccessHelper {
	if helper, ok := accessHelpers[method]; ok {
		return helper
	}
	return nil
}
