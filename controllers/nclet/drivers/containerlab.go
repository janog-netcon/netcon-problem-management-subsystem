package drivers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/srl-labs/containerlab/clab"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netconv1alpha1 "github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
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

	topologyConfig := clab.Config{}
	if err := yaml.UnmarshalStrict([]byte(topology), &topologyConfig); err != nil {
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
) (ProblemEnvironmentStatus, error) {
	directoryPath := d.getDirectoryPathFor(&problemEnvironment)
	if !d.fileExists(directoryPath) {
		return StatusDown, nil
	}

	return StatusUp, nil
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	log := log.FromContext(ctx)

	topologyFile, err := d.getTopologyFileFor(ctx, reader, &problemEnvironment)
	if err != nil {
		return err
	}

	directoryPath := d.getDirectoryPathFor(&problemEnvironment)
	log.V(1).Info("ensuring directory for ProblemEnvironment", "path", directoryPath)
	if err := d.ensureDirectory(directoryPath); err != nil {
		log.Error(err, "failed to create directory")
		return err
	}

	topologyFilePath := path.Join(directoryPath, "manifest.yml")
	log.V(1).Info("creating topology file", "path", topologyFilePath)
	if _, err := d.createOrUpdateFile(topologyFilePath, []byte(topologyFile)); err != nil {
		log.Error(err, "failed to create topology file")
		return err
	}

	cmd := exec.CommandContext(ctx, "/usr/bin/clab", "-t", "manifest.yml", "deploy")
	cmd.Dir = directoryPath

	return cmd.Run()
}

// Destroy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	directoryPath := d.getDirectoryPathFor(&problemEnvironment)
	if err := d.ensureDirectory(directoryPath); err != nil {
		return err
	}

	topologyFilePath := path.Join(directoryPath, "manifest.yml")

	if d.fileExists(topologyFilePath) {
		cmd := exec.CommandContext(ctx, "/usr/bin/clab", "-t", "manifest.yml", "destroy")
		cmd.Dir = directoryPath

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	_, err := d.delete(directoryPath)
	return err
}
