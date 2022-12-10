package drivers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerDetailStatus) {
	log := log.FromContext(ctx)

	clabClient := containerlab.NewContainerLabClientFor(&problemEnvironment)

	directoryPath := clabClient.WorkingDirectoryPath()
	if !d.fileExists(directoryPath) {
		// directory not found, ProblemEnvironment haven't been created
		log.V(1).Info("working directory not found, skip to inspect")
		return StatusNotReady, nil
	}

	topologyFilePath := clabClient.TopologyFilePath()
	if !d.fileExists(topologyFilePath) {
		log.V(1).Info("topology file not found, skip to inspect")
		return StatusNotReady, nil
	}

	labData, err := clabClient.Inspect(ctx)
	if err != nil {
		log.Error(err, "failed to inspect ContainerLab")
		return StatusNotReady, nil
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

	return StatusReady, containerStatuses
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	log := log.FromContext(ctx)

	clabClient := containerlab.NewContainerLabClientFor(&problemEnvironment)

	topologyFile, err := d.getTopologyFileFor(ctx, client, &problemEnvironment)
	if err != nil {
		return err
	}

	directoryPath := clabClient.WorkingDirectoryPath()
	log.V(1).Info("ensuring directory for ProblemEnvironment", "path", directoryPath)
	if err := d.ensureDirectory(directoryPath); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	topologyFilePath := clabClient.TopologyFilePath()
	log.V(1).Info("creating topology file", "path", topologyFilePath)
	if _, err := d.createOrUpdateFile(topologyFilePath, []byte(topologyFile)); err != nil {
		return fmt.Errorf("failed to create or update topology file: %w", err)
	}

	return d.deploy(ctx, client, clabClient, &problemEnvironment)
}

func (d *ContainerLabProblemEnvironmentDriver) deploy(
	ctx context.Context,
	client client.Client,
	clabClient *containerlab.ContainerLabClient,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) error {
	log := log.FromContext(ctx)

	startedAt := time.Now()
	stdout, stderr, err := clabClient.DeployWithOutput(ctx)
	endedAt := time.Now()

	if err != nil {
		log.Error(err, "finished deploying", "elapsed", endedAt.Sub(startedAt))
	} else {
		log.V(1).Info("finished deploying", "elapsed", endedAt.Sub(startedAt))
	}

	configMap := corev1.ConfigMap{}
	configMap.Namespace = problemEnvironment.Namespace
	configMap.Name = fmt.Sprintf("deploy-%s-%d", problemEnvironment.Name, startedAt.Unix())
	configMap.Data = map[string]string{
		"stdout":    string(stdout),
		"stderr":    string(stderr),
		"startedAt": startedAt.Format(time.RFC3339Nano),
		"endedAt":   endedAt.Format(time.RFC3339Nano),
	}
	controllerutil.SetOwnerReference(problemEnvironment, &configMap, client.Scheme())

	if err := client.Create(ctx, &configMap); err != nil {
		log.Info("failed to record deploy log")
	}

	return err
}

// Destroy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	clabClient := containerlab.NewContainerLabClientFor(&problemEnvironment)

	status, _ := d.Check(ctx, client, problemEnvironment)

	if status == StatusReady {
		if err := clabClient.Destroy(ctx); err != nil {
			return fmt.Errorf("failed to destroy ContainerLab: %w", err)
		}
	}

	directoryPath := clabClient.WorkingDirectoryPath()
	if _, err := d.delete(directoryPath); err != nil {
		return fmt.Errorf("failed to delete directory for ProblemEnvironment: %w", err)
	}

	return nil
}
