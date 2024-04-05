package datahub_job_testing

import (
	"fmt"
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/jobs"
	"github.com/mimiro-io/datahub-job-testing/testing"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
)

type TestRunner struct {
	Manifest *testing.Manifest
}

func NewTestRunner(manifestPath string) *TestRunner {
	return &TestRunner{
		Manifest: testing.LoadManifest(manifestPath),
	}
}

func (tr *TestRunner) RunSingleTest(testId string) bool {
	return tr.runTests(testId)
}

func (tr *TestRunner) RunAllTests() bool {
	return tr.runTests("")
}

func (tr *TestRunner) runTests(testId string) bool {
	successful := true
	startedTests := 0

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
			dm.Cleanup()
			log.Printf("failed to create datahub client for test %s: %s", test.Id, err)
			continue
		}

		// upload required datasets
		for _, dataset := range test.RequiredDatasets {
			err := testing.LoadEntities(dataset, client)
			if err != nil {
				dm.Cleanup()
				log.Printf("failed to load required dataset %s for test %s: %s", dataset.Name, test.Id, err)
				continue
			}
		}

		if test.IncludeCommon && tr.Manifest.Common.RequiredDatasets != nil {
			for _, dataset := range tr.Manifest.Common.RequiredDatasets {
				err := testing.LoadEntities(dataset, client)
				if err != nil {
					dm.Cleanup()
					log.Printf("failed to load required dataset %s for test %s: %s", dataset.Name, test.Id, err)
					continue
				}
			}
		}

		// upload job
		err = client.AddJob(test.Job)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to upload job for test %s: %s", test.Id, err)
			continue
		}

		// Create job sink dataset
		err = client.AddDataset(test.Job.Sink["Name"].(string), nil)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to create sink dataset for test %s: %s", test.Id, err)
			continue
		}

		// run job
		err = jobs.RunAndWait(client, test.Job.Id)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to run job for test %s: %s", test.Id, err)
			continue
		}

		// compare output
		entities, err := client.GetEntities(test.Job.Sink["Name"].(string), "", 0, false, true)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to get entities from sink dataset for test %s: %s", test.Id, err)
			continue
		}
		if len(entities.GetEntities()) == 0 {
			successful = false
			dm.Cleanup()
			log.Printf("No entities found in sink dataset for test %s", test.Id)
			continue
		}
		log.Printf("Found %d entities in sink dataset for test %s", len(entities.GetEntities()), test.Id)

		dm.Cleanup()

		equal, diffs := testing.CompareEntities(test.ExpectedOutput, entities)
		if !equal {
			successful = false
			log.Printf("Listing diffs for test %s", test.Id)
			logDiffs(diffs, test.Id)
		}
	}
	if startedTests == 0 && testId != "" {
		log.Fatalf("No test found with id %s", testId)
		return false
	}
	if successful {
		log.Printf("All %d tests ran successfully!", startedTests)
		return true
	} else {
		log.Printf("Finished running %d tests. One or more tests failed", startedTests)
		return false
	}
}

func (tr *TestRunner) DetermineRequiredDatasets(testId string) ([]*testing.StoredDataset, error) {
	var usedDatasets []*testing.StoredDataset
	var success bool
	test := tr.Manifest.GetTest(testId)
	if test == nil {
		return nil, fmt.Errorf("no test found with id %s", testId)
	}
	testRuns := 0
	for _, dataset := range test.RequiredDatasets {
		usedDatasets = append(usedDatasets, dataset)
		tr.Manifest.GetTest(testId).RemoveAllRequiredDatasets()
		for _, newDataset := range usedDatasets {
			tr.Manifest.GetTest(testId).AddRequiredDataset(newDataset)
		}
		success = tr.runTests(testId)
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
