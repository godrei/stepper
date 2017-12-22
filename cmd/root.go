package cmd

import (
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/spf13/cobra"
)

// RootCmd ...
var RootCmd = &cobra.Command{
	Use:   "stepper",
	Short: "Solves some Bitrise step / steplib related tasks",
}

// Execute ...
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Errorf(err.Error())
		os.Exit(-1)
	}
}
