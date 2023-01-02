package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	clientset "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/clientset/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/kubectl-netcon/deploylog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func newProblemEnvironmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "problem-environment",
		Short: "ProblemEnvironment",
		Aliases: []string{
			"pe",
			"probenv",
		},
		SilenceUsage: true,
	}

	cmd.AddCommand(newProblemEnvironmentListCmd())
	cmd.AddCommand(newProblemEnvironmentShowDeployLogCmd())

	return cmd
}

func newProblemEnvironmentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "List ProblemEnvironment",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			v1alpha1.AddToScheme(scheme.Scheme)

			config, err := globalConfig.configFlags.ToRESTConfig()
			if err != nil {
				return err
			}

			clientset, err := clientset.NewForConfig(config)
			if err != nil {
				return err
			}

			list, err := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			for _, d := range list.Items {
				fmt.Println(d.Name)
			}

			return nil
		},
	}
}

func newProblemEnvironmentShowDeployLogCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:          "show-deploy-log",
		Short:        "Show deploy log for given ProblemEnvironment",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return nil
			}

			ctx := cmd.Context()

			v1alpha1.AddToScheme(scheme.Scheme)

			config, err := globalConfig.configFlags.ToRESTConfig()
			if err != nil {
				return err
			}

			namespace := *globalConfig.configFlags.Namespace

			configMapClient, err := newConfigMapClientForConfig(config, namespace)
			if err != nil {
				return err
			}

			problemEnvironmentClient, err := newProblemEnvironmentClientForConfig(config, namespace)
			if err != nil {
				return err
			}

			jst, err := time.LoadLocation("Asia/Tokyo")
			if err != nil {
				return err
			}

			parser := deploylog.DeployLogParser{}
			printer := deploylog.NewPrettyDeployLogPrinterWithLocation(os.Stdout, jst)

			if verbose {
				printer.SetLevel(deploylog.LogLevelDebug)
			}

			configMapList, err := configMapClient.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			problemEnvironmentList, err := problemEnvironmentClient.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			problemEnvironmentName := args[0]

			for _, problemEnvironment := range problemEnvironmentList.Items {
				if problemEnvironment.Name != problemEnvironmentName {
					continue
				}

				for _, configMap := range configMapList.Items {
					if !strings.HasPrefix(configMap.Name, "deploy-"+problemEnvironment.Name) {
						continue
					}

					stderr, ok := configMap.Data["stderr"]
					if !ok {
						continue
					}

					log, err := parser.Parse([]byte(stderr))
					if err != nil {
						return err
					}

					if err := printer.Print(log); err != nil {
						return err
					}

					return nil
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show more verbose log")

	return cmd
}
