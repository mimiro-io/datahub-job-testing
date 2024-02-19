package testing

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manifest struct {
	Common        Common `json:"common"`
	Tests         []Test `json:"tests"`
	VariablesPath string `json:"variablesPath"`
}

type Test struct {
	Id               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	IncludeCommon    bool            `json:"includeCommon,omitempty"`
	JobPath          string          `json:"jobPath,omitempty"`
	RequiredDatasets []StoredDataset `json:"requiredDatasets,omitempty"`
	ExpectedOutput   string          `json:"expectedOutput,omitempty"`
}

type Common struct {
	RequiredDatasets []StoredDataset `json:"requiredDatasets,omitempty"`
}

type StoredDataset struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ManifestManager struct {
	Manifest    *Manifest
	ProjectRoot string
	Variables   map[string]any
}

// NewManifestManager creates a new ManifestManager from the given manifest file path
func NewManifestManager(path string) *ManifestManager {
	projectRoot := getGitRootPath(filepath.Dir(path))
	manifest := parseManifest(path)

	var variables map[string]any
	if manifest.VariablesPath != "" {
		variables = readVariables(filepath.Join(projectRoot, manifest.VariablesPath))
	}

	mm := &ManifestManager{
		ProjectRoot: projectRoot,
		Manifest:    manifest,
		Variables:   variables,
	}
	mm.ExpandFilePaths()
	return mm
}

// parseManifest parses the manifest file at the given path and returns a *Manifest
func parseManifest(path string) *Manifest {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var manifest *Manifest
	err = json.Unmarshal(bytes, &manifest)
	if err != nil {
		panic(err)
	}
	return manifest
}

// readVariables reads the variables from the given file path and returns a map of the variables
func readVariables(path string) map[string]any {
	var variables map[string]any

	varBytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(varBytes, &variables)
	if err != nil {
		panic(err)
	}

	return variables
}

// ExpandFilePaths expands all file paths in the manifest to absolute paths
func (mm *ManifestManager) ExpandFilePaths() {
	for i, test := range mm.Manifest.Tests {
		mm.Manifest.Tests[i].JobPath = filepath.Join(mm.ProjectRoot, test.JobPath)
		for j, dataset := range test.RequiredDatasets {
			mm.Manifest.Tests[i].RequiredDatasets[j].Path = filepath.Join(mm.ProjectRoot, dataset.Path)
		}
		mm.Manifest.Tests[i].ExpectedOutput = filepath.Join(mm.ProjectRoot, test.ExpectedOutput)
	}
	for i, dataset := range mm.Manifest.Common.RequiredDatasets {
		mm.Manifest.Common.RequiredDatasets[i].Path = filepath.Join(mm.ProjectRoot, dataset.Path)
	}
}
