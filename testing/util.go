package testing

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-client-sdk-go"
	"github.com/mimiro-io/datahub-job-testing/jobs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// getGitRootPath returns the root path of the repo
func getGitRootPath(path string) string {
	rootPath, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		fmt.Println("Failed to determine repo root. ", err)
		os.Exit(1)
	}
	return strings.TrimSuffix(string(rootPath), "\n")
}

// ReadJobConfig takes a path to a jobs config file and returns a datahub.Job struct
func ReadJobConfig(projectRoot string, jobPath string, variables map[string]any) (*datahub.Job, error) {
	bytes, err := os.ReadFile(jobPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read jobs config in path '%s': %s", jobPath, err)
	}

	//replace variables in jobs config
	if variables != nil {
		bytes, err = jobs.NewTemplating().ReplaceVariables(bytes, variables)
		if err != nil {
			panic(err)
		}
	}

	//Get optional transform path
	transformPath := jobs.GetTransformPath(bytes)

	var job *datahub.Job
	err = json.Unmarshal(bytes, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal jobs config in path '%s': %s", jobPath, err)
	}

	if transformPath != "" {
		code := jobs.GetTransformFromFile(filepath.Join(projectRoot, "transforms", transformPath)) // TODO: make more generic
		if code != "" {
			job.Transform.Code = code
		}
	}
	return job, nil
}
