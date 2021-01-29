package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"webhook/builder"
	"webhook/config"
	"webhook/deployer"
	"webhook/domain/requests"
	"webhook/services"
	"webhook/utils"
)

func main() {
	addr := ":8888"
	token := "abc"

	packageDir, _ := filepath.Abs("./static/")

	utils.Mkdir(packageDir)

	option := &config.Option{
		BuildDir:   "/tmp/webhook/build/",
		DeployDir:  "/tmp/webhook/deploy/",
		PackageDir: packageDir,
	}

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetReportCaller(false)

	builder := builder.New(option)
	deployer := deployer.New(option)

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(packageDir))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gitlab-Token") != token {

			logrus.WithFields(logrus.Fields{
				"token": r.Header.Get("X-Gitlab-Token"),
			}).Error("error token")

			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var event requests.Event
		var buffer bytes.Buffer

		reader := io.TeeReader(r.Body, &buffer)

		if err := json.NewDecoder(reader).Decode(&event); err != nil || event.Name != "tag_push" {
			logrus.WithFields(logrus.Fields{
				"body": buffer.String(),
			}).Error("error event")

			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var e struct {
			Commit  string            `json:"after"`
			Ref     string            `json:"ref"`
			Project *requests.Project `json:"project"`
		}

		if err := json.NewDecoder(&buffer).Decode(&e); err != nil {
			logrus.WithFields(logrus.Fields{
				"body": buffer.String(),
			}).Error("error tag push event")

			w.WriteHeader(http.StatusBadRequest)
			return
		}

		project := services.GetProject(e.Project.ID)
		if project == nil {
			logrus.WithFields(logrus.Fields{
				"project": e.Project,
			}).Info("create project")
			return
		}

		if !project.AutoBuild {
			logrus.WithFields(logrus.Fields{
				"project": e.Project,
			}).Debug("project buiild auto is false")
			return
		}

		if e.Commit == "0000000000000000000000000000000000000000" {
			logrus.WithFields(logrus.Fields{
				"project": e.Project,
				"ref":     e.Ref,
			}).Debug("tag push delete event")

			w.WriteHeader(http.StatusNotModified)
			return
		}

		logrus.WithFields(logrus.Fields{
			"project": e.Project,
			"ref":     e.Ref,
			"commit":  e.Commit,
		}).Info("build project")

		// go func() {
		tagName := strings.TrimPrefix(e.Ref, "refs/tags/")
		err, _, packageFile := builder.Build(e.Project, tagName, e.Commit, project.BuildScript)
		if err == nil {

			logrus.WithFields(logrus.Fields{
				"project": e.Project,
				"ref":     e.Ref,
				"commit":  e.Commit,
			}).Info("deploy project")
			fmt.Println(deployer.Deploy(e.Project, tagName, packageFile, project.DeployHosts, project.DeployScript))
		}
		// }()

		fmt.Fprintf(w, "%d", time.Now().Unix())
	})

	mux.HandleFunc("/deploy/", func(w http.ResponseWriter, r *http.Request) {

		var e = struct {
			Commit  string            `json:"after"`
			Ref     string            `json:"ref"`
			Project *requests.Project `json:"project"`
		}{
			Commit: "",
			Ref:    "",
			Project: &requests.Project{
				ID:        0,
				Namespace: "silence",
				Name:      "cmdb",
			},
		}

		project := services.GetProject(e.Project.ID)
		packageFile := "/tmp/webhook/build/2/v4.0.0/2021-01-29_14-28-03/v4.0.0_2021-01-29_14-28-03.tar.gz"
		tagName := "v4.0.0"
		logrus.WithFields(logrus.Fields{
			"project": e.Project,
			"ref":     e.Ref,
			"commit":  e.Commit,
		}).Info("deploy project")
		fmt.Println(deployer.Deploy(e.Project, tagName, packageFile, project.DeployHosts, project.DeployScript))

		fmt.Fprintf(w, "%d", time.Now().Unix())
	})

	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		interrupt := make(chan os.Signal)
		signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

		<-interrupt

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		server.Shutdown(ctx)
		cancel()
	}()

	logrus.WithFields(logrus.Fields{
		"pid": os.Getpid(),
	}).Info("running...")
	logrus.Error(server.ListenAndServe())
}
