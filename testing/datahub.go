package testing

import (
	"context"
	dh "github.com/mimiro-io/datahub"
	"os"
)

type DatahubManager struct {
	Instance *dh.DatahubInstance
	Location string
}

func StartTestDatahub(port string) (*DatahubManager, error) {

	tmpDir, err := os.MkdirTemp("", "datahub-jobs-testing-")
	if err != nil {
		panic(err)
	}

	// create store and security folders
	os.MkdirAll(tmpDir+"/store", 0777)
	os.MkdirAll(tmpDir+"/security", 0777)

	cfg, err := dh.LoadConfig("")
	if err != nil {
		return nil, err
	}
	cfg.Port = port
	cfg.StoreLocation = tmpDir + "/store"
	cfg.SecurityStorageLocation = tmpDir + "/security"

	dhi, err := dh.NewDatahubInstance(cfg)
	if err != nil {
		return nil, err
	}
	go dhi.Start()

	return &DatahubManager{Instance: dhi, Location: tmpDir}, nil
}

func (dm *DatahubManager) Cleanup() {
	dm.Instance.Stop(context.Background())
	os.RemoveAll(dm.Location)
	os.Unsetenv("LOG_LEVEL")
}
