package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/spf13/cobra"
)

var dependentProjectsCmd = &cobra.Command{
	Use:   "dependentProjects",
	Short: "List projects depending on the given package",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		dependentPackageFinder := DependentPackageFinder{logger: logger}

		rootDir := "/Users/godrei/Development/turbolift"
		pkg := packageFlag
		if pkg == "" {
			logger.Errorf("package not specified")
			os.Exit(1)
		}
		if err := dependentPackageFinder.FindDependentPackages(rootDir, pkg); err != nil {
			logger.Errorf(err.Error())
			os.Exit(1)
		}
	},
}

var (
	packageFlag string
)

func init() {
	RootCmd.AddCommand(dependentProjectsCmd)
	dependentProjectsCmd.Flags().StringVarP(&packageFlag, "pkg", "", "", "List projects depending on the given package.")
}

type DependentPackageFinder struct {
	logger log.Logger
}

func (a DependentPackageFinder) iterateOnTurboliftProjects(projectDirs []string, do func(string)) error {
	for _, root := range projectDirs {
		// <pth>/work/bitrise-io/go-utils
		workDir := filepath.Join(root, "work")
		orgDirs, err := os.ReadDir(workDir)
		if err != nil {
			return err
		}

		for _, orgDir := range orgDirs {
			orgDirPth := filepath.Join(workDir, orgDir.Name())
			stat, err := os.Stat(orgDirPth)
			if err != nil {
				return err
			}
			if stat.IsDir() == false {
				continue
			}

			projDirs, err := os.ReadDir(orgDirPth)
			if err != nil {
				return err
			}

			for _, projDir := range projDirs {
				projDirPth := filepath.Join(orgDirPth, projDir.Name())
				do(projDirPth)
			}
		}
	}
	return nil
}

func (a DependentPackageFinder) FindDependentPackages(rootDir, pkg string) error {
	stepsDir := filepath.Join(rootDir, "bitrise-steps")
	libsDir := filepath.Join(rootDir, "bitrise-libs")
	toolsDir := filepath.Join(rootDir, "bitrise-tools")

	err := a.iterateOnTurboliftProjects([]string{stepsDir, libsDir, toolsDir}, func(projDirPth string) {
		goModFilePth := filepath.Join(projDirPth, "go.mod")
		_, err := os.Stat(goModFilePth)
		if err != nil {
			return
		}

		imports, err := allImportedBitrisePackages(projDirPth)
		if err != nil {
			return
		}

		isDependent := false
		for _, imp := range imports {
			if strings.HasPrefix(imp, pkg) {
				isDependent = true
				break
			}
		}

		if isDependent {
			repo := filePathToRepo(projDirPth)
			fmt.Println(repo)
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func filePathToRepo(pth string) string {
	org := filepath.Base(filepath.Dir(pth))
	repo := filepath.Base(pth)
	isV1 := false
	if strings.HasSuffix(repo, "-v1") {
		isV1 = true
		repo = strings.TrimSuffix(repo, "-v1")
	}

	s := fmt.Sprintf("%s/%s", org, repo)
	if isV1 {
		s = s + "@v1"
	}
	return s
}
