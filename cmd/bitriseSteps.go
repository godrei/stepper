package cmd

import (
	"github.com/bitrise-io/stepman/models"
	"github.com/godrei/stepper/tools"
	"github.com/spf13/cobra"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
)

var bitriseStepsCmd = &cobra.Command{
	Use:   "bitriseSteps",
	Short: "Lists Bitrise steps.",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		stepLister := StepLister{logger: logger}
		if err := stepLister.ListSteps(&ListOptions{RepoURLFilters: []string{"https://github.com/bitrise-steplib", "https://github.com/bitrise-io"}}); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(bitriseStepsCmd)
}

type StepLister struct {
	logger log.Logger
}

type ListOptions struct {
	RepoURLFilters []string
}

type Step struct {
	models.StepModel
	StepID string
}

func (l StepLister) ListSteps(opts *ListOptions) error {
	if err := tools.StepmanUpdate(defaultSteplibURI); err != nil {
		return err
	}

	steplib, err := tools.StepmanExportSpec(defaultSteplibURI, tools.ExportTypesLatest)
	if err != nil {
		return err
	}

	var steps []Step
	for stepID, stepGroup := range steplib.Steps {
		for _, step := range stepGroup.Versions {
			if step.Source == nil {
				l.logger.Warnf("step without source: %s", stepID)
				break
			}

			if opts != nil {
				matches := false
				for _, filter := range opts.RepoURLFilters {
					if strings.Contains(step.Source.Git, filter) {
						matches = true
						break
					}
				}

				if matches {
					steps = append(steps, Step{StepModel: step, StepID: stepID})
				}
			}
		}
	}

	for idx, step := range steps {
		l.logger.Printf("%d. %s: %s", idx+1, step.StepID, step.Source.Git)
	}

	return nil
}
