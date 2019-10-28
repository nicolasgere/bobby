package commands

import (
	"bobby/services"
	"fmt"
	"github.com/urfave/cli"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"log"
	"strconv"
	"time"
)

func ServicesScale(c *cli.Context) {

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}

	count_s := c.Args().Get(1)
	count_64, err := strconv.Atoi(count_s)
	if err != nil {
		log.Fatal("second arg dyno count is required")
	}
	count := int32(count_64)

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
	step = services.NewStepper("Accessing kubernetes cluster")
	kub, err := services.GetCluster(p.ProjectId)
	if err != nil {
		step.Fail(err.Error())
		return
	}

	found := -1
	for i, s := range dbc.Config.Services {
		if s.Name == name {
			found = i
			break
		}
	}
	if found == -1 {
		step.Fail("Didnt found services")
		return
	}

	step.Complete()
	step = services.NewStepper("Scaling service")
	service := dbc.Config.Services[found]

	dep, err := kub.AppsV1().Deployments("default").Get(service.Name, v1.GetOptions{})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	old_count := *dep.Spec.Replicas
	dep.Spec.Replicas = &count
	dep, err = kub.AppsV1().Deployments("default").Update(dep)
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()
	over := false
	try := 0
	for !over && try < 2 {
		time.Sleep(10 * time.Second)
		dep, err = kub.AppsV1().Deployments("default").Get(service.Name, v1.GetOptions{})
		if err != nil {
			step.Fail(err.Error())
			return
		}
		if dep.Status.AvailableReplicas == dep.Status.Replicas {
			over = true
			continue
		}
		labelSelector := v1.LabelSelector{MatchLabels: map[string]string{"app": service.Name}}
		pods, _ := kub.CoreV1().Pods("default").List(v1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		})
		available := 0
		pending := 0
		reason := ""
		for _, p := range pods.Items {

			switch p.Status.Phase {
			case "Running":
				{
					available++

				}
			case "Pending":
				{
					pending++
					reason = p.Status.Conditions[0].Reason
					if reason == "Unschedulable" {
						try = 100
					}

				}
			}

		}
		try++
		fmt.Printf("/ Scaling ongoing: Available:%d Pending:%d Reason: %s \n", available, pending, reason)
	}
	if over == false {
		step = services.NewStepper(fmt.Sprintf("Rollback scaling to %d", old_count))
		dep.Spec.Replicas = &old_count
		dep, err = kub.AppsV1().Deployments("default").Update(dep)
		if err != nil {
			step.Fail(err.Error())
			return
		}
		step.Complete()

	}
}
