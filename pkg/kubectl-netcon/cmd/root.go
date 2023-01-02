package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var globalConfig struct {
	configFlags *genericclioptions.ConfigFlags
}

func stringPtr(v string) *string {
	return &v
}

func NewRootCmd() *cobra.Command {
	globalConfig.configFlags = genericclioptions.NewConfigFlags(true)
	globalConfig.configFlags.Namespace = stringPtr("netcon")

	cmd := &cobra.Command{
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	globalConfig.configFlags.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(newProblemEnvironmentCmd())

	return cmd
}
