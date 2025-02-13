package testing

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/mimiro-io/datahub-client-sdk-go"
	egdm "github.com/mimiro-io/entity-graph-data-model"
	"os"
	"path/filepath"
)

type Manifest struct {
	Common        Common         `json:"common"`
	Tests         []*Test        `json:"tests"`
	Variables     map[string]any `json:"variables"`
	VariablesPath string         `json:"variablesPath"`
}

type Test struct {
	Id                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	IncludeCommon      bool                   `json:"includeCommon,omitempty"`
	Job                *datahub.Job           `json:"-"`
	JobPath            string                 `json:"jobPath"`
	RequiredDatasets   []*StoredDataset       `json:"requiredDatasets,omitempty"`
	ExpectedOutput     *egdm.EntityCollection `json:"-"`
	ExpectedOutputPath string                 `json:"expectedOutput,omitempty"`
}

type Common struct {
	RequiredDatasets []*StoredDataset `json:"requiredDatasets,omitempty"`
}

type StoredDataset struct {
	Name             string                 `json:"name"`
	Path             string                 `json:"path"`
	EntityCollection *egdm.EntityCollection `json:"-"`
}

func (sd StoredDataset) String() string {
	return sd.Name
}

func (t *Test) AddRequiredDataset(dataset *StoredDataset) {
	t.RequiredDatasets = append(t.RequiredDatasets, dataset)
}

func (t *Test) RemoveRequiredDataset(dataset *StoredDataset) {
	for i, d := range t.RequiredDatasets {
		if d.Name == dataset.Name {

			t.RequiredDatasets = append(t.RequiredDatasets[:i], t.RequiredDatasets[i+1:]...)
			break
		}

	}
}

func (t *Test) RemoveAllRequiredDatasets() {
	t.RequiredDatasets = nil
}

func (m *Manifest) AddTest(test *Test) error {
	for _, t := range m.Tests {
		if t.Id == test.Id {
			return fmt.Errorf("test with id %s already exists", test.Id)
		}
	}
	m.Tests = append(m.Tests, test)
	return nil
}

func (m *Manifest) RemoveAllTests() {
	m.Tests = nil
}

func (m *Manifest) GetTest(testId string) *Test {
	for i, test := range m.Tests {
		if test.Id == testId {
			return m.Tests[i]
		}
	}
	return nil
}

func LoadManifest(path string) *Manifest {
	projectRoot := getGitRootPath(filepath.Dir(path))
	manifest := parseManifestConfig(path)

	var variables map[string]any
	if manifest.VariablesPath != "" {
		variables = readVariables(filepath.Join(projectRoot, manifest.VariablesPath))
	}
	manifest.Variables = variables

	for i, dataset := range manifest.Common.RequiredDatasets {
		ec, err := ReadEntities(filepath.Join(projectRoot, dataset.Path))
		if err != nil {
			log.Printf("failed to read entities from common dataset %s: %s", dataset.Name, err)
		}
		manifest.Common.RequiredDatasets[i].EntityCollection = ec
	}

	for i, test := range manifest.Tests {
		job, err := ReadJobConfig(projectRoot, test.JobPath, variables)
		if err != nil {
			log.Printf("failed to read job for test %s: %s", test.Id, err)
			continue
		}
		manifest.Tests[i].Job = job

		for y, dataset := range test.RequiredDatasets {
			ec, err := ReadEntities(filepath.Join(projectRoot, dataset.Path))
			if err != nil {
				log.Printf("failed to read entities from dataset %s: %s", dataset.Name, err)
			}
			manifest.Tests[i].RequiredDatasets[y].EntityCollection = ec
		}

		expected, err := ReadEntities(filepath.Join(projectRoot, test.ExpectedOutputPath))
		if err != nil {
			log.Printf("failed to read expected output for test %s: %s", test.Id, err)
			continue
		}
		manifest.Tests[i].ExpectedOutput = expected
	}
	return manifest
}

// parseManifest parses the manifest file at the given path and returns a *Manifest
func parseManifestConfig(path string) *Manifest {
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
