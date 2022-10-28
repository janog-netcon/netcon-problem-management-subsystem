package drivers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
	"github.com/pkg/errors"
)

const TopologyFileKey = "manifest.yml"

type ContainerLabProblemEnvironmentDriver struct{}

var _ ProblemEnvironmentDriver = &ContainerLabProblemEnvironmentDriver{}

func NewContainerLabProblemEnvironmentDriver() *ContainerLabProblemEnvironmentDriver {
	return &ContainerLabProblemEnvironmentDriver{}
}

func (d *ContainerLabProblemEnvironmentDriver) ensureDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func (d *ContainerLabProblemEnvironmentDriver) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (d *ContainerLabProblemEnvironmentDriver) createOrUpdateFile(path string, content []byte) (bool, error) {
	if d.fileExists(path) {
		currentContent, err := os.ReadFile(path)
		if err != nil || bytes.Equal(currentContent, content) {
			return false, err
		}
	}
	return true, os.WriteFile(path, content, 0755)
}

func (d *ContainerLabProblemEnvironmentDriver) delete(path string) (bool, error) {
	if !d.fileExists(path) {
		return false, nil
	}
	return true, os.RemoveAll(path)
}

func (d *ContainerLabProblemEnvironmentDriver) ensureFilesFor(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	client *containerlab.ContainerLabClient,
) error {
	configMap := corev1.ConfigMap{}
	if err := reader.Get(ctx, types.NamespacedName{
		Namespace: problemEnvironment.Namespace,
		Name:      problemEnvironment.Spec.Files.ConfigMapRef.Name,
	}, &configMap); err != nil {
		return errors.Wrap(err, "failed to load configMap")
	}

	workingDirectory := client.WorkingDirectoryPath()
	if err := d.ensureDirectory(workingDirectory); err != nil {
		return errors.Wrap(err, "failed to create working directory")
	}

	for key, content := range configMap.Data {
		path := path.Join(workingDirectory, key)
		if _, err := d.createOrUpdateFile(path, []byte(content)); err != nil {
			return errors.Wrap(err, "failed to create or update file")
		}
	}

	topologyFilePath := client.TopologyFilePath()
	topology, err := ioutil.ReadFile(topologyFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to load topology file")
	}

	topologyConfig := containerlab.Config{}
	if err := yaml.UnmarshalStrict(topology, &topologyConfig); err != nil {
		return errors.Wrap(err, "failed to unmarshal topology file")
	}

	topologyConfig.Name = problemEnvironment.Name

	topology, err = yaml.Marshal(topologyConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal topology file")
	}

	if _, err := d.createOrUpdateFile(topologyFilePath, topology); err != nil {
		return errors.Wrap(err, "failed to create or update topology file")
	}

	return nil
}

func (d *ContainerLabProblemEnvironmentDriver) getContainerLabClientFor(
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) *containerlab.ContainerLabClient {
	manifestPath := path.Join("data", problemEnvironment.Name, "manifest.yml")
	return containerlab.NewContainerLabClient(manifestPath)
}

func (d *ContainerLabProblemEnvironmentDriver) getDirectoryPathFor(
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) string {
	return path.Join("data", problemEnvironment.Name)
}

// Check implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Check(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerDetailStatus, error) {
	log := log.FromContext(ctx)

	client := containerlab.NewContainerLabClientFor(&problemEnvironment)

	directoryPath := client.WorkingDirectoryPath()
	if !d.fileExists(directoryPath) {
		// directory not found, ProblemEnvironment haven't been created
		log.V(1).Info("working directory not found, skip to inspect")
		return StatusDown, nil, nil
	}

	topologyFilePath := client.TopologyFilePath()
	if !d.fileExists(topologyFilePath) {
		log.V(1).Info("topology file not found, skip to inspect")
		return StatusDown, nil, nil
	}

	labData, err := client.Inspect(ctx)
	if err != nil {
		return StatusUnknown, nil, fmt.Errorf("failed to inspect ContainerLab: %w", err)
	}

	containerStatuses := []netconv1alpha1.ContainerDetailStatus{}
	for i := range labData.Containers {
		containerDetail := &labData.Containers[i]

		containerPrefix := fmt.Sprintf("clab-%s-", problemEnvironment.Name)
		containerName := strings.ReplaceAll(containerDetail.Name, containerPrefix, "")

		containerStatuses = append(containerStatuses, netconv1alpha1.ContainerDetailStatus{
			Name:                containerName,
			Image:               containerDetail.Image,
			ContainerID:         containerDetail.ContainerID,
			Ready:               containerDetail.State == "running",
			ManagementIPAddress: containerDetail.IPv4Address,
		})
	}

	return StatusUp, containerStatuses, nil
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	client := containerlab.NewContainerLabClientFor(&problemEnvironment)

	if err := d.ensureFilesFor(ctx, reader, &problemEnvironment, client); err != nil {
		return err
	}

	if err := client.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy ContainerLab: %w", err)
	}

	return nil
}

// Destroy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	client := containerlab.NewContainerLabClientFor(&problemEnvironment)

	status, _, err := d.Check(ctx, reader, problemEnvironment)
	if err != nil {
		return fmt.Errorf("failed to destroy ContainerLab: %w", err)
	}

	if status != StatusDown {
		// Destroy may be failed when status is StatusUnknown
		if err := client.Destroy(ctx); err != nil {
			return fmt.Errorf("failed to destroy ContainerLab: %w", err)
		}
	}

	directoryPath := client.WorkingDirectoryPath()
	if _, err := d.delete(directoryPath); err != nil {
		return fmt.Errorf("failed to delete directory for ProblemEnvironment: %w", err)
	}

	return nil
}
