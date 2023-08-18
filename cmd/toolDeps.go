package cmd

import (
	"os"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/spf13/cobra"
)

var toolDeps = &cobra.Command{
	Use:   "toolDeps",
	Short: "Print dependencies of Tools",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		toolDependencyAnalyser := ToolDependencyAnalyser{logger: logger}

		stepsRootDir := "/Users/godrei/Development/turbolift/bitrise-steps"
		if err := toolDependencyAnalyser.Analyse(stepsRootDir); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(toolDeps)
}

type ToolDependencyAnalyser struct {
	logger log.Logger
}

func (a ToolDependencyAnalyser) Analyse(dir string) error {
	return nil
}
