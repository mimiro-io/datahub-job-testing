# Datahub Job Testing

This is a simple test runner for datahub jobs. It is designed to be used in a CI/CD pipeline to verify that jobs are working as expected.

### Usage
This tool expects a manifest file to be present inside your datahub config project. The manifest file should contain a list of tests to run. 
For each test, the runner will spin up a new datahub, upload the defined required datasets, and run the job. It will then compare the output with the expected output.

#### Test configuration
Each test case defined has the following properties:
```json

{
  "name": "Case1: Unique test name",
  "description": "Description of the test case",
  "includeCommon": true, # Include common configuration in this test run. Default is false
  "jobPath": "relative/filepath/to/my/job.json",
  "requiredDatasets": [
    {
      "name": "sdb.Animal",
      "path": "tests/testdata/case1/sdb-Animal.json"
    }
  ],
  "expectedOutput": "tests/expected/myJob/case1.json"
}
```
*Note: All file paths in the manifest file are relative to the repo root of the datahub config project*


#### Common configuration
Some configuration is common to all tests. To add datasets for all test cases, use the top-level property `common.requiredDatasets`. (See [example manifest](example-manifest.json) for details.)





## TODO:
- [x] Create test manifest
    * Test name
        * jobtitle.js
        * tags
        * sink dataset
        * source dataset
        * required datasets
            * where is the data stored
        * description
        * expected output location

- [x] Load manifest
- [ ] find all .ts transforms
    * copy into separate ts dir
    * compile them somehow
- [x] iterate through list of tests
- [x] spin up datahub for each
- [x] load required datasets
- [x] build job with sdk and run it
- [x] compare output with expected
- [ ] run egcl on the output (secondary)
