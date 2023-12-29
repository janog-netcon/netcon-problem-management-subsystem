package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

func init() {
	registerAccessHelper(AccessMethodExec, &ExecAccessHelper{})
}

const AccessMethodExec AccessMethod = "exec"

type ExecAccessHelper struct{}

func (h *ExecAccessHelper) access(
	ctx context.Context,
	nodeDefinition containerlab.NodeDefinition,
	containerDetails containerlab.ContainerDetails,
) error {
	execCommand := defaultExecCommand
	if v, ok := nodeDefinition.Labels[execCommandKey]; ok {
		execCommand = v
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", "-it", containerDetails.Name, execCommand)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	err := cmd.Wait()
	if _, ok := err.(*exec.ExitError); ok {
		// User may occur ExitError, but it's not needed to handle here.
		return nil
	}

	return err
}
