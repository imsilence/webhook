package services

import (
	"webhook/domain/models"
)

func GetProject(project int64) *models.Project {
	/*
	   	return &models.Project{
	   		AutoBuild: true,
	   		BuildScript: `
	   go build .
	   if [[ $? != 0 ]]; then
	   	exit 1
	   fi
	   tar zvcf ${PackageFile} cmdb
	   		`,
	   	}
	*/

	return &models.Project{
		DeployHosts: []*models.Host{
			{
				Addr: "192.168.0.3",
				Port: 22,
				User: "root",
				Key:  "/root/.ssh/root",
			},
		},
		AutoBuild: true,
		BuildScript: `
docker build -t ${Namespace}/${Name}:${TagName} --build-arg VERSION=${TagName} .
docker image save ${Namespace}/${Name}:${TagName} >  ${PackageFile}
		`,
		AutoDeploy: true,
		DeployScript: `
docker image load < $PackageFile
docker container stop cmdb
docker container rm cmdb
docker container run -itd --rm --name cmdb -p10000:8888 ${Namespace}/${Name}:${TagName}
		`,
	}
}
