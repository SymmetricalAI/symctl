package cmd

import (
	"fmt"
	"github.com/SymmetricalAI/symctl/internal/installer"
	"github.com/SymmetricalAI/symctl/internal/logger"
	"github.com/spf13/cobra"
	"net/url"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: func(cmd *cobra.Command, args []string) error {
		logger.Debugf("install Args called")
		if len(args) < 1 {
			return cobra.ExactArgs(1)(cmd, args)
		}
		_, err := url.ParseRequestURI(args[0])
		if err != nil {
			return fmt.Errorf("invalid URL: %s", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debugf("install Run called")
		logger.Debugf("install args: %v\n", args)
		installer.Install(args[0])
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().SetInterspersed(false)
}
