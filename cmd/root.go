package cmd

import (
	"github.com/SymmetricalAI/symctl/internal/executor"
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
		Args: func(cmd *cobra.Command, args []string) error {
			logger.Debugf("root Args called")
			if len(args) < 1 {
				return cobra.MinimumNArgs(1)(cmd, args)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Debugf("root Run called")
			executor.Execute(args[0], args[1:])
		},
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
}
