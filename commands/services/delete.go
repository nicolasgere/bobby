package commands

import (
	"bobby/services"
	"context"
	"fmt"
	"github.com/urfave/cli"
	"google.golang.org/api/cloudresourcemanager/v1"

	"log"
	"strings"
)

func ServicesDelete(c *cli.Context) {
	ctx := context.Background()

	///////// DEPLOY NEW VERSION OF IMAGE\

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}

	cloudResourceService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		fmt.Println("Error during getting gcloud access")
		fmt.Println(err)
		return
	}
	responseProjects, _ := cloudResourceService.Projects.List().Do()
	var project *cloudresourcemanager.Project
	for _, p := range responseProjects.Projects {
		if p.LifecycleState == "ACTIVE" && strings.Contains(p.Name, BASE_NAME) {
			project = p
		}
	}
	if project == nil {
		fmt.Println("No bobby project found, did you init it?")
		return
	}

	dbC := services.DbConfig{}
	dbC.Init(project.ProjectId)
	dbC.Load()
	found := -1
	for i, s := range dbC.Config.Services {
		if s.Name == name {
			found = i
			break
		}
	}
	if found == -1 {
		fmt.Println("NOTf found")
		return
	}
	dbC.Config.Services = append(dbC.Config.Services[:found], dbC.Config.Services[found+1:]...)
	dbC.Save()

}
