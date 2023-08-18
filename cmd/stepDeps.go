package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/spf13/cobra"
)

var stepDeps = &cobra.Command{
	Use:   "stepDeps",
	Short: "Print dependencies of Steps",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		stepDependencyAnalyser := StepDependencyAnalyser{logger: logger}

		stepsRootDir := "/Users/godrei/Development/turbolift/bitrise-steps"
		if err := stepDependencyAnalyser.AnalyseAllBitriseSteps(stepsRootDir); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(stepDeps)
}

type StepDependencyAnalyser struct {
	logger log.Logger
}

func (a StepDependencyAnalyser) Analyse() error {
	imports, err := allImportedBitrisePackages("./")
	if err != nil {
		return err
	}

	//imports, err := allImportedBitriseRootPackages("./")
	//if err != nil {
	//	return err
	//}

	fmt.Println(strings.Join(imports, "\n"))
	return nil
}

func (a StepDependencyAnalyser) AnalyseAllBitriseSteps(rootDir string) error {
	// <rootDir>/work/bitrise-io/bitrise-step-update-gitops-repository
	workDir := filepath.Join(rootDir, "work")
	orgDirs, err := os.ReadDir(workDir)
	if err != nil {
		return err
	}

	allStepImportedBitriseRootPackagesMap := map[string]bool{}

	for _, orgDir := range orgDirs {
		orgDirPth := filepath.Join(workDir, orgDir.Name())
		stat, err := os.Stat(orgDirPth)
		if err != nil {
			return err
		}
		if stat.IsDir() == false {
			continue
		}

		stepsDirs, err := os.ReadDir(orgDirPth)
		if err != nil {
			return err
		}

		for _, stepDir := range stepsDirs {
			a.logger.Println()

			stepDirPth := filepath.Join(orgDirPth, stepDir.Name())
			a.logger.Infof("Analysing: %s", stepDirPth)

			mainGoFilePth := filepath.Join(stepDirPth, "main.go")
			_, err := os.Stat(mainGoFilePth)
			if err != nil {
				a.logger.Warnf("%s is not a go step", stepDir.Name())
				continue
			}

			goModFilePth := filepath.Join(stepDirPth, "go.mod")
			_, err = os.Stat(goModFilePth)
			if err != nil {
				a.logger.Warnf("%s is not a go module based step", stepDir.Name())
				continue
			}

			imports, err := allImportedBitriseRootPackages(stepDirPth)
			if err != nil {
				return err
			}

			a.logger.Printf("%d bitrise root packages imported", len(imports))

			for _, pkg := range imports {
				allStepImportedBitriseRootPackagesMap[pkg] = true
			}
		}
	}

	var allStepImportedBitriseRootPackages []string
	for pkg := range allStepImportedBitriseRootPackagesMap {
		allStepImportedBitriseRootPackages = append(allStepImportedBitriseRootPackages, pkg)
	}

	depsByCategory, err := categoriseDeps(allStepImportedBitriseRootPackages)
	if err != nil {
		return err
	}

	for cat, deps := range depsByCategory {
		a.logger.Println()
		a.logger.Printf("%s:", cat)
		a.logger.Printf(strings.Join(deps, "\n"))
	}

	return nil
}

func categoriseDeps(rootPks []string) (map[string][]string, error) {
	depsByCategory := map[string][]string{}

	for _, pkg := range rootPks {
		split := strings.Split(pkg, "/")
		if len(split) < 3 {
			return nil, fmt.Errorf("invalid package: %s: should contain at least 3 parts separated by '/' character", pkg)
		}

		pkgName := split[2]
		switch {
		case isLib(pkgName):
			libs := depsByCategory["lib"]
			libs = append(libs, pkg)
			depsByCategory["lib"] = libs
		case isStep(pkgName):
			steps := depsByCategory["step"]
			steps = append(steps, pkg)
			depsByCategory["step"] = steps
		case isTool(pkgName):
			tools := depsByCategory["tool"]
			tools = append(tools, pkg)
			depsByCategory["tool"] = tools
		default:
			return nil, fmt.Errorf("unknown category for dep: %s", pkg)
		}
	}

	for cat, deps := range depsByCategory {
		sort.Strings(deps)
		depsByCategory[cat] = deps
	}

	return depsByCategory, nil
}

func isLib(pkgName string) bool {
	return strings.HasPrefix(pkgName, "go-") ||
		pkgName == "bitrise-init" ||
		pkgName == "doublestar" ||
		pkgName == "appcenter" ||
		pkgName == "goinp"
}

func isStep(pkgName string) bool {
	return strings.HasPrefix(pkgName, "steps-") || strings.HasPrefix(pkgName, "bitrise-step-")
}

func isTool(pkgName string) bool {
	return pkgName == "bitrise" || pkgName == "stepman" || pkgName == "envman" || pkgName == "depman"
}

func currentPackageName(dir string) (string, error) {
	mainGoPth := filepath.Join(dir, "main.go")
	_, err := os.Stat(mainGoPth)

	args := []string{"list"}
	if err != nil {
		args = append(args, "./...")
	}

	cmd := command.NewFactory(env.NewRepository()).Create("go", args, &command.Opts{
		Stdout:      nil,
		Stderr:      nil,
		Stdin:       nil,
		Env:         nil,
		Dir:         dir,
		ErrorFinder: nil,
	})
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		fmt.Println(out)
		return "", err
	}

	split := strings.Split(out, "\n")
	var packages []string
	for _, e := range split {
		if e == "" {
			continue
		}
		packages = append(packages, e)
	}
	if len(packages) == 1 {
		return packages[0], nil
	}

	rootPackage := ""
	for _, e := range split {
		if e == "" {
			continue
		}
		packagePath, err := parsePkg(e)
		if err != nil {
			return "", err
		}
		pkg := fmt.Sprintf("%s/%s/%s", packagePath.Host, packagePath.Owner, packagePath.Name)

		if rootPackage == "" {
			rootPackage = pkg
		} else if rootPackage != pkg {
			return "", fmt.Errorf("multiple root package detected: %s, %s", rootPackage, pkg)
		}
	}
	return rootPackage, nil
}

func allImportedBitrisePackages(dir string) ([]string, error) {
	imports, err := allImportPaths(dir)
	if err != nil {
		return nil, err
	}

	currentPkg, err := currentPackageName(dir)
	if err != nil {
		return nil, err
	}

	var normalised []string
	for _, pkg := range imports {
		if strings.HasPrefix(pkg, currentPkg) {
			continue
		}

		if strings.HasPrefix(pkg, "github.com/bitrise-io") || strings.HasPrefix(pkg, "github.com/bitrise-steplib") {
			normalised = append(normalised, pkg)
		}
	}

	return normalised, nil
}

func allImportedBitriseRootPackages(dir string) ([]string, error) {
	imports, err := allImportedBitrisePackages(dir)
	if err != nil {
		return nil, err
	}

	normalisedImportMap := map[string]bool{}
	for _, pkg := range imports {
		split := strings.Split(pkg, "/")
		if len(split) < 3 {
			return nil, fmt.Errorf("invalid package: %s: should contain at least 3 parts separated by '/' character", pkg)
		}

		var rootPkg string
		if len(split) > 3 && split[3] == "v2" {
			rootPkg = strings.Join(split[0:4], "/")
		} else {
			rootPkg = strings.Join(split[0:3], "/")
		}

		normalisedImportMap[rootPkg] = true
	}

	var normalisedImports []string
	for pkg := range normalisedImportMap {
		normalisedImports = append(normalisedImports, pkg)
	}

	return normalisedImports, nil
}

func allImportPaths(dir string) ([]string, error) {
	cmd := command.NewFactory(env.NewRepository()).Create("go", []string{"list", "-f", `{{ join .Imports  "\n"}}`, "./..."}, &command.Opts{
		Stdout:      nil,
		Stderr:      nil,
		Stdin:       nil,
		Env:         nil,
		Dir:         dir,
		ErrorFinder: nil,
	})
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		fmt.Println(out)
		return nil, err
	}

	return strings.Split(out, "\n"), nil
}
