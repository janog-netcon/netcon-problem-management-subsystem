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

type ExecAccessHelper struct {
}

func (h *ExecAccessHelper) access(
	ctx context.Context,
	nodeDefinition containerlab.NodeDefinition,
	containerDetails containerlab.ContainerDetails,
) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-it", containerDetails.Name, "sh")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	err := cmd.Wait()
	if err, ok := err.(*exec.ExitError); ok {
		os.Exit(err.ExitCode())
	}

	return err
}
