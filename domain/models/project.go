package models

type Project struct {
	ID           int64
	ProjectID    int64
	Name         string
	Decription   string
	WebURL       string
	GitSSHURL    string
	GitHTTPURL   string
	AutoBuild    bool
	BuildScript  string
	AutoDeploy   bool
	Deploy       string
	DeployHosts  []*Host
	DeployScript string
}
