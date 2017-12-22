package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/bitrise-io/go-utils/log"
	"github.com/godrei/stepper/tools"
	"github.com/google/go-github/github"
	ver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	defaultSteplibURI     = "https://github.com/bitrise-io/bitrise-steplib.git"
	lastReleaseTimeLayout = "2006-01-02"
)

var (
	flagGithubAPIToken string
	flagStartTime      string
)

var stepChangesCmd = &cobra.Command{
	Use:   "stepChanges",
	Short: "Collects step changes from the given time to now in markdown ready format.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := stepChanges(); err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func getRelease(ctx context.Context, client *github.Client, repoOwner, repoName, tag string) (*github.RepositoryRelease, error) {
	release, response, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag)
	if response.StatusCode < http.StatusOK || response.StatusCode > http.StatusMultipleChoices {
		return release, nil
	}
	return release, err
}

func findLatestVersion(versions []string) (string, error) {
	var latestVersion *ver.Version
	for _, version := range versions {
		v, err := ver.NewVersion(version)
		if err != nil {
			return "", err
		}

		if latestVersion == nil || latestVersion.LessThan(v) {
			latestVersion = v
		}
	}

	if latestVersion == nil {
		return "", errors.New("failed to find latest version")
	}

	return latestVersion.String(), nil
}

func lowerCharacterFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func normalizeReleaseLine(line string) string {
	trimmed := strings.TrimPrefix(line, "*")
	trimmed = strings.TrimPrefix(trimmed, "-")
	trimmed = strings.TrimSuffix(trimmed, ".")
	trimmed = strings.TrimSpace(trimmed)

	if trimmed == "" {
		return ""
	}

	trimmed = lowerCharacterFirst(trimmed)

	return "- " + trimmed
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func stepChanges() error {
	if flagGithubAPIToken == "" {
		flagGithubAPIToken = os.Getenv("STEPPER_GITHUB_API_TOKEN")
	}

	if flagGithubAPIToken == "" {
		return fmt.Errorf("api-token not defined")
	}

	if flagStartTime == "" {
		return fmt.Errorf("start not defined")
	}

	// Collect new & updated step repos
	if err := tools.StepmanUpdate(defaultSteplibURI); err != nil {
		return err
	}

	steplib, err := tools.StepmanExportSpec(defaultSteplibURI, tools.ExportTypesFull)
	if err != nil {
		return err
	}

	lastCheckTime, err := time.Parse(lastReleaseTimeLayout, flagStartTime)
	if err != nil {
		return err
	}

	updatedSteps := map[string]map[string]string{}
	newSteps := map[string]map[string]string{}

	for stepID, stepGroup := range steplib.Steps {
		isFirstVersion := len(stepGroup.Versions) == 1

		for version, step := range stepGroup.Versions {
			if step.PublishedAt.After(lastCheckTime) {
				if isFirstVersion {
					stepVersionURLMap, ok := newSteps[stepID]
					if !ok {
						stepVersionURLMap = map[string]string{}
					}

					stepVersionURLMap[version] = strings.TrimSuffix(step.Source.Git, ".git")
					newSteps[stepID] = stepVersionURLMap
				} else {
					stepVersionURLMap, ok := updatedSteps[stepID]
					if !ok {
						stepVersionURLMap = map[string]string{}
					}

					stepVersionURLMap[version] = strings.TrimSuffix(step.Source.Git, ".git")
					updatedSteps[stepID] = stepVersionURLMap
				}
			}
		}
	}
	//

	// print release
	backgroundContext := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: flagGithubAPIToken},
	)
	oauthClient := oauth2.NewClient(backgroundContext, tokenSource)
	client := github.NewClient(oauthClient)

	fmt.Println()
	fmt.Printf("## New steps\n")
	fmt.Println()

	newStepIDs := []string{}
	for stepID := range newSteps {
		newStepIDs = append(newStepIDs, stepID)
	}
	sort.Strings(newStepIDs)

	for _, stepID := range newStepIDs {
		stepVersionURLMap := newSteps[stepID]

		for version := range stepVersionURLMap {
			fmt.Printf("- __%s %s__\n", stepID, version)
		}
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println()

	fmt.Printf("## Step updates\n")
	fmt.Println()

	stepIDs := []string{}
	for stepID := range updatedSteps {
		stepIDs = append(stepIDs, stepID)
	}
	sort.Strings(stepIDs)

	for _, stepID := range stepIDs {
		stepVersionURLMap := updatedSteps[stepID]

		versions := []string{}
		for version := range stepVersionURLMap {
			versions = append(versions, version)
		}

		newVersion, err := findLatestVersion(versions)
		if err != nil {
			return err
		}

		fmt.Printf("- __%s %s:__\n", stepID, newVersion)

		for version, url := range stepVersionURLMap {
			split := strings.Split(url, "/")
			if len(split) < 2 {
				return fmt.Errorf("invalid step url: %s", url)
			}
			name := split[len(split)-1]
			owner := split[len(split)-2]

			release, err := getRelease(backgroundContext, client, owner, name, version)
			if err != nil {
				return err
			}

			if release != nil && release.Body != nil {
				split := strings.Split(*release.Body, "\n")
				for _, note := range split {
					normalized := normalizeReleaseLine(note)
					if normalized != "" {
						fmt.Printf("  %s\n", normalized)
					}
				}
			}
		}
	}

	return nil
}

func init() {
	RootCmd.AddCommand(stepChangesCmd)
	stepChangesCmd.Flags().StringVarP(&flagGithubAPIToken, "api-token", "", "", "Github API Access token. Define this flag or set STEPPER_GITHUB_API_TOKEN env.")
	stepChangesCmd.Flags().StringVarP(&flagStartTime, "start", "", "", "From which time should collect the step changes? Format: 2006-01-02.")
}
