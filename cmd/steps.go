package cmd

import (
	"bytes"
	"os"
	"strings"
	"text/template"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/stepman/models"
	"github.com/godrei/stepper/tools"
	giturl "github.com/kubescape/go-git-url"
	"github.com/spf13/cobra"
)

const defaultPrintTemplate = "{{range $i, $step := .}}{{$i}},{{$step.StepID}}\n{{end}}"

var bitriseStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "Lists steps from the Bitrise StepLib.",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		stepLister := StepLister{logger: logger}

		var repoURLFilters []string
		if repoURLFilterFlag != "" {
			repoURLFilters = strings.Split(repoURLFilterFlag, ",")
		}
		printTemplate := printTemplateFlag

		if err := stepLister.ListSteps(&ListOptions{
			//RepoURLFilters: []string{"https://github.com/bitrise-steplib", "https://github.com/bitrise-io"},
			RepoURLFilters: repoURLFilters,
			PrintTemplate:  printTemplate,
			//PrintTemplate:  "{{range $i, $step := .}}{{$step.Repository.Owner}}/{{$step.Repository.Repo}}\n{{end}}",
		}); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

var (
	repoURLFilterFlag string
	printTemplateFlag string
)

func init() {
	RootCmd.AddCommand(bitriseStepsCmd)
	bitriseStepsCmd.Flags().StringVarP(&repoURLFilterFlag, "repo-url-filter", "", "", "List of repo URL filters, separated by a comma character. Filters are compared to the repository URL with strings.Contains.")
	bitriseStepsCmd.Flags().StringVarP(&printTemplateFlag, "print-template", "", "", "Template for printing the list of steps. The template is executed on the []Step.")
}

type StepLister struct {
	logger log.Logger
}

type ListOptions struct {
	RepoURLFilters []string
	PrintTemplate  string
}

type StepRepository struct {
	Host  string
	Owner string
	Repo  string
}

type Step struct {
	models.StepModel
	StepID     string
	Repository StepRepository
}

func (l StepLister) ListSteps(opts *ListOptions) error {
	steplib, err := l.getStpLibSpec()
	if err != nil {
		return err
	}

	steps, err := l.listSteps(steplib, opts)
	if err != nil {
		return err
	}

	tmpl := ""
	if opts != nil {
		tmpl = opts.PrintTemplate
	}
	if tmpl == "" {
		tmpl = defaultPrintTemplate
	}
	if err := l.printSteps(steps, tmpl); err != nil {
		return err
	}

	return nil
}

func (l StepLister) getStpLibSpec() (models.StepCollectionModel, error) {
	if err := tools.StepmanUpdate(defaultSteplibURI); err != nil {
		return models.StepCollectionModel{}, err
	}

	steplib, err := tools.StepmanExportSpec(defaultSteplibURI, tools.ExportTypesLatest)
	if err != nil {
		return models.StepCollectionModel{}, err
	}

	return steplib, nil
}

func (l StepLister) listSteps(steplib models.StepCollectionModel, opts *ListOptions) ([]Step, error) {
	var steps []Step
	for stepID, stepGroup := range steplib.Steps {
		for _, step := range stepGroup.Versions {
			if step.Source == nil {
				l.logger.Warnf("step without source: %s", stepID)
				break
			}

			matches := true

			if opts != nil && len(opts.RepoURLFilters) > 0 {
				matches = false

				for _, filter := range opts.RepoURLFilters {
					if strings.Contains(step.Source.Git, filter) {
						matches = true
						break
					}
				}
			}

			if matches {
				gitURL, err := giturl.NewGitURL(step.Source.Git)
				if err != nil {
					return nil, err
				}

				steps = append(steps, Step{
					StepModel: models.StepModel{},
					StepID:    stepID,
					Repository: StepRepository{
						Host:  gitURL.GetHostName(),
						Owner: gitURL.GetOwnerName(),
						Repo:  gitURL.GetRepoName(),
					},
				})
			}
		}
	}

	return steps, nil
}

func (l StepLister) printSteps(steps []Step, tmpl string) error {
	t := template.New("steps")
	t, err := t.Parse(tmpl)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, steps); err != nil {
		return err
	}

	l.logger.Printf(buff.String())
	return nil
}
