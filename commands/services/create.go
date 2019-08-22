package commands

import (
	"context"
	"github.com/urfave/cli"
	"google.golang.org/api/cloudresourcemanager/v1"
	"log"
	"strings"

	"tempv2/services"
)

const BASE_NAME = "bobby-home"

func ServicesCreate(c *cli.Context) {
	ctx := context.Background()
	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}
	url := c.Args().Get(1)
	if url == "" {
		log.Fatal("second arg url is required")
	}
	cloudResourceService, _ := cloudresourcemanager.NewService(ctx)
	responseProjects, _ := cloudResourceService.Projects.List().Do()
	var project *cloudresourcemanager.Project
	for _, p := range responseProjects.Projects {
		if p.LifecycleState == "ACTIVE" && strings.Contains(p.Name, BASE_NAME) {
			project = p
		}
	}
	dbC := services.DbConfig{}
	dbC.Init(project.ProjectId)
	dbC.Load()
	for _, s := range dbC.Config.Services {
		if s.Name == name || s.Url == url {
			log.Fatal("A service already exist")
			return
		}
	}

	dbC.Config.Services = append(dbC.Config.Services, services.Services{Url: url, Name: name})
	dbC.Save()
}
