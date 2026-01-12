package drivers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

type ContainerLabProblemEnvironmentDriver struct {
	configDir    string
	dockerClient dockerClient.APIClient
}

var _ ProblemEnvironmentDriver = &ContainerLabProblemEnvironmentDriver{}

func NewContainerLabProblemEnvironmentDriver(configDir string, dockerClient dockerClient.APIClient) *ContainerLabProblemEnvironmentDriver {
	return &ContainerLabProblemEnvironmentDriver{
		configDir:    configDir,
		dockerClient: dockerClient,
	}
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

func (d *ContainerLabProblemEnvironmentDriver) fetchFile(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
	fileSource *netconv1alpha1.FileSource,
) ([]byte, error) {
	log := log.FromContext(ctx)

	configMap := corev1.ConfigMap{}
	if err := reader.Get(ctx, types.NamespacedName{
		Namespace: problemEnvironment.Namespace,
		Name:      fileSource.ConfigMapRef.Name,
	}, &configMap); err != nil {
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	data, ok := configMap.Data[fileSource.ConfigMapRef.Key]
	if !ok {
		err := fmt.Errorf(
			"ConfigMap found, but key `%s` missing",
			fileSource.ConfigMapRef.Key,
		)
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	return []byte(data), nil
}

func (d *ContainerLabProblemEnvironmentDriver) getTopologyFileFor(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) ([]byte, error) {
	log := log.FromContext(ctx)

	topology, err := d.fetchFile(ctx, reader, problemEnvironment, &problemEnvironment.Spec.TopologyFile)
	if err != nil {
		log.Error(err, "failed to load topology file")
		return nil, err
	}

	topologyConfig := containerlab.Config{}
	if err := yaml.Unmarshal([]byte(topology), &topologyConfig); err != nil {
		return nil, err
	}

	// enforce requirements for nclet

	// To avoid name conflict, override Prefix and Name
	topologyConfig.Prefix = nil
	topologyConfig.Name = problemEnvironment.Name
	topologyConfig.Mgmt = &containerlab.MgmtNet{
		Network: "nc-mgmt",
	}

	// rewrite filepath forcibly to fill the directory gap
	// TODO: make base directory configurable
	prefix := path.Join(d.configDir, problemEnvironment.Name)
	for _, node := range topologyConfig.Topology.Nodes {
		// rewrite the path of startupConfig source
		if node.StartupConfig != "" {
			node.StartupConfig = path.Join(prefix, node.StartupConfig)
		}

		// rewrite the path of bind source
		for i := range node.Binds {
			parts := strings.Split(node.Binds[i], ":")
			parts[0] = path.Join(node.StartupConfig, parts[0])
			node.Binds[i] = strings.Join(parts, ":")
		}
	}

	return yaml.Marshal(topologyConfig)
}

func (d *ContainerLabProblemEnvironmentDriver) placeFiles(
	ctx context.Context,
	clabClient *containerlab.ContainerLabClient,
	reader client.Reader,
	problemEnvironment *netconv1alpha1.ProblemEnvironment,
) error {
	log := log.FromContext(ctx)

	// ensure working directory
	workingDirectoryPath := clabClient.WorkingDirectoryPath()
	log.V(1).Info("ensuring directory for ProblemEnvironment", "path", workingDirectoryPath)
	if err := d.ensureDirectory(workingDirectoryPath); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// ensure config directory
	configDirectoryPath := path.Join(workingDirectoryPath, "config")
	log.V(1).Info("ensuring directory for ProblemEnvironment", "path", configDirectoryPath)
	if err := d.ensureDirectory(configDirectoryPath); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// place topology file
	topologyFile, err := d.getTopologyFileFor(ctx, reader, problemEnvironment)
	if err != nil {
		return err
	}

	topologyFilePath := clabClient.TopologyFilePath()
	log.V(1).Info("creating topology file", "path", topologyFilePath)
	if _, err := d.createOrUpdateFile(topologyFilePath, []byte(topologyFile)); err != nil {
		return fmt.Errorf("failed to create or update topology file: %w", err)
	}

	// place config file
	for _, config := range problemEnvironment.Spec.ConfigFiles {
		data, err := d.fetchFile(ctx, reader, problemEnvironment, &config)
		if err != nil {
			return err
		}

		configFilePath := path.Join(configDirectoryPath, config.ConfigMapRef.Key)
		if _, err := d.createOrUpdateFile(configFilePath, []byte(data)); err != nil {
			return fmt.Errorf("failed to create or update topology file: %w", err)
		}
	}

	return nil
}

// Check implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Check(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, []netconv1alpha1.ContainerStatus) {
	log := log.FromContext(ctx)

	clabClient := containerlab.NewContainerLabClientFor(&problemEnvironment)

	directoryPath := clabClient.WorkingDirectoryPath()
	if !d.fileExists(directoryPath) {
		// directory not found, ProblemEnvironment haven't been created
		log.V(1).Info("working directory not found, skip to inspect")
		return StatusInit, nil
	}

	topologyFilePath := clabClient.TopologyFilePath()
	if !d.fileExists(topologyFilePath) {
		log.V(1).Info("topology file not found, skip to inspect")
		return StatusError, nil
	}

	containers, err := clabClient.Inspect(ctx)
	if err != nil {
		log.Error(err, "failed to inspect ContainerLab")
		return StatusError, nil
	}

	containerPrefix := fmt.Sprintf("clab-%s-", problemEnvironment.Name)
	containerStatuses := []netconv1alpha1.ContainerStatus{}
	for _, c := range containers {
		name := strings.ReplaceAll(c.Name, containerPrefix, "")
		containerStatus := netconv1alpha1.ContainerStatus{
			Name:                name,
			Image:               c.Image,
			ContainerID:         c.ContainerID,
			ManagementIPAddress: c.IPv4Address,
		}

		containerInfo, err := d.dockerClient.ContainerInspect(ctx, c.ContainerID)
		if err != nil {
			log.Error(err, "failed to fetch container information from docker daemon")
			containerStatuses = append(containerStatuses, containerStatus)
			continue
		}

		ready := false
		if containerInfo.State.Running {
			if containerInfo.State.Health == nil {
				// If containerInfo doesn't have Health, we can consider the container is ready
				ready = true
			} else if containerInfo.State.Health.Status == "healthy" {
				// If Health.Status is "healthy", we can consider the container is ready
				// ref: https://pkg.go.dev/github.com/docker/docker/api/types#Health
				ready = true
			}
		}

		containerStatus.ContainerName = containerInfo.Name
		containerStatus.Ready = ready

		containerStatuses = append(containerStatuses, containerStatus)
	}

	return StatusDeployed, containerStatuses
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	client client.Client,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	clabClient := containerlab.NewContainerLabClientFor(&problemEnvironment)

	if err := d.placeFiles(ctx, clabClient, client, &problemEnvironment); err != nil {
		return err
	}

	// Before deploying ContainerLab, ensure that the management network exists
	if err := d.ensureManagementNetwork(ctx); err != nil {
		return err
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

	if status == StatusDeployed {
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

func (d *ContainerLabProblemEnvironmentDriver) ensureManagementNetwork(ctx context.Context) error {
	name := "nc-mgmt"

	managementNetwork, err := d.dockerClient.NetworkInspect(ctx, name, dockerTypes.NetworkInspectOptions{})
	if err != nil && !dockerClient.IsErrNotFound(err) {
		return err
	}

	if err == nil {
		if d.checkManagementNetwork(managementNetwork) {
			return nil
		}

		// If the management network is not desired, remove and recreate.
		if err := d.dockerClient.NetworkRemove(ctx, name); err != nil {
			return err
		}
	}

	options := dockerTypes.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  "100.64.0.0/10",
					Gateway: "100.64.0.1",
				},
			},
		},
		Options: map[string]string{},
	}

	_, err = d.dockerClient.NetworkCreate(ctx, name, options)
	return err
}

func (d *ContainerLabProblemEnvironmentDriver) checkManagementNetwork(network dockerTypes.NetworkResource) bool {
	if network.Driver != "bridge" {
		return false
	}

	if network.IPAM.Driver != "default" {
		return false
	}

	if len(network.IPAM.Config) != 1 {
		return false
	}

	if network.IPAM.Config[0].Subnet != "100.64.0.0/10" || network.IPAM.Config[0].Gateway != "100.64.0.1" {
		return false
	}

	return true
}
