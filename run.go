package main

import (
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/app/jobs"
	"github.com/mimiro-io/datahub-job-testing/app/testing"
	"log"
)

func main() {

	// Read manifest
	manifestPath := "/Users/andebor/mimiro/datahub-config/tests/manifest.json" // Todo: make this a flag
	manifestManager := testing.NewManifestManager(manifestPath)

	successful := true
	for _, test := range manifestManager.Manifest.Tests {
		// Read jobs config
		job, err := testing.ReadJobConfig(manifestManager.ProjectRoot, test.JobPath, manifestManager.Variables)
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

		if test.IncludeCommon && manifestManager.Manifest.Common.RequiredDatasets != nil {
			for _, dataset := range manifestManager.Manifest.Common.RequiredDatasets {
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

		if !testing.CompareEntities(expected, entities, test.Id) {
			successful = false
		}
		dm.Cleanup()

	}
	if successful {
		log.Printf("All tests ran successfully!")
	} else {
		log.Printf("Finished running tests. One or more tests failed")
	}
}
