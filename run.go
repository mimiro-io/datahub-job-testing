package main

import (
	"fmt"
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/app/jobs"
	"github.com/mimiro-io/datahub-job-testing/app/testing"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"os"
)

type TestRunner struct {
	ManifestManager *testing.ManifestManager
}

func NewTestRunner(manifestPath string) *TestRunner {
	return &TestRunner{
		ManifestManager: testing.NewManifestManager(manifestPath),
	}
}

func (tr *TestRunner) RunSingleTest(testId string) {
	tr.runTests(testId)
}

func (tr *TestRunner) RunAllTests() {
	tr.runTests("")
}

func main() {

	usage := `
Usage:
  djt path/to/manifest.json [test_id]

Help:
  https://github.com/mimiro-io/datahub-job-testing
`

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	tr := NewTestRunner(args[0])

	var singleTest string
	if len(args) > 1 {
		singleTest = args[1]
		tr.RunSingleTest(singleTest)
	} else {
		tr.RunAllTests()
	}
}

func (tr *TestRunner) runTests(testId string) {
	successful := true
	ranTests := 0

	for _, test := range tr.ManifestManager.Manifest.Tests {
		if testId != "" && test.Id != testId {
			continue
		}

		// Read jobs config
		job, err := testing.ReadJobConfig(tr.ManifestManager.ProjectRoot, test.JobPath, tr.ManifestManager.Variables)
		if err != nil {
			log.Print(err)
			continue
		}

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

		if test.IncludeCommon && tr.ManifestManager.Manifest.Common.RequiredDatasets != nil {
			for _, dataset := range tr.ManifestManager.Manifest.Common.RequiredDatasets {
				err := testing.LoadEntities(dataset, client)
				if err != nil {
					dm.Cleanup()
					log.Printf("failed to load required dataset %s for test %s: %s", dataset.Name, test.Id, err)
					continue
				}
			}
		}

		// upload job
		err = client.AddJob(job)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to upload job for test %s: %s", test.Id, err)
			continue
		}

		// Create job sink dataset
		err = client.AddDataset(job.Sink["Name"].(string), nil)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to create sink dataset for test %s: %s", test.Id, err)
			continue
		}

		// run job
		err = jobs.RunAndWait(client, job.Id)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to run job for test %s: %s", test.Id, err)
			continue
		}

		// compare output
		entities, err := client.GetEntities(job.Sink["Name"].(string), "", 0, false, true)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to get entities from sink dataset for test %s: %s", test.Id, err)
			continue
		}

		expected, err := testing.ReadEntities(test.ExpectedOutput)
		if err != nil {
			dm.Cleanup()
			log.Printf("failed to read expected output for test %s: %s", test.Id, err)
			continue
		}

		dm.Cleanup()

		equal, diffs := testing.CompareEntities(expected, entities)
		if !equal {
			successful = false
			log.Printf("Listing diffs for test %s", test.Id)
			logDiffs(diffs, test.Id)
		}
		ranTests++
	}
	if ranTests == 0 && testId != "" {
		log.Fatalf("No test found with id %s", testId)
	}
	if successful {
		log.Printf("All %d tests ran successfully!", ranTests)
	} else {
		log.Fatalf("Finished running %d tests. One or more tests failed", ranTests)
	}
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
