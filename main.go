package main

import (
	commands3 "bobby/commands/deploy"
	commands "bobby/commands/init"
	commands2 "bobby/commands/services"
	"github.com/urfave/cli"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   "Init a new gcp project",
			Action:  commands.Init,
		},
		{
			Name:      "deploy",
			Aliases:   []string{"d"},
			Usage:     "Deploy ",
			ArgsUsage: "[SERVICE_NAME] [DOCKER_IMAGE]",
			Action:    commands3.Deploy,
		},

		{
			Name:    "services",
			Aliases: []string{"s"},
			Usage:   "Manage services",
			Subcommands: []cli.Command{
				{
					Name:    "create",
					Aliases: []string{"c"},

					Usage:     "Create a new services",
					ArgsUsage: "[NAME] [URL]",
					Action:    commands2.ServicesCreate,
				},
				{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "List all services",
					Action:  commands2.ServicesList,
				},
				{
					Name:      "delete",
					Aliases:   []string{"d"},
					ArgsUsage: "[NAME]",
					Usage:     "Delete service",
					Action:    commands2.ServicesDelete,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
