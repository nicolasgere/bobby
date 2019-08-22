package main

import (
	"github.com/urfave/cli"
	"log"
	"math/rand"
	"os"
	commands3 "tempv2/commands/deploy"
	commands "tempv2/commands/init"
	commands2 "tempv2/commands/services"

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
			Name:    "deploy",
			Aliases: []string{"d"},
			Usage:   "Deploy ",
			Action:  commands3.Deploy,
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
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
