package main

import (
	"os"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/kubectl-netcon/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
