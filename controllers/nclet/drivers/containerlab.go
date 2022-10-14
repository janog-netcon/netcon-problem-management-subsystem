package drivers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

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

func (d *ContainerLabProblemEnvironmentDriver) loadTopologyFile(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) ([]byte, error) {
	log := log.FromContext(ctx)

	configMap := corev1.ConfigMap{}
	if err := reader.Get(ctx, types.NamespacedName{
		Namespace: problemEnvironment.Namespace,
		Name:      problemEnvironment.Spec.TopologyFile.ConfigMapRef.Name,
	}, &configMap); err != nil {
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	topology, ok := configMap.Data[problemEnvironment.Spec.TopologyFile.ConfigMapRef.Key]
	if !ok {
		err := fmt.Errorf(
			"ConfigMap found, but key `%s` missing",
			problemEnvironment.Spec.TopologyFile.ConfigMapRef.Key,
		)
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	return []byte(topology), nil
}

func (d *ContainerLabProblemEnvironmentDriver) getTopologyFileFor(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) ([]byte, error) {
	log := log.FromContext(ctx)

	topology, err := d.loadTopologyFile(ctx, reader, problemEnvironment)
	if err != nil {
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	topologyConfig := containerlab.Config{}
	if err := yaml.Unmarshal([]byte(topology), &topologyConfig); err != nil {
		return nil, err
	}

	topologyConfig.Name = problemEnvironment.Name

	return yaml.Marshal(topologyConfig)
}

// Check implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Check(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, []ContainerStatus, error) {
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

	containerStatuses := []ContainerStatus{}
	for i := range labData.Containers {
		containerDetail := &labData.Containers[i]

		containerStatuses = append(containerStatuses, ContainerStatus{
			Name:              containerDetail.Name,
			Image:             containerDetail.Image,
			ContainerID:       containerDetail.ContainerID,
			IsReady:           containerDetail.State == "running",
			ManagementAddress: containerDetail.IPv4Address,
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
	log := log.FromContext(ctx)

	client := containerlab.NewContainerLabClientFor(&problemEnvironment)

	topologyFile, err := d.getTopologyFileFor(ctx, reader, &problemEnvironment)
	if err != nil {
		return err
	}

	directoryPath := client.WorkingDirectoryPath()
	log.V(1).Info("ensuring directory for ProblemEnvironment", "path", directoryPath)
	if err := d.ensureDirectory(directoryPath); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	topologyFilePath := client.TopologyFilePath()
	log.V(1).Info("creating topology file", "path", topologyFilePath)
	if _, err := d.createOrUpdateFile(topologyFilePath, []byte(topologyFile)); err != nil {
		return fmt.Errorf("failed to create or update topology file: %w", err)
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
