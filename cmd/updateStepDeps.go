package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/spf13/cobra"
)

var updateDeps = &cobra.Command{
	Use:   "updateStepDeps",
	Short: "Update dependencies of Steps",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		stepDependencyUpdater := StepDependencyUpdater{logger: logger}

		if pkgFlag == "" {
			logger.Errorf("go package not specified")
			os.Exit(1)
		}

		if err := stepDependencyUpdater.UpdateIfNeeded(pkgFlag, verFlag, "./"); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

var (
	pkgFlag    string
	verFlag    string
	dryRunFlag bool
)

func init() {
	RootCmd.AddCommand(updateDeps)
	updateDeps.Flags().StringVarP(&pkgFlag, "pkg", "", "", "Go package path to be updated.")
	updateDeps.Flags().StringVarP(&verFlag, "ver", "", "", "Go package version to be updated.")
	updateDeps.Flags().BoolVarP(&dryRunFlag, "dry", "", false, "Dry run.")
}

type StepDependencyUpdater struct {
	logger log.Logger
}

func (u StepDependencyUpdater) UpdateIfNeeded(pkg, ver, dir string) error {
	mainGoFilePth := filepath.Join(dir, "main.go")
	_, err := os.Stat(mainGoFilePth)
	if err != nil {
		u.logger.Warnf("not a go step")
		return nil
	}

	goModFilePth := filepath.Join(dir, "go.mod")
	_, err = os.Stat(goModFilePth)
	if err != nil {
		u.logger.Warnf("not a go module based step")
		return nil
	}

	imports, err := allImportedBitriseRootPackages(dir)
	if err != nil {
		return err
	}

	if sliceutil.IsStringInSlice(pkg, imports) {
		u.logger.Infof("Updating %s", pkg)
		if err := updateDependency(pkg, ver, dir); err != nil {
			return err
		}
	} else {
		u.logger.Infof("Not depending on: %s", pkg)
	}

	return nil
}

func updateDependency(pkg, ver, dir string) error {
	if err := getDependency(pkg, ver, dir); err != nil {
		return err
	}
	if err := tidyDependencies(dir); err != nil {
		return err
	}
	if err := vendorDependencies(dir); err != nil {
		return err
	}
	return nil
}

func getDependency(pkg, ver, dir string) error {
	pkgPth := pkg
	if ver != "" {
		pkgPth += "@" + ver
	}
	args := []string{"get", "-u", pkgPth}
	cmd := command.NewFactory(env.NewRepository()).Create("go", args, &command.Opts{
		Stdout:      nil,
		Stderr:      nil,
		Stdin:       nil,
		Env:         nil,
		Dir:         dir,
		ErrorFinder: nil,
	})

	if dryRunFlag {
		fmt.Println(cmd.PrintableCommandArgs())
	} else {
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			fmt.Println(out)
			return err
		}
	}

	return nil
}

func tidyDependencies(dir string) error {
	cmd := command.NewFactory(env.NewRepository()).Create("go", []string{"mod", "tidy"}, &command.Opts{
		Stdout:      nil,
		Stderr:      nil,
		Stdin:       nil,
		Env:         nil,
		Dir:         dir,
		ErrorFinder: nil,
	})

	if dryRunFlag {
		fmt.Println(cmd.PrintableCommandArgs())
	} else {
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			fmt.Println(out)
			return err
		}
	}

	return nil
}

func vendorDependencies(dir string) error {
	cmd := command.NewFactory(env.NewRepository()).Create("go", []string{"mod", "vendor"}, &command.Opts{
		Stdout:      nil,
		Stderr:      nil,
		Stdin:       nil,
		Env:         nil,
		Dir:         dir,
		ErrorFinder: nil,
	})

	if dryRunFlag {
		fmt.Println(cmd.PrintableCommandArgs())
	} else {
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			fmt.Println(out)
			return err
		}
	}

	return nil
}
