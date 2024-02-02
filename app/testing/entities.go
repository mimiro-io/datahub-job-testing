package testing

import (
	"encoding/json"
	"github.com/mimiro-io/datahub-client-sdk-go"
	egdm "github.com/mimiro-io/entity-graph-data-model"
	"log"
	"os"
	"reflect"
)

// LoadEntities loads entities from file path and upload to the given datahub dataset
func LoadEntities(dataset StoredDataset, client *datahub.Client) {
	entities := ReadEntities(dataset.Path)

	// create dataset
	err := client.AddDataset(dataset.Name, nil)
	if err != nil {
		panic(err)
	}

	// upload entities
	err = client.StoreEntities(dataset.Name, entities)
	if err != nil {
		panic(err)
	}
}

// ReadEntities reads entities from file path and returns *egdm.EntityCollection
func ReadEntities(path string) *egdm.EntityCollection {
	nsmanager := egdm.NewNamespaceContext()
	parser := egdm.NewEntityParser(nsmanager)
	parser.WithExpandURIs()

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	ec, err := parser.LoadEntityCollection(file)
	if err != nil {
		panic(err)
	}
	return ec
}

// CompareEntities compares two EntityCollections and returns true if they are equal
func CompareEntities(expected *egdm.EntityCollection, result *egdm.EntityCollection) bool {
	// strip recorded
	expected = stripRecorded(expected)
	result = stripRecorded(result)

	equal := reflect.DeepEqual(expected.Entities, result.Entities)
	if !equal {
		for _, entity := range expected.Entities {
			for _, entity2 := range result.Entities { // TODO: should also log entities only existing in result
				if entity2.ID == entity.ID {
					if !reflect.DeepEqual(entity, entity2) {
						expectedJson, _ := json.Marshal(entity) // TODO: sort prop and ref keys for easier comparison
						resultJson, _ := json.Marshal(entity2)
						log.Printf("Entity '%s' does not match expected output:\nExpected: %s\nResult: %s", entity.ID, expectedJson, resultJson)
					}
				}
			}
		}
	}

	return equal
}

// stripRecorded set recorded timestamp to 0 on all entities
// in the given EntityCollection to avoid false positives on entity comparison
func stripRecorded(collection *egdm.EntityCollection) *egdm.EntityCollection {
	for _, entity := range collection.Entities {
		entity.Recorded = 0
	}
	return collection
}
