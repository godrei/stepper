package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/godrei/stepper/tools"
	"github.com/spf13/cobra"
)

var (
	flagStepsConstFilePath string
)

var stepLatestsCmd = &cobra.Command{
	Use:   "stepLatests",
	Short: "Creates a steps/const.go file for bitrise-init tool with the current latest step versions.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := stepLatests(); err != nil {
			log.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func collectStepIds(stepIDConstPth string) ([]string, error) {
	content, err := fileutil.ReadStringFromFile(stepIDConstPth)
	if err != nil {
		return []string{}, err
	}

	// CertificateAndProfileInstallerID = "certificate-and-profile-installer"
	pattern := `.*ID = "(?P<id>.*)"`
	re := regexp.MustCompile(pattern)

	reader := strings.NewReader(content)
	scanner := bufio.NewScanner(reader)

	stepIDs := []string{}
	for scanner.Scan() {
		line := scanner.Text()

		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			stepID := matches[1]
			stepIDs = append(stepIDs, stepID)
		}
	}

	return stepIDs, nil
}

func stepIDFrom(stepName string) string {
	stepID := ""
	l := 0
	for s := stepName; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 {
			l = len(s)
		}
		stepID += "-" + s[:l]
	}
	return stepID
}

func replaceStepVersions(stepIDConstPth string, stepIDVersionMap map[string]string) (string, error) {
	content, err := fileutil.ReadStringFromFile(stepIDConstPth)
	if err != nil {
		return "", err
	}

	// CertificateAndProfileInstallerID = "certificate-and-profile-installer"
	idPattern := `.*ID = "(?P<id>.*)"`
	idRe := regexp.MustCompile(idPattern)

	// CertificateAndProfileInstallerVersion = "1.8.4"
	versionPattern := `.*Version = "(?P<version>.*)"`
	versionRe := regexp.MustCompile(versionPattern)

	reader := strings.NewReader(content)
	scanner := bufio.NewScanner(reader)

	currentStepID := ""

	lines := []string{}
	for scanner.Scan() {
		line := scanner.Text()

		if matches := idRe.FindStringSubmatch(line); len(matches) == 2 {
			stepID := matches[1]
			currentStepID = stepID

			fmt.Printf("replacing step version: %s\n", currentStepID)
		}

		if matches := versionRe.FindStringSubmatch(line); len(matches) == 2 {
			stepVersion := matches[1]

			newVersion, ok := stepIDVersionMap[currentStepID]
			if !ok {
				return "", fmt.Errorf("no version found for: %s", currentStepID)
			}

			fmt.Printf("new version: %s\n", newVersion)

			lines = append(lines, strings.Replace(line, stepVersion, newVersion, -1))
		} else {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), nil
}

func stepLatests() error {
	if flagStepsConstFilePath == "" {
		return fmt.Errorf("steps-const-file not defined")
	}
	if exist, err := pathutil.IsPathExists(flagStepsConstFilePath); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("steps-const-file does not exist at: %s", flagStepsConstFilePath)
	}

	desiredStepIDs, err := collectStepIds(flagStepsConstFilePath)
	if err != nil {
		return err
	}

	if err := tools.StepmanUpdate(defaultSteplibURI); err != nil {
		return err
	}

	steplib, err := tools.StepmanExportSpec(defaultSteplibURI, tools.ExportTypesLatest)
	if err != nil {
		return err
	}

	stepIDVersionMap := map[string]string{}
	for stepID, stepGroup := range steplib.Steps {
		for version := range stepGroup.Versions {
			for _, desiredStepID := range desiredStepIDs {
				if desiredStepID == stepID {
					stepIDVersionMap[stepID] = version
				}
			}
			continue
		}
	}

	generatedContent, err := replaceStepVersions(flagStepsConstFilePath, stepIDVersionMap)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Generated:\n%s\n", generatedContent)

	return nil
}

func init() {
	RootCmd.AddCommand(stepLatestsCmd)
	stepLatestsCmd.Flags().StringVarP(&flagStepsConstFilePath, "steps-const-file", "", "", "Path to the local steps/const.go file in the bitrise-init project.")
}
