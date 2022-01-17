package cmd

import (
	"github.com/makkes/fluxmc/cmd/bootstrap"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.AddCommand(bootstrap.NewCommand())

	return cmd
}
