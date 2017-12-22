package tools

import (
	"encoding/json"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/stepman/models"
)

// StepmanUpdate ...
func StepmanUpdate(stepLibURI string) error {
	updateCmd := command.New("bitrise", "stepman", "update", "--collection", stepLibURI)
	return updateCmd.Run()
}

// ExportTypes ...
type ExportTypes string

const (
	// ExportTypesFull ...
	ExportTypesFull ExportTypes = "full"
	// ExportTypesLatest ...
	ExportTypesLatest ExportTypes = "latest"
	// ExportTypesMinimal ...
	ExportTypesMinimal ExportTypes = "minimal"
)

// StepmanExportSpec ...
func StepmanExportSpec(stepLibURI string, exportType ExportTypes) (models.StepCollectionModel, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__spec__")
	if err != nil {
		return models.StepCollectionModel{}, err
	}
	specPth := filepath.Join(tmpDir, "spec.json")

	exportCmd := command.New("bitrise", "stepman", "export-spec", "--steplib", stepLibURI, "--output", specPth, "--export-type", string(exportType))
	if err := exportCmd.Run(); err != nil {
		return models.StepCollectionModel{}, err
	}

	specContentBytes, err := fileutil.ReadBytesFromFile(specPth)
	if err != nil {
		return models.StepCollectionModel{}, err
	}

	var steplib models.StepCollectionModel
	if err := json.Unmarshal(specContentBytes, &steplib); err != nil {
		return models.StepCollectionModel{}, err
	}

	return steplib, nil
}
