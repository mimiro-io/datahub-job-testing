package main

import (
	"context"
	"fmt"
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/app/jobs"
	"github.com/mimiro-io/datahub-job-testing/app/testing"
	"log"
	"os"
)

func main() {

	// Read manifest
	manifestPath := "/Users/andebor/mimiro/datahub-config/tests/manifest.json" // Todo: make this a flag
	manifestManager := testing.NewManifestManager(manifestPath)

	for _, test := range manifestManager.Manifest.Tests {
		// Read jobs config
		job, err := testing.ReadJobConfig(manifestManager.ProjectRoot, test.JobPath, manifestManager.Variables)
		if err != nil {
			panic(err)
		}
		fmt.Println(job)

		// Start datahub instance
		tmpDir, err := os.MkdirTemp("", "datahub-jobs-testing-")
		if err != nil {
			panic(err)
		}

		// create store and security folders
		os.MkdirAll(tmpDir+"/store", 0777)
		os.MkdirAll(tmpDir+"/security", 0777)

		// startup data hub instance
		dhi, err := testing.StartTestDatahub(tmpDir, "10778")
		if err != nil {
			panic(err)
		}
		fmt.Print(dhi)

		// create client
		client, err := datahub.NewClient("http://localhost:10778")
		if err != nil {
			panic(err)
		}

		// upload required datasets
		for _, dataset := range test.RequiredDatasets {
			testing.LoadEntities(dataset, client)
		}

		if test.IncludeCommon && manifestManager.Manifest.Common.RequiredDatasets != nil {
			for _, dataset := range manifestManager.Manifest.Common.RequiredDatasets {
				testing.LoadEntities(dataset, client)
			}
		}

		// upload job
		err = client.AddJob(job)
		if err != nil {
			panic(err)
		}

		// Create job sink dataset
		err = client.AddDataset(job.Sink["Name"].(string), nil)
		if err != nil {
			panic(err)
		}

		// run job
		err = jobs.RunAndWait(client, job.Id)
		if err != nil {
			panic(err)
		}

		// compare output
		entities, err := client.GetEntities(job.Sink["Name"].(string), "", 0, false, true)
		if err != nil {
			panic(err)
		}

		expected := testing.ReadEntities(test.ExpectedOutput)

		dhi.Stop(context.Background()) // TODO: Should run this before exiting on panics
		os.RemoveAll(tmpDir)

		if !testing.CompareEntities(expected, entities) {
			log.Printf("One or more entities does not match expected output")
		}

	}
	log.Printf("Finished running tests!")
}
