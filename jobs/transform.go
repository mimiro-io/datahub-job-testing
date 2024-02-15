package jobs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/evanw/esbuild/pkg/cli"
	"github.com/labstack/gommon/log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// GetTransformPath returns a file path if an extra property Path is defined in the transform config
func GetTransformPath(rawJob []byte) string {
	pattern := "\"transform\":\\s?{\\s*\"Path\":\\s?\"(.+)\",\\s*\"Type"
	r, _ := regexp.Compile(pattern)
	matches := r.FindStringSubmatch(string(rawJob))
	if len(matches) > 0 {
		return matches[1]
	}
	return ""
}

// GetTransformFromFile returns as base64 encoded string of the transform file
func GetTransformFromFile(path string) string {
	importer := NewImporter(path)
	code, err := importer.ImportJs()
	if err != nil {
		panic(err)
	}
	return importer.Encode(code)
}

type Importer struct {
	file string
}

func NewImporter(file string) *Importer {
	return &Importer{
		file: file,
	}
}

func (imp *Importer) ImportJs() ([]byte, error) {
	result, err := imp.buildCode()
	if err != nil {
		return nil, err
	}

	transform := result.OutputFiles[0]

	code := imp.fix(string(transform.Contents))

	return []byte(code), nil
}

func (imp *Importer) Cmd(cmd []string) ([]byte, error) {
	cmdExec := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(cmd, " ")))
	return cmdExec.CombinedOutput()
}

func (imp *Importer) Encode(code []byte) string {
	return base64.StdEncoding.EncodeToString(code)
}

func (imp *Importer) buildCode() (*api.BuildResult, error) {
	options, err := cli.ParseBuildOptions([]string{
		imp.file,
		"--bundle",
		"--format=esm",
		"--target=es2016",
		"--outfile=out.js",
	})
	if err != nil {
		return nil, err
	}

	result := api.Build(options)

	for _, w := range result.Warnings {
		log.Warnf(w.Text)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Errorf(fmt.Sprintf("%s:%v", e.Text, e.Location))
		}
		return nil, errors.New("something wrong happened with the compile")
	}

	return &result, nil
}

func (imp *Importer) fix(content string) string {
	if strings.Contains(content, "export") {
		i := strings.Index(content, "export")
		c := content[:i]
		return c
	}
	return content
}

func (imp *Importer) LoadRaw() ([]byte, error) {
	return os.ReadFile(imp.file)
}
