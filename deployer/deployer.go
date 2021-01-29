package deployer

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
	"webhook/config"
	"webhook/domain/models"
	"webhook/domain/requests"
	"webhook/utils"

	"github.com/sirupsen/logrus"
)

var RunningError = errors.New("running")
var PackageNotFoundError = errors.New("package not found")

type Deployer struct {
	mutex    sync.Mutex
	option   *config.Option
	projects map[int64]bool
}

func New(option *config.Option) *Deployer {
	return &Deployer{
		mutex:    sync.Mutex{},
		option:   option,
		projects: make(map[int64]bool),
	}
}

func (d *Deployer) cmds(dir string, project *requests.Project, tag, commit string) []string {
	cmds := []string{}

	cmds = append(cmds, fmt.Sprintf(`/bin/rm -fr "%s"`, dir))
	cmds = append(cmds, fmt.Sprintf(`mkdir -p "%s"`, dir))
	cmds = append(cmds, fmt.Sprintf(`cd "%s" && git clone -q "%s" "%s"`, dir, project.GitHTTPURL, tag))
	cmds = append(cmds, fmt.Sprintf(`cd "%s/%s" && git checkout -b "%s"`, dir, tag, tag))
	cmds = append(cmds, fmt.Sprintf(`cd "%s/%s" && git reset --hard "%s"`, dir, tag, commit))

	return cmds
}

func (d *Deployer) Deploy(project *requests.Project, tag, packageFile string, hosts []*models.Host, script string) error {
	defer func() {
		d.mutex.Lock()
		delete(d.projects, project.ID)
		d.mutex.Unlock()
	}()

	d.mutex.Lock()
	if _, ok := d.projects[project.ID]; ok {
		return RunningError
	}
	d.projects[project.ID] = true
	d.mutex.Unlock()

	if ok := utils.FileExists(packageFile); !ok {
		return PackageNotFoundError
	}

	now := time.Now().Format("2006-01-02_15-04-05")

	dir, err := filepath.Abs(filepath.Join(d.option.DeployDir, strconv.FormatInt(project.ID, 10), tag, now))
	if err != nil {
		return err
	}

	if err := utils.Mkdir(dir); err != nil {
		return err
	}

	deployScriptName := "deploy.sh"
	deployScriptFile := filepath.Join(dir, deployScriptName)
	packageName := filepath.Base(packageFile)

	var scriptBuffer bytes.Buffer

	targetTempDir := fmt.Sprintf("/tmp/deploy/%d/%s/%s", project.ID, tag, now)

	scriptBuffer.WriteString(fmt.Sprintf(`#!/bin/bash
TempDir="%s"
PackageFile="%s"

Namespace="%s"
Name="%s"
TagName="%s"

cd "%s"
%s
	`, targetTempDir, filepath.Join(targetTempDir, packageName), project.Namespace, project.Name, tag, targetTempDir, script))

	err = ioutil.WriteFile(deployScriptFile, scriptBuffer.Bytes(), os.ModePerm)

	if err != nil {
		return err
	}

	for _, host := range hosts {
		d.DeployHost(host, packageFile, targetTempDir, deployScriptFile)
	}

	return nil
}

func (d *Deployer) DeployHost(host *models.Host, packageFile string, tempDir, deployScriptFile string) (error, string) {
	cmds := []string{}

	cmds = append(cmds, fmt.Sprintf(`ssh %s@%s -p %d -i %s "%s"`, host.User, host.Addr, host.Port, host.Key, fmt.Sprintf(`mkdir -p "%s"`, tempDir)))
	cmds = append(cmds, fmt.Sprintf(`scp -P %d -i %s "%s" %s@%s:"%s"`, host.Port, host.Key, packageFile, host.User, host.Addr, tempDir))
	cmds = append(cmds, fmt.Sprintf(`scp -P %d -i %s "%s" %s@%s:"%s"`, host.Port, host.Key, deployScriptFile, host.User, host.Addr, tempDir))
	cmds = append(cmds, fmt.Sprintf(`ssh %s@%s -p %d -i %s "%s"`, host.User, host.Addr, host.Port, host.Key, fmt.Sprintf(`bash "%s/%s"`, tempDir, filepath.Base(deployScriptFile))))
	cmds = append(cmds, fmt.Sprintf(`ssh %s@%s -p %d -i %s "%s"`, host.User, host.Addr, host.Port, host.Key, fmt.Sprintf(`/bin/rm -fr "%s"`, tempDir)))

	var resultBuffer bytes.Buffer

	defer func() {
		fmt.Println(resultBuffer.String())
	}()

	for _, cmd := range cmds {
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
			return err, resultBuffer.String()
		}
	}
	return nil, resultBuffer.String()
}
