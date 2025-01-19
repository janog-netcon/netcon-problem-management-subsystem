package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/shlex"
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
	isAdmin bool,
) error {
	execCommand := defaultExecCommand
	if v, ok := nodeDefinition.Labels[execCommandForAdminKey]; ok && isAdmin {
		execCommand = v
	} else if v, ok := nodeDefinition.Labels[execCommandKey]; ok {
		execCommand = v
	}

	commands, err := shlex.Split(execCommand)
	if err != nil {
		return fmt.Errorf("failed to parse exec command: %w", err)
	}

	commands = append([]string{"exec", "-it", containerDetails.Name}, commands...)

	cmd := exec.CommandContext(ctx, "docker", commands...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	err = cmd.Wait()
	if _, ok := err.(*exec.ExitError); ok {
		// User may occur ExitError, but it's not needed to handle here.
		return nil
	}

	return err
}
