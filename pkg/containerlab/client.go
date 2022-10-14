package containerlab

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
)

type ContainerLabClient struct {
	workingDirectoryPath string
	topologyFileName     string
}

func NewContainerLabClient(topologyFilePath string) *ContainerLabClient {
	workingDirectoryPath := filepath.Dir(topologyFilePath)
	topologyFileName := filepath.Base(topologyFilePath)

	return &ContainerLabClient{
		workingDirectoryPath: workingDirectoryPath,
		topologyFileName:     topologyFileName,
	}
}

func NewContainerLabClientFor(
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) *ContainerLabClient {
	if problemEnvironment == nil {
		return nil
	}

	topologyFilePath := path.Join("data", problemEnvironment.Name, "manifest.yml")
	return NewContainerLabClient(topologyFilePath)
}

func (c *ContainerLabClient) WorkingDirectoryPath() string {
	return c.workingDirectoryPath
}

func (c *ContainerLabClient) TopologyFilePath() string {
	return path.Join(c.workingDirectoryPath, c.topologyFileName)
}

func (c *ContainerLabClient) TopologyFileName() string {
	return c.topologyFileName
}

func (c *ContainerLabClient) LoadTopologyFile() (*Config, error) {
	topologyFileData, err := os.ReadFile(c.TopologyFilePath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to read topology file")
	}

	config := Config{}
	if err := yaml.Unmarshal(topologyFileData, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal topology file")
	}

	return &config, nil
}

func (c *ContainerLabClient) Deploy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx,
		"clab",
		"--log-level", "debug", "-t", c.topologyFileName, "deploy",
	)
	cmd.Dir = c.workingDirectoryPath
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *ContainerLabClient) Destroy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx,
		"clab",
		"--log-level", "debug", "-t", c.topologyFileName, "destroy",
	)
	cmd.Dir = c.workingDirectoryPath
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *ContainerLabClient) Inspect(ctx context.Context) (*LabData, error) {
	cmd := exec.CommandContext(ctx,
		"clab",
		"-t", c.topologyFileName, "inspect", "-f", "json",
	)
	cmd.Dir = c.workingDirectoryPath
	cmd.Stderr = nil

	stdout, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data := LabData{}
	if err := json.Unmarshal(stdout, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	return &data, nil
}
