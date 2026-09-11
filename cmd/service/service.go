package service

import (
	"github.com/Bridgeless-Project/tss-wrapper-svc/cmd/service/migrate"
	"github.com/Bridgeless-Project/tss-wrapper-svc/cmd/service/run"
	"github.com/Bridgeless-Project/tss-wrapper-svc/cmd/utils"
	"github.com/spf13/cobra"
)

func init() {
	registerServiceCommands(Cmd)
	utils.RegisterConfigFlag(Cmd)
}

func registerServiceCommands(cmd *cobra.Command) {
	cmd.AddCommand(migrate.Cmd)
	cmd.AddCommand(run.Cmd)
}

var Cmd = &cobra.Command{
	Use:   "service",
	Short: "Command for running service operations",
}
