package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/spf13/cobra"
)

var libDeps = &cobra.Command{
	Use:   "libDeps",
	Short: "Print dependencies of Libs",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		libDependencyAnalyser := LibDependencyAnalyser{logger: logger}

		stepsRootDir := "/Users/godrei/Development/turbolift/bitrise-libs"
		if err := libDependencyAnalyser.Analyse(stepsRootDir); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(libDeps)
}

type LibDependencyAnalyser struct {
	logger log.Logger
}

func (a LibDependencyAnalyser) Analyse(rootDir string) error {
	// <rootDir>/work/bitrise-io/go-utils
	workDir := filepath.Join(rootDir, "work")
	orgDirs, err := os.ReadDir(workDir)
	if err != nil {
		return err
	}

	allLibImportedBitriseRootPackagesMap := map[string]bool{}

	for _, orgDir := range orgDirs {
		orgDirPth := filepath.Join(workDir, orgDir.Name())
		stat, err := os.Stat(orgDirPth)
		if err != nil {
			return err
		}
		if stat.IsDir() == false {
			continue
		}

		libsDirs, err := os.ReadDir(orgDirPth)
		if err != nil {
			return err
		}

		for _, libsDir := range libsDirs {
			a.logger.Println()

			libDirPth := filepath.Join(orgDirPth, libsDir.Name())
			a.logger.Infof("Analysing: %s", libDirPth)

			goModFilePth := filepath.Join(libDirPth, "go.mod")
			_, err = os.Stat(goModFilePth)
			if err != nil {
				a.logger.Warnf("%s is not a go module based step", libsDir.Name())
				continue
			}

			imports, err := allImportedBitriseRootPackages(libDirPth)
			if err != nil {
				return err
			}

			a.logger.Printf("%d bitrise root packages imported", len(imports))

			for _, pkg := range imports {
				allLibImportedBitriseRootPackagesMap[pkg] = true
			}
		}
	}

	var allLibImportedBitriseRootPackages []string
	for pkg := range allLibImportedBitriseRootPackagesMap {
		allLibImportedBitriseRootPackages = append(allLibImportedBitriseRootPackages, pkg)
	}

	depsByCategory, err := categoriseDeps(allLibImportedBitriseRootPackages)
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

type PackagePath struct {
	Host  string
	Owner string
	Name  string
	IsV2  bool
}

func parsePkg(pkg string) (PackagePath, error) {
	split := strings.Split(pkg, "/")
	if len(split) < 3 {
		return PackagePath{}, fmt.Errorf("invalid package: %s: should contain at least 3 parts separated by '/' character", pkg)
	}
	return PackagePath{
		Host:  split[0],
		Owner: split[1],
		Name:  split[2],
		IsV2:  len(split) > 3 && split[3] == "v2",
	}, nil
}
