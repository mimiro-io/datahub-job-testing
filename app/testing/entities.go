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
func LoadEntities(dataset StoredDataset, client *datahub.Client) error {
	entities, err := ReadEntities(dataset.Path)
	if err != nil {
		return err
	}

	// create dataset
	err = client.AddDataset(dataset.Name, nil)
	if err != nil {
		return err
	}

	// upload entities
	err = client.StoreEntities(dataset.Name, entities)
	if err != nil {
		return err

	}
	return nil
}

// ReadEntities reads entities from file path and returns *egdm.EntityCollection
func ReadEntities(path string) (*egdm.EntityCollection, error) {
	nsmanager := egdm.NewNamespaceContext()
	parser := egdm.NewEntityParser(nsmanager)
	parser.WithExpandURIs()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	ec, err := parser.LoadEntityCollection(file)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

// CompareEntities compares two EntityCollections and returns true if they are equal
func CompareEntities(expected *egdm.EntityCollection, result *egdm.EntityCollection, label string) bool {
	// strip recorded
	expected = stripRecorded(expected)
	result = stripRecorded(result)

	equal := reflect.DeepEqual(expected.Entities, result.Entities)
	if !equal {
		for _, entity := range expected.Entities {
			found := false
			for _, entity2 := range result.Entities { // TODO: should also log entities only existing in result
				if entity2.ID == entity.ID {
					found = true
					if !reflect.DeepEqual(entity, entity2) {
						expectedJson, _ := json.Marshal(entity) // TODO: sort prop and ref keys for easier comparison
						resultJson, _ := json.Marshal(entity2)
						log.Printf("%s: Entity '%s' does not match expected output:\nExpected: %s\nResult: %s", label, entity.ID, expectedJson, resultJson)
					}
				}
			}
			if !found {
				log.Printf("%s: Entity '%s' does not exist in result", label, entity.ID)
			}
		}
		for _, entity := range result.Entities {
			found := false
			for _, entity2 := range expected.Entities {
				if entity2.ID == entity.ID {
					found = true
				}
			}
			if !found {
				log.Printf("%s: Entity '%s' does not exist in expected output", label, entity.ID)
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
