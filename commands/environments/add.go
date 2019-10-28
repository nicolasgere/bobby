package commands

import (
	"bobby/services"
	"fmt"
	"github.com/urfave/cli"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func EnvironmentAddConfig(c *cli.Context) {
	environmentName := c.Args().Get(0)
	if environmentName == "" {
		log.Fatal("first arg environment name is required")
		return
	}
	configName := c.Args().Get(1)
	if configName == "" {
		log.Fatal("second arg config name is required")
		return
	}
	configValue := c.Args().Get(2)
	if configValue == "" {
		log.Fatal("second arg config value is required")
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
	step = services.NewStepper("Creating secret/config")
	sec, err := kub.CoreV1().Secrets("default").Get(fmt.Sprintf("%s-%s", environmentName, configName), v1.GetOptions{})
	if err != nil {
		sec.Namespace = "default"
		sec.Name = fmt.Sprintf("%s-%s", environmentName, configName)
		sec.StringData = map[string]string{
			"value": configValue,
		}
		sec, err = kub.CoreV1().Secrets("default").Create(sec)
		if err != nil {
			step.FailWithError("cannot create secret", err)
			return
		}
	} else {
		sec.Namespace = "default"
		sec.Name = fmt.Sprintf("%s-%s", environmentName, configName)
		sec.StringData = map[string]string{
			"value": configValue,
		}
		sec, err = kub.CoreV1().Secrets("default").Update(sec)
		if err != nil {
			step.FailWithError("cannot create secret", err)
			return
		}
	}

	step.Complete()

}
