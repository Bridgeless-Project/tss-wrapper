package cmd

import (
	"os"

	"github.com/Bridgeless-Project/tss-wrapper-svc/cmd/service"
	"github.com/spf13/cobra"
)

func Execute() {
	root := &cobra.Command{
		Use:   "tss-wrapper-svc",
		Short: "TSS wrapper service",
	}

	root.AddCommand(service.Cmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
