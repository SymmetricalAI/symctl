package cmd

import (
	"fmt"
	"github.com/SymmetricalAI/symctl/internal/executor"
	"github.com/SymmetricalAI/symctl/internal/installer"
	"github.com/SymmetricalAI/symctl/internal/logger"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "symctl",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if talkative {
				logger.Debug = true
			}
			logger.Debugf("root PersistentPreRun called")
			logger.Debugf("root args: %v\n", args)
			logger.Debugf("Whole command-line: %v\n", os.Args)
		},
		Args:    cobra.MinimumNArgs(1),
		Version: version,
	}
	talkative bool
	version   = "0.0.0"
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().SetInterspersed(false)
	rootCmd.PersistentFlags().BoolVarP(&talkative, "talkative", "t", false, "talkative messages")

	binaryDir, err := installer.GetInstallDir()
	if err != nil {
		logger.Fatalf("Error getting binary directory: %s\n", err)
	}
	plugins, err := executor.ListPlugins(binaryDir)
	if err != nil {
		logger.Fatalf("Error listing plugins: %s\n", err)
	}
	for _, plugin := range plugins {
		cmd := &cobra.Command{
			Use:   plugin,
			Short: fmt.Sprintf("%s plugin subcommands", plugin),
			Run: func(cmd *cobra.Command, args []string) {
				logger.Debugf("plugin Run called for %s", plugin)
				executor.Execute(plugin, args)
			},
		}
		cmd.Flags().SetInterspersed(false)
		rootCmd.AddCommand(cmd)
	}
}
