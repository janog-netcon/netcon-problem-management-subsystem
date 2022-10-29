package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

func f(c *containerlab.Config) (string, error) {
	choices := []string{}

	for key, _ := range c.Topology.Nodes {
		// TODO: implement the way to skip selection
		choices = append(choices, key)
	}

	whitespaces := 0
	if len(choices) != 0 {
		whitespaces = int(math.Log10(float64(len(choices))))
	}

	fmt.Println("Node List:")
	for i, name := range choices {
		fmt.Printf(" %*x: %s\n", whitespaces, i, name)
	}

	for {
		var choice int
		fmt.Printf("your choice: ")
		fmt.Scanf("%x", &choice)

		if 0 <= choice && choice < len(choices) {
			return choices[choice], nil
		}

		fmt.Println("invalid input")
	}
}

func accessToNode(
	client *containerlab.ContainerLabClient,
	config *containerlab.Config,
	nodeName string,
) error {
	labData, err := client.Inspect(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to inspect ContainerLab: %w", err)
	}

	var containerDetails *containerlab.ContainerDetails = nil

	containerName := fmt.Sprintf("clab-%s-%s", config.Name, nodeName)
	for i := range labData.Containers {
		container := &labData.Containers[i]
		if container.Name == containerName {
			containerDetails = container
			break
		}
	}

	if containerDetails == nil {
		return errors.New("node defined, but not working")
	}

	// TODO: implement the way to access node via SSH with custom label
	// TODO: implement the way to change executable with custom label
	cmd := exec.Command("docker", "exec", "-it", containerName, "sh")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to start pty")
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				// TODO: ...
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}

func main() {
	var topologyFilePath string

	cmd := cobra.Command{
		Use: path.Base(os.Args[0]),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("invalid argument")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			topologyFilePath, err := filepath.Abs(topologyFilePath)
			if err != nil {
				fmt.Printf("failed to resolve path to the topology file: %v", err)
				return nil
			}

			client := containerlab.NewContainerLabClient(topologyFilePath)
			config, err := client.LoadTopologyFile()
			if err != nil {
				fmt.Printf("failed to load topology file: %v\n", err)
				return nil
			}

			var nodeName string
			if len(args) == 1 {
				nodeName = args[0]
			} else {
				selected, err := f(config)
				if err != nil {
					return err
				}
				nodeName = selected
			}

			_, ok := config.Topology.Nodes[nodeName]
			if !ok {
				fmt.Printf("No such node: \"%s\"", nodeName)
				return nil
			}

			return accessToNode(client, config, nodeName)
		},
	}

	cmd.PersistentFlags().StringVarP(&topologyFilePath, "topo", "t", "", "path to the topology file")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
