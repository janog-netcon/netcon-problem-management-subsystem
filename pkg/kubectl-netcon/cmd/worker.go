package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	clientset "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/clientset/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/printers"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func newWorkerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Worker",
		Aliases: []string{
			"w",
		},
		SilenceUsage: true,
	}

	cmd.AddCommand(newWorkerListCmd())
	cmd.AddCommand(newWorkerEnableCmd())
	cmd.AddCommand(newWorkerDisableCmd())

	return cmd
}

func newWorkerListCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List Worker",
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

			client := clientset.Worker()

			workerList, err := client.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			printOptions := printers.PrintOptions{}
			if verbose {
				printOptions.Wide = true
			}

			printers.PrintObject(
				os.Stdout,
				workerList,
				printOptions,
			)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show more verbose log")

	return cmd
}
func newWorkerEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "enable",
		Short:        "Enable Worker",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			v1alpha1.AddToScheme(scheme.Scheme)

			config, err := globalConfig.configFlags.ToRESTConfig()
			if err != nil {
				return err
			}

			clientset, err := clientset.NewForConfig(config)
			if err != nil {
				return err
			}

			client := clientset.Worker()

			worker, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if !worker.Spec.DisableSchedule {
				return errors.New("failed to enable Worker: Worker is already enabled")
			}

			worker.Spec.DisableSchedule = false

			if _, err := client.Update(ctx, worker, metav1.UpdateOptions{}); err != nil {
				return err
			}

			fmt.Printf("Worker \"%s\" enabled\n", name)

			return nil
		},
	}

	return cmd
}

func newWorkerDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "disable",
		Short:        "Disable Worker",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			v1alpha1.AddToScheme(scheme.Scheme)

			config, err := globalConfig.configFlags.ToRESTConfig()
			if err != nil {
				return err
			}

			clientset, err := clientset.NewForConfig(config)
			if err != nil {
				return err
			}

			client := clientset.Worker()

			worker, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if worker.Spec.DisableSchedule {
				return errors.New("failed to disable Worker: Worker is already disabled")
			}

			worker.Spec.DisableSchedule = true

			if _, err := client.Update(ctx, worker, metav1.UpdateOptions{}); err != nil {
				return err
			}

			fmt.Printf("Worker \"%s\" disabled\n", name)

			return nil
		},
	}

	return cmd
}
