package datahub_job_testing

import (
	"fmt"
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/jobs"
	"github.com/mimiro-io/datahub-job-testing/testing"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"os"
)

type TestRunner struct {
	Manifest *testing.Manifest
}

func NewTestRunner(manifestPath string) *TestRunner {
	return &TestRunner{
		Manifest: testing.LoadManifest(manifestPath),
	}
}

func (tr *TestRunner) RunSingleTest(testId string) ([]testing.Diff, bool) {
	return tr.runTests(testId)
}

func (tr *TestRunner) RunAllTests() bool {
	_, success := tr.runTests("")
	return success
}

func (tr *TestRunner) runTests(testId string) ([]testing.Diff, bool) {
	successfulCount := 0
	startedTests := 0
	var diffs []testing.Diff

	for _, test := range tr.Manifest.Tests {
		if testId != "" && test.Id != testId {
			continue
		}

		startedTests++

		// startup data hub instance
		dm, err := testing.StartTestDatahub("10778")
		if err != nil {
			log.Printf("failed to start test datahub for test %s: %s", test.Id, err)
			continue
		}

		// create client
		client, err := datahub.NewClient("http://localhost:10778")
		if err != nil {
			log.Printf("failed to create datahub client for test %s: %s", test.Id, err)
			dm.Cleanup()
			continue
		}

		// upload required datasets
		for _, dataset := range test.RequiredDatasets {
			existInCommon := false
			if test.IncludeCommon {
				for _, commonDataset := range tr.Manifest.Common.RequiredDatasets {
					if dataset.Name == commonDataset.Name {
						existInCommon = true
						log.Printf("Required dataset %s found in common datasets. Will not upload", dataset.Name)
						break
					}
				}
			}
			if !existInCommon {
				err := testing.LoadEntities(dataset, client)
				if err != nil {
					log.Printf("failed to load required dataset %s for test %s: %s", dataset.Name, test.Id, err)
					dm.Cleanup()
					continue
				}
			}

		}

		if test.IncludeCommon && tr.Manifest.Common.RequiredDatasets != nil {
			for _, dataset := range tr.Manifest.Common.RequiredDatasets {
				err := testing.LoadEntities(dataset, client)
				if err != nil {
					log.Printf("failed to load required dataset %s for test %s: %s. Will exit", dataset.Name, test.Id, err)
					dm.Cleanup()
					os.Exit(1)
				}
			}
		}

		// upload job
		err = client.AddJob(test.Job)
		if err != nil {
			log.Printf("failed to upload job for test %s: %s", test.Id, err)
			dm.Cleanup()
			continue
		}

		// Create job sink dataset
		err = client.AddDataset(test.Job.Sink["Name"].(string), nil)
		if err != nil {
			log.Printf("failed to create sink dataset for test %s: %s", test.Id, err)
			dm.Cleanup()
			continue
		}

		// run job
		err = jobs.RunAndWait(client, test.Job.Id)
		if err != nil {
			log.Printf("failed to run job for test %s: %s", test.Id, err)
			dm.Cleanup()
			continue
		}

		// compare output
		entities, err := client.GetEntities(test.Job.Sink["Name"].(string), "", 0, false, true)
		if err != nil {
			log.Printf("failed to get entities from sink dataset for test %s: %s", test.Id, err)
			dm.Cleanup()
			continue
		}
		if len(entities.GetEntities()) == 0 {
			log.Printf("No entities found in sink dataset for test %s", test.Id)
			dm.Cleanup()
			continue
		}
		log.Printf("Found %d entities in sink dataset for test %s", len(entities.GetEntities()), test.Id)

		dm.Cleanup()

		equal, entityDiff := testing.CompareEntities(test.ExpectedOutput, entities)
		if !equal {
			diffs = append(diffs, entityDiff...)
			log.Printf("Listing diffs for test %s", test.Id)
			logDiffs(diffs, test.Id)
		} else {
			successfulCount++
		}
	}
	if startedTests == 0 && testId != "" {
		log.Fatalf("No test found with id %s", testId)
		return nil, false
	}
	if successfulCount == startedTests {
		log.Printf("All %d tests ran successfully!", startedTests)
		return nil, true
	} else {
		log.Printf("Finished running %d tests. One or more tests failed", startedTests)
		return diffs, false
	}
}

func (tr *TestRunner) DetermineRequiredDatasets(testId string, includeCommon bool) ([]*testing.StoredDataset, error) {
	var usedDatasets []*testing.StoredDataset
	var success bool
	var diffs []testing.Diff
	test := tr.Manifest.GetTest(testId)
	if test == nil {
		return nil, fmt.Errorf("no test found with id %s", testId)
	}
	test.IncludeCommon = includeCommon
	if includeCommon {
		log.Printf("Including common datasets when determining required datasets")
	}
	testRuns := 0
	for _, dataset := range test.RequiredDatasets {
		usedDatasets = append(usedDatasets, dataset)
		log.Printf("Running test with dataset(s): %s", usedDatasets)
		tr.Manifest.GetTest(testId).RemoveAllRequiredDatasets()
		for _, newDataset := range usedDatasets {
			tr.Manifest.GetTest(testId).AddRequiredDataset(newDataset)
		}
		diffs, success = tr.runTests(testId)
		// Check if diff is only additional entities
		if !success && len(diffs) > 0 {
			onlyExtra := 0
			for _, diff := range diffs {
				if diff.Type == "extra" && diff.ValueType == "entity" {
					onlyExtra++
				}
			}
			if onlyExtra == len(diffs) {
				success = true
				log.Printf("Only additional entities found in diff. No more required datasets")
			}
		}
		testRuns++
		if success {
			break
		}
	}
	if !success {
		return nil, fmt.Errorf("unable to determine required datasets for test %s after %d test runs: No successful runs with collected data", testId, testRuns)

	}
	return usedDatasets, nil
}

func logDiffs(diffs []testing.Diff, label string) {
	for _, diff := range diffs {
		caser := cases.Title(language.English)
		log.Printf("%s - %s: Key: %s ExpectedValue: %s ResultValue: %s ValueType: %s",
			label,
			caser.String(diff.Type),
			diff.Key,
			diff.ExpectedValue,
			diff.ResultValue,
			diff.ValueType)
	}

}
