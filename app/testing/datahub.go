package testing

import (
	dh "github.com/mimiro-io/datahub"
)

func StartTestDatahub(location string, port string) (*dh.DatahubInstance, error) {

	cfg, err := dh.LoadConfig("")
	if err != nil {
		return nil, err
	}
	cfg.Port = port
	cfg.StoreLocation = location + "/store"
	cfg.SecurityStorageLocation = location + "/security"

	dhi, err := dh.NewDatahubInstance(cfg)
	if err != nil {
		return nil, err
	}
	go dhi.Start()

	return dhi, err
}
