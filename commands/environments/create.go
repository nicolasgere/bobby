package commands

import (
	"bobby/services"
	"github.com/urfave/cli"
	"log"
)

func EnvironmentCreate(c *cli.Context) {

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
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
	step = services.NewStepper("Creating environments")
	if dbc.Config.Environments == nil {
		dbc.Config.Environments = []string{}
	}
	found := false
	for _, v := range dbc.Config.Environments {
		if v == name {
			found = true
		}
	}
	if found {
		step.Fail("Already exist: " + name)
	}
	dbc.Config.Environments = append(dbc.Config.Environments, name)
	dbc.Save()
	step.Complete()

}
