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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/term"

	"github.com/janog-netcon/netcon-problem-management-subsystem/internal/tracing"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/containerlab"
)

const internalErrorMessage = "Internal error occured. Please contact NETCON members with Session ID and Trace ID."

func askUserForNode(ctx context.Context, config *containerlab.Config, isAdmin bool) string {
	ctx, span := tracing.Tracer.Start(ctx, "askUserForNode")
	defer span.End()

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
	ctx, span := tracing.Tracer.Start(
		ctx, "accessNode",
		trace.WithAttributes(
			attribute.String("access.node", nodeName),
		),
	)
	defer span.End()

	nodeDefinition, ok := config.Topology.Nodes[nodeName]
	if !ok || nodeDefinition == nil {
		span.AddEvent("specified node doesn't exist")
		fmt.Printf("no such node: \"%s\"\n", nodeName)
		return nil
	}

	// if adminOnly is "true", normal user can't access such node
	if nodeDefinition.Labels[AdminOnlyKey] == "true" && !isAdmin {
		span.AddEvent("specified node is not permitted to access by participants")
		fmt.Printf("no such node: \"%s\"\n", nodeName)
		return nil
	}

	containers, err := client.Inspect(ctx)
	if err != nil {
		tracing.SetError(span, fmt.Errorf("failed to inspect Containerlab: %w", err))
		fmt.Println(internalErrorMessage)
		return nil
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
		tracing.SetError(span, fmt.Errorf("node defined, but not working"))
		fmt.Println(internalErrorMessage)
		return nil
	}

	accessMethod := AccessMethodSSH
	if value, ok := nodeDefinition.Labels[AccessMethodKey]; ok {
		accessMethod = AccessMethod(value)
	}

	if helper := findAccessHelper(accessMethod); helper != nil {
		if err := helper.access(ctx, *nodeDefinition, *containerDetails, isAdmin); err != nil {
			return tracing.WrapError(span, err, "failed to access node")
		}
		return nil
	}

	tracing.SetError(span, fmt.Errorf("no such access method: %s", accessMethod))
	fmt.Println(internalErrorMessage)
	return nil
}

func run(ctx context.Context, topologyFilePath string, isAdmin bool, args []string) error {
	span := trace.SpanFromContext(ctx)

	// access-helper requires stdin bound to terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		tracing.SetError(span, fmt.Errorf("stdin is not bound to terminal"))
		fmt.Println(internalErrorMessage)
		return nil
	}

	topologyFilePath, err := filepath.Abs(topologyFilePath)
	if err != nil {
		tracing.SetError(span, fmt.Errorf("failed to resolve path to the topology file: %w", err))
		fmt.Println(internalErrorMessage)
		return nil
	}

	client := containerlab.NewContainerLabClient(topologyFilePath)
	config, err := client.LoadTopologyFile()
	if err != nil {
		tracing.SetError(span, fmt.Errorf("failed to load topology file: %w", err))
		fmt.Println(internalErrorMessage)
		return nil
	}

	if len(args) == 1 { // if nodeName is specified
		if err := accessNode(ctx, client, config, args[0], isAdmin); err != nil {
			tracing.SetError(span, fmt.Errorf("failed to access node: %w", err))
			fmt.Println(internalErrorMessage)
			return nil
		}
	} else {
		for {
			nodeName := askUserForNode(ctx, config, isAdmin)
			if nodeName == "" {
				return nil
			}
			if err := accessNode(ctx, client, config, nodeName, isAdmin); err != nil {
				tracing.SetError(span, fmt.Errorf("failed to access node: %w", err))
				fmt.Println(internalErrorMessage)
			}
		}
	}

	return nil
}

func main() {
	var (
		topologyFilePath    string
		isAdmin             bool
		enableOpenTelemetry bool
	)

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

			if enableOpenTelemetry {
				shutdown, err := tracing.SetupOpenTelemetry(ctx, "access-helper")
				if err != nil {
					return fmt.Errorf("failed to setup OpenTelemetry: %w", err)
				}
				defer shutdown(ctx)

				carrier := tracing.EnvCarrier{}
				ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
			}

			ctx, span := tracing.Tracer.Start(
				ctx, "Run",
				trace.WithAttributes(
					attribute.String("access.topology.path", topologyFilePath),
					attribute.Bool("access.admin", isAdmin),
				),
			)
			defer span.End()

			if err := run(ctx, topologyFilePath, isAdmin, args); err != nil {
				span.SetStatus(codes.Error, "failed to run access-helper")
				span.RecordError(err)
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&topologyFilePath, "topo", "t", "", "path to the topology file")
	cmd.PersistentFlags().BoolVar(&isAdmin, "admin", false, "whether access user is admin or not")
	cmd.PersistentFlags().BoolVar(&enableOpenTelemetry, "enable-otel", false, "enable OpenTelemetry")

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
