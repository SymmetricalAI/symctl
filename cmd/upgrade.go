package cmd

import (
	"github.com/SymmetricalAI/symctl/internal/logger"
	"github.com/SymmetricalAI/symctl/internal/upgrader"

	"github.com/spf13/cobra"
)

var (
	upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			logger.Debugf("upgrade Run called")
			logger.Debugf("upgrade args: %v\n", args)
			logger.Debugf("dry-run: %v\n", dryRun)
			upgrader.Upgrade(dryRun)
		},
	}
	dryRun bool
)

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().SetInterspersed(false)
	upgradeCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "dry run mode")
}
