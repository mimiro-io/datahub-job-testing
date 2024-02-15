package testing

import (
	"github.com/mimiro-io/datahub-client-sdk-go"
	egdm "github.com/mimiro-io/entity-graph-data-model"
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

type Diff struct {
	Type          string // missing, diff, extra
	Key           string
	ExpectedValue any
	ResultValue   any
	ValueType     string // prop, ref or deleted
}

// CompareEntities compares two EntityCollections and returns true if they are equal
func CompareEntities(expected *egdm.EntityCollection, result *egdm.EntityCollection) (bool, []Diff) {
	// strip recorded
	expected = stripRecorded(expected)
	result = stripRecorded(result)
	var diffs []Diff
	equal := reflect.DeepEqual(expected.Entities, result.Entities)
	if !equal {
		for _, expectedEntity := range expected.Entities {
			found := false
			for _, resultEntity := range result.Entities {
				if resultEntity.ID == expectedEntity.ID {
					found = true
					if !reflect.DeepEqual(expectedEntity, resultEntity) {
						if !reflect.DeepEqual(expectedEntity.Properties, resultEntity.Properties) {
							diffs = append(diffs, findMapDiff(expectedEntity.Properties, resultEntity.Properties, "prop")...)
						}
						if !reflect.DeepEqual(expectedEntity.References, resultEntity.References) {
							diffs = append(diffs, findMapDiff(expectedEntity.References, resultEntity.References, "ref")...)
						}
					}
				}
			}
			if !found {
				diffs = append(diffs, Diff{
					Type:          "missing",
					Key:           expectedEntity.ID,
					ExpectedValue: "N/A",
					ResultValue:   "N/A",
					ValueType:     "entity",
				})
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
				diffs = append(diffs, Diff{
					Type:          "extra",
					Key:           entity.ID,
					ExpectedValue: "N/A",
					ResultValue:   "N/A", // TODO: Optional verbose logging of the extra entity
					ValueType:     "entity",
				})
			}
		}
	}
	return equal, diffs
}

// stripRecorded set recorded timestamp to 0 on all entities
// in the given EntityCollection to avoid false positives on entity comparison
func stripRecorded(collection *egdm.EntityCollection) *egdm.EntityCollection {
	for _, entity := range collection.Entities {
		entity.Recorded = 0
	}
	return collection
}

// findMapDiff finds the diff between an entity's prop or ref map and returns a []Diff slice
func findMapDiff(expected, result map[string]any, valueType string) []Diff {
	var diffs []Diff
	for key, val := range expected {
		val2, exist := result[key]
		if !exist {
			// Missing in result
			diffs = append(diffs, Diff{
				Type:          "missing",
				Key:           key,
				ExpectedValue: val,
				ResultValue:   nil,
				ValueType:     valueType,
			})
		} else if val != val2 {
			// Different value in result
			diffs = append(diffs, Diff{
				Type:          "diff",
				Key:           key,
				ExpectedValue: val,
				ResultValue:   val2,
				ValueType:     valueType,
			})
		}
	}

	for key, val := range result {
		_, exist := expected[key]
		if !exist {
			// Extra in result
			diffs = append(diffs, Diff{
				Type:          "extra",
				Key:           key,
				ExpectedValue: "N/A",
				ResultValue:   val,
				ValueType:     valueType,
			})
		}
	}
	return diffs
}
