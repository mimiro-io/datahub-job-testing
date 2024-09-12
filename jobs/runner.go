package jobs

import (
	"fmt"
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
	result, err := client.GetJobsHistory()
	if err != nil {
		return err
	}
	for _, job := range result {
		if job.ID == jobId {
			if job.LastError != "" {
				return fmt.Errorf(job.LastError)
			}
		}
	}
	return nil
}
