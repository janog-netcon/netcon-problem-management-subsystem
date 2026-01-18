package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/janog-netcon/netcon-problem-management-subsystem/api/v1alpha1"
	clientset "github.com/janog-netcon/netcon-problem-management-subsystem/pkg/clientset/v1alpha1"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/kubectl-netcon/deploylog"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/printers"
	"github.com/janog-netcon/netcon-problem-management-subsystem/pkg/util"
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
	cmd.AddCommand(newProblemEnvironmentDeleteCmd())
	cmd.AddCommand(newProblemEnvironmentAssignCmd())
	cmd.AddCommand(newProblemEnvironmentUnassignCmd())
	cmd.AddCommand(newProblemEnvironmentShowDeployLogCmd())
	cmd.AddCommand(newProblemEnvironmentSSHCmd())

	return cmd
}

func newProblemEnvironmentListCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
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

			client := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace)

			problemEnvironmentList, err := client.List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			printOptions := printers.PrintOptions{}
			if verbose {
				printOptions.Wide = true
			}

			printers.PrintObject(
				os.Stdout,
				problemEnvironmentList,
				printOptions,
			)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show more verbose log")

	return cmd
}

func newProblemEnvironmentDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:          "delete",
		Short:        "Delete ProblemEnvironment",
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

			client := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace)

			problemEnvironment, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if !force && util.GetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionAssigned,
			) == metav1.ConditionTrue {
				return errors.New("failed to delete: ProblemEnvironment is already assigned, please unassign before deleting")
			}

			propagationPolicy := metav1.DeletePropagationForeground
			if err := client.Delete(ctx, name, metav1.DeleteOptions{
				PropagationPolicy: &propagationPolicy,
			}); err != nil {
				return err
			}

			fmt.Printf("ProblemEnvironment \"%s\" deleted\n", name)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete ProblemEnvironment even if it is already assigned")

	return cmd
}

func newProblemEnvironmentAssignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "assign",
		Short:        "Assign ProblemEnvironment for debug purpose",
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

			client := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace)

			problemEnvironment, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionReady,
			) != metav1.ConditionTrue {
				return errors.New("failed to update status: ProblemEnvironment is not ready")
			}

			if util.GetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionAssigned,
			) != metav1.ConditionFalse {
				return errors.New("failed to update status: ProblemEnvironment is already assigned")
			}

			util.SetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionAssigned,
				metav1.ConditionTrue,
				"AdminUpdated",
				"assigned by admin forcibly",
			)

			if _, err := client.UpdateStatus(ctx, problemEnvironment, metav1.UpdateOptions{}); err != nil {
				return err
			}

			fmt.Printf("ProblemEnvironment \"%s\" assigned\n", name)

			return nil
		},
	}

	return cmd
}

func newProblemEnvironmentUnassignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "unassign",
		Short:        "Unassign ProblemEnvironment for debug purpose",
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

			client := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace)

			problemEnvironment, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if util.GetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionAssigned,
			) != metav1.ConditionTrue {
				return errors.New("failed to update status: ProblemEnvironment is not assigned")
			}

			util.SetProblemEnvironmentCondition(
				problemEnvironment,
				v1alpha1.ProblemEnvironmentConditionAssigned,
				metav1.ConditionFalse,
				"AdminUpdated",
				"assigned by admin forcibly",
			)

			if _, err := client.UpdateStatus(ctx, problemEnvironment, metav1.UpdateOptions{}); err != nil {
				return err
			}

			fmt.Printf("ProblemEnvironment \"%s\" unassigned\n", name)

			return nil
		},
	}

	return cmd
}

func newProblemEnvironmentShowDeployLogCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:          "show-deploy-log",
		Short:        "Show deploy log for given ProblemEnvironment",
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

			problemEnvironment, err := problemEnvironmentClient.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
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

				break
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show more verbose log")

	return cmd
}

func newProblemEnvironmentSSHCmd() *cobra.Command {
	var admin bool

	cmd := &cobra.Command{
		Use:          "ssh",
		Short:        "SSH into ProblemEnvironment",
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

			client := clientset.ProblemEnvironment(*globalConfig.configFlags.Namespace)

			problemEnvironment, err := client.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if problemEnvironment.Spec.WorkerName == "" {
				return errors.New("ProblemEnvironment is not scheduled yet")
			}

			workerClient := clientset.Worker()
			worker, err := workerClient.Get(ctx, problemEnvironment.Spec.WorkerName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			externalIP := worker.Status.WorkerInfo.ExternalIPAddress
			externalPort := worker.Status.WorkerInfo.ExternalPort

			if externalIP == "" || externalPort == 0 {
				return errors.New("Worker is not ready yet")
			}

			user := fmt.Sprintf("nc_%s", problemEnvironment.Name)
			if admin {
				user = fmt.Sprintf("ncadmin_%s", problemEnvironment.Name)
			} else {
				fmt.Printf("Password: %s\n", problemEnvironment.Status.Password)
			}

			sshCmd := exec.CommandContext(ctx, "ssh",
				"-p", fmt.Sprintf("%d", externalPort),
				fmt.Sprintf("%s@%s", user, externalIP),
			)
			sshCmd.Stdin = os.Stdin
			sshCmd.Stdout = os.Stdout
			sshCmd.Stderr = os.Stderr

			if err := sshCmd.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&admin, "admin", false, "Login as admin")

	return cmd
}
