package commands

import (
	"bobby/services"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"strings"
)

func EnvironmentDetail(c *cli.Context) {
	environmentName := c.Args().Get(0)
	if environmentName == "" {
		log.Fatal("first arg environment name is required")
		return
	}

	/////////  STEP 1 GET CONFIG
	step := services.NewStepper("Loading config")
	p, err := services.GetBobbyProject()
	if err != nil {
		step.Fail("Not able to found a project. Did you init a bobby project in gloud?")
		return
	}
	//TODO VERIFY PROJECT IS READY
	dbc, err := services.GetConfig(p.ProjectId)
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()

	//// GET CLUSTER
	step = services.NewStepper("Accecing cluster")
	if dbc.Config.Environments == nil {
		dbc.Config.Environments = []string{}
	}
	found := false
	for _, v := range dbc.Config.Environments {
		if v == environmentName {
			found = true
		}
	}
	if !found {
		step.Fail("Do not exist: " + environmentName)
	}
	kub, err := services.GetCluster(p.ProjectId)
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()
	step = services.NewStepper("Get secret/config")
	secrets, err := kub.CoreV1().Secrets("default").List(v1.ListOptions{})
	step.Complete()
	fmt.Println("Environment: " + environmentName)
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Name", "Value"})
	for _, s := range secrets.Items {
		if strings.Contains(s.Name, "default-token-") {
			continue
		}
		s.Name = strings.Replace(s.Name, environmentName+"-", "", 1)
		for _, s1 := range s.Data {
			table.Append([]string{s.Name, string(s1)})
		}
	}
	table.Render()
}
