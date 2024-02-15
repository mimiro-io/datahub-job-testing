package main

import (
	"fmt"
	djt "github.com/mimiro-io/datahub-job-testing"
	"os"
)

func main() {

	usage := `
Usage:
  djt path/to/manifest.json [test_id]

Help:
  https://github.com/mimiro-io/datahub-job-testing
`

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	tr := djt.NewTestRunner(args[0])

	var singleTest string
	if len(args) > 1 {
		singleTest = args[1]
		tr.RunSingleTest(singleTest)
	} else {
		tr.RunAllTests()
	}
}
