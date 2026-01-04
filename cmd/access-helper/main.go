package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

func askUserForNode(config *containerlab.Config, isAdmin bool) string {
	nodeNames := []string{}
	for nodeName, nodeDefinition := range config.Topology.Nodes {
		if nodeDefinition.Labels[AdminOnlyKey] == "true" && !isAdmin {
			continue
		}
		nodeNames = append(nodeNames, nodeName)
	}

	sort.Strings(nodeNames)

	fmt.Println("Enter the number of the node you want to access")

	fmt.Println("Nodes:")
	for i := 0; i < len(nodeNames); i++ {
		fmt.Printf("   %3d: %s\n", i+1, nodeNames[i])
	}
	fmt.Println("     0: (exit)")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Your select: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			// exit if Ctrl+D entered
			if errors.Is(err, io.EOF) {
				return ""
			}

			fmt.Println("Please input collect value.")
			continue
		}

		// Trim rightmost "\n" for easy handling
		input = strings.TrimRight(input, "\n")

		switch input {
		case "":
			// if input is empty, continue
			continue

		case "exit":
			// Normally, "0" should be entered to exit.
			// However, if the user enters "exit", it will be treated as "0".
			return ""
		}

		selected, err := strconv.ParseInt(strings.TrimRight(input, "\n"), 10, 64)
		if err != nil {
			fmt.Println("Please input collect value.")
			continue
		}
		if selected == 0 {
			return ""
		}

		if !(1 <= selected && int(selected) <= len(nodeNames)) {
			fmt.Println("Please input collect value.")
			continue
		}

		return nodeNames[selected-1]
	}
}

func accessNode(
	ctx context.Context,
	client *containerlab.ContainerLabClient,
	config *containerlab.Config,
	nodeName string,
	isAdmin bool,
) error {
	nodeDefinition, ok := config.Topology.Nodes[nodeName]
	if !ok || nodeDefinition == nil {
		fmt.Printf("no such node: \"%s\"", nodeName)
		return nil
	}

	// if adminOnly is "true", normal user can't access such node
	if nodeDefinition.Labels[AdminOnlyKey] == "true" && !isAdmin {
		fmt.Printf("no such node: \"%s\"", nodeName)
		return nil
	}

	containers, err := client.Inspect(ctx)
	if err != nil {
		return fmt.Errorf("failed to inspect ContainerLab: %w", err)
	}

	var containerDetails *containerlab.ContainerDetails = nil

	// TODO: we need to implement the same logic in containerlab
	prefix := ""
	if config.Prefix != nil {
		if *config.Prefix != "" {
			prefix = fmt.Sprintf("%s%s-", *config.Prefix, config.Name)
		}
	} else {
		prefix = fmt.Sprintf("clab-%s-", config.Name)
	}
	containerName := fmt.Sprintf("%s%s", prefix, nodeName)

	for i, c := range containers {
		if c.Name == containerName {
			containerDetails = &containers[i]
			break
		}
	}

	if containerDetails == nil {
		return errors.New("node defined, but not working")
	}

	accessMethod := AccessMethodSSH
	if value, ok := nodeDefinition.Labels[AccessMethodKey]; ok {
		accessMethod = AccessMethod(value)
	}

	if helper := findAccessHelper(accessMethod); helper != nil {
		return helper.access(ctx, *nodeDefinition, *containerDetails, isAdmin)
	}

	return errors.New(fmt.Sprintf("no such method: %s", accessMethod))
}

func main() {
	var topologyFilePath string
	var isAdmin bool

	cmd := cobra.Command{
		Use: path.Base(os.Args[0]),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("invalid argument")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// access-helper requires stdin bound to terminal
			if !term.IsTerminal(0) {
				return errors.New("stdin is not bound to terminal")
			}

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

			if len(args) == 1 { // if nodeName is specified
				if err := accessNode(ctx, client, config, args[0], isAdmin); err != nil {
					fmt.Printf("failed to access node: %v\n", err)
				}
			} else {
				for {
					nodeName := askUserForNode(config, isAdmin)
					if nodeName == "" {
						return nil
					}
					if err := accessNode(ctx, client, config, nodeName, isAdmin); err != nil {
						fmt.Printf("failed to access node: %v\n", err)
					}
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&topologyFilePath, "topo", "t", "", "path to the topology file")
	cmd.PersistentFlags().BoolVar(&isAdmin, "admin", false, "whether access user is admin or not")

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
