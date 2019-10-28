package commands

import (
	"bobby/services"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"os"
)

func EnvironmentList(c *cli.Context) {

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

	if dbc.Config.Environments == nil {
		dbc.Config.Environments = []string{}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name"})

	for _, v := range dbc.Config.Environments {
		table.Append([]string{v})
	}

	table.Render()
}
