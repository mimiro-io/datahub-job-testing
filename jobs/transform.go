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
	"path/filepath"
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
func GetTransformFromFile(projectRoot string, path string) string {
	importer := NewImporter(path)
	var code []byte
	var err error
	if filepath.Ext(path) == ".ts" {
		code, err = importer.ImportTs(projectRoot)
	} else {
		code, err = importer.ImportJs()
	}

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

func (imp *Importer) ImportTs(projectRoot string) ([]byte, error) {
	VerifyNodeInstallation(imp)

	var typescriptCmd []string

	if projectRoot != "" {
		typescriptCmd = []string{"cd", projectRoot, "&&", "npx", "tt", imp.file}
	} else {
		typescriptCmd = []string{"npx", "tt", imp.file}
	}
	code, err := imp.Cmd(typescriptCmd)

	if err != nil {
		return code, errors.New(fmt.Sprintf("%s\n%s", string(code), err))
	}
	return code, nil
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

func VerifyNodeInstallation(imp *Importer) {
	//check if node is installed
	checkForNodeCmd := []string{"node", "-v"}
	_, err := imp.Cmd(checkForNodeCmd)
	if err != nil {
		panic(err)
	}
	//list out npm packages
	checkForLibCmd := []string{"npm", "list"}
	checkForLibCmdGlobal := []string{"npm", "list", "-g"}
	library, _ := imp.Cmd(checkForLibCmd)
	libraryGlobal, _ := imp.Cmd(checkForLibCmdGlobal)

	//check if the package needed is installed.
	pkgList := strings.Split(string(library), "\n")
	pkgList = append(pkgList, strings.Split(string(libraryGlobal), "\n")...)
	pkgName := "datahub-tslib"
	isPackageInstalled := ListContainsSubstr(pkgList, pkgName)

	if isPackageInstalled == false {
		fmt.Println(fmt.Sprintf("Missing datahub-tslib package."))
		fmt.Println(fmt.Sprintf("Please install it. https://open.mimiro.io/software/typescript/"))
		os.Exit(1)
	}
}

func ListContainsSubstr(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(a, e) {
			return true
		}
	}
	return false
}
