package commands

import (
	"bobby/services"
	"context"
	"fmt"
	"github.com/urfave/cli"
	"google.golang.org/api/cloudresourcemanager/v1"
	"io/ioutil"
	"os/exec"

	"log"
	"strings"
)

const BASE_NAME = "bobby-home"

func ServicesCreate(c *cli.Context) {
	ctx := context.Background()

	///////// DEPLOY NEW VERSION OF IMAGE\

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}
	url := c.Args().Get(1)
	if url == "" {
		log.Fatal("second arg url is required")
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
	for _, s := range dbC.Config.Services {
		if s.Name == name || s.Url == url {
			log.Fatal("A service already exist")
			return
		}
	}

	dbC.Config.Services = append(dbC.Config.Services, &services.Services{Url: url, Name: name})

	file := fmt.Sprintf(`
apiVersion: networking.gke.io/v1beta1
kind: ManagedCertificate
metadata:
  name: %s-certificate
spec:
  domains:
    - %s
`, name, url)
	f, err := ioutil.TempFile("", "")
	f.Write([]byte(file))
	out, err := exec.Command("sh", "-c", "gcloud --project="+project.ProjectId+" container clusters get-credentials bobby-cluster --zone us-east1-c").Output()
	if err != nil {
		//step.Fail("Cannot get gcloud token")
		log.Fatal(err)
		return
	}
	fmt.Println(out)
	out, err = exec.Command("sh", "-c", "kubectl apply -f "+f.Name()).Output()
	if err != nil {
		//step.Fail("Cannot get gcloud token")
		log.Fatal(err)
		return
	}
	dbC.Save()

}
