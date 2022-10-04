package drivers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

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
	return true, os.Remove(path)
}

// Check implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Check(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) (ProblemEnvironmentStatus, error) {
	rootDirectoryPath := path.Join("data", "topologies", problemEnvironment.Name)
	if err := d.ensureDirectory(rootDirectoryPath); err != nil {
		return StatusUnknown, err
	}

	if !d.fileExists(rootDirectoryPath) {
		return StatusDown, nil
	}

	return StatusDown, nil
}

// Deploy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Deploy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	log := log.FromContext(ctx)

	configMap := corev1.ConfigMap{}
	if err := reader.Get(ctx, types.NamespacedName{
		Namespace: problemEnvironment.Namespace,
		Name:      problemEnvironment.Spec.TopologyFile.ConfigMapRef.Name,
	}, &configMap); err != nil {
		return err
	}

	topologyFile, ok := configMap.Data[problemEnvironment.Spec.TopologyFile.ConfigMapRef.Key]
	if !ok {
		err := fmt.Errorf(
			"ConfigMap found, but key `%s` missing",
			problemEnvironment.Spec.TopologyFile.ConfigMapRef.Key,
		)
		log.Error(err, "failed to deploy")
		return err
	}

	rootDirectoryPath := path.Join("data", problemEnvironment.Name)
	if err := d.ensureDirectory(rootDirectoryPath); err != nil {
		return err
	}

	topologyFilePath := path.Join(rootDirectoryPath, "manifest.yaml")
	if _, err := d.createOrUpdateFile(topologyFilePath, []byte(topologyFile)); err != nil {
		return err
	}

	return nil
}

// Destroy implements ProblemEnvironmentDriver
func (d *ContainerLabProblemEnvironmentDriver) Destroy(
	ctx context.Context,
	reader client.Reader,
	problemEnvironment netconv1alpha1.ProblemEnvironment,
) error {
	rootDirectoryPath := path.Join("data", "topologies", problemEnvironment.Name)
	if err := d.ensureDirectory(rootDirectoryPath); err != nil {
		return err
	}

	_, err := d.delete(rootDirectoryPath)
	return err
}
