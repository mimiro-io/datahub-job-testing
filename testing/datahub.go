package testing

import (
	"context"
	dh "github.com/mimiro-io/datahub"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	os.Setenv("LOG_LEVEL", "ERROR")

	cfg, err := dh.LoadConfig("")
	if err != nil {
		return nil, err
	}
	cfg.Port = port
	cfg.StoreLocation = tmpDir + "/store"
	cfg.SecurityStorageLocation = tmpDir + "/security"
	cfg.Logger = GetLogger()

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

// GetLogger returns it's own *zap.SugaredLogger to override the default logger to minimize datahub log output
func GetLogger() *zap.SugaredLogger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.FatalLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := cfg.Build()
	return logger.Sugar()
}
