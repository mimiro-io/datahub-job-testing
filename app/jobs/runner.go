package jobs

import (
	"github.com/mimiro-io/datahub-client-sdk-go"
	"time"
)

func RunAndWait(client *datahub.Client, jobId string) error {
	err := client.RunJobAsFullSync(jobId)
	if err != nil {
		return err
	}
	for {
		status, err := client.GetJobStatus(jobId)
		if err != nil {
			return err
		}

		if status == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}
	return nil
}
