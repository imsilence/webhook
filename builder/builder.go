package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"webhook/domain/requests"
	"webhook/utils"

	"webhook/config"

	"github.com/sirupsen/logrus"
)

var RunningError = errors.New("running")

type Builder struct {
	mutex    sync.Mutex
	option   *config.Option
	projects map[int64]bool
}

func New(option *config.Option) *Builder {
	return &Builder{
		mutex:    sync.Mutex{},
		option:   option,
		projects: make(map[int64]bool),
	}
}

func (b *Builder) cmds(dir string, project *requests.Project, tag, commit string) []string {
	cmds := []string{}

	cmds = append(cmds, fmt.Sprintf(`/bin/rm -fr "%s"`, dir))
	cmds = append(cmds, fmt.Sprintf(`mkdir -p "%s"`, dir))
	cmds = append(cmds, fmt.Sprintf(`cd "%s" && git clone -q "%s" "%s"`, dir, project.GitSSHURL, tag))
	cmds = append(cmds, fmt.Sprintf(`cd "%s/%s" && git checkout -b "%s"`, dir, tag, tag))
	cmds = append(cmds, fmt.Sprintf(`cd "%s/%s" && git reset --hard "%s"`, dir, tag, commit))

	return cmds
}

func (b *Builder) Build(project *requests.Project, tag string, commit, script string) (error, string, string) {
	defer func() {
		b.mutex.Lock()
		delete(b.projects, project.ID)
		b.mutex.Unlock()
	}()

	b.mutex.Lock()
	if _, ok := b.projects[project.ID]; ok {
		return RunningError, "", ""
	}
	b.projects[project.ID] = true
	b.mutex.Unlock()

	now := time.Now().Format("2006-01-02_15-04-05")

	dir, err := filepath.Abs(filepath.Join(b.option.BuildDir, strconv.FormatInt(project.ID, 10), tag, now))
	if err != nil {
		return err, "", ""
	}

	if err := utils.Mkdir(dir); err != nil {
		return err, "", ""
	}

	var resultBuffer bytes.Buffer

	defer func() {
		fmt.Println(resultBuffer.String())
	}()

	for _, cmd := range b.cmds(dir, project, tag, commit) {
		resultBuffer.WriteString(strings.Repeat("-", 20))
		resultBuffer.WriteString("\n")
		resultBuffer.WriteString(cmd)
		resultBuffer.WriteString("\n")
		output, err := utils.Run(cmd)
		resultBuffer.Write(output)
		resultBuffer.WriteString("\n")
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"cmd":   cmd,
				"error": err,
			}).Error("error command")
			return err, "", ""
		}
	}

	packageName := fmt.Sprintf("%s_%s.tar.gz", tag, now)
	buildDir, packageFile := filepath.Join(dir, tag), filepath.Join(dir, packageName)

	var scriptBuffer bytes.Buffer
	scriptFile := filepath.Join(dir, "build.sh")

	scriptBuffer.WriteString(fmt.Sprintf(`#!/bin/bash
BuildDir="%s"
PackageFile="%s"

Namespace="%s"
Name="%s"
TagName="%s"

cd "%s"
%s
	`, buildDir, packageFile, project.Namespace, project.Name, tag, buildDir, script))

	err = ioutil.WriteFile(scriptFile, scriptBuffer.Bytes(), os.ModePerm)

	if err != nil {
		return err, "", ""
	}

	output, err := utils.RunFile(scriptFile)
	resultBuffer.Write(output)
	resultBuffer.WriteString(strings.Repeat("-", 20))
	resultBuffer.WriteString("\n")
	resultBuffer.WriteString("file:\n")
	resultBuffer.WriteString(scriptFile)
	resultBuffer.WriteString("\n")
	resultBuffer.WriteString("script:\n")
	resultBuffer.Write(scriptBuffer.Bytes())
	resultBuffer.WriteString("\n")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file":   scriptFile,
			"script": scriptBuffer.String(),
			"error":  err,
		}).Error("error command")
		return err, "", ""
	}

	resultFile := filepath.Join(
		b.option.PackageDir,
		fmt.Sprintf("%d_%s_%s", project.ID, project.Namespace, project.Name),
		packageName,
	)
	return utils.CopyFile(packageFile, resultFile), resultBuffer.String(), resultFile
}
