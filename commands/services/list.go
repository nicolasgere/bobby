package commands

import (
	"bobby/services"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func ServicesList(c *cli.Context) {

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
	step.Complete()

	data := [][]string{}

	serv, err := kub.ExtensionsV1beta1().Ingresses("default").Get("bobby-ingress", v1.GetOptions{})
	if err == nil {
		fmt.Printf("Endpoint IP: %s \n", serv.Status.LoadBalancer.Ingress[0].IP)
	}
	for _, s := range dbc.Config.Services {
		if s.Versions == nil {
			s.Versions = []services.Version{}
		}
		version := ""
		lastDeploy := ""
		status := ""
		endpoint := ""
		if len(s.Versions) != 0 {
			version = s.Versions[len(s.Versions)-1].Value
			lastDeploy = s.Versions[len(s.Versions)-1].LastDeploy.Format("2006-01-02 15:04:05")

			deployment, err := kub.AppsV1().Deployments("default").Get(s.Name, v1.GetOptions{})
			if err != nil {
				continue
			}
			endpoint = fmt.Sprintf("%s", s.Url)
			status = fmt.Sprintf("%d / %d (%d)", deployment.Status.ReadyReplicas, deployment.Status.Replicas, deployment.Status.UnavailableReplicas)
		}
		data = append(data, []string{
			"web", status, s.Name, endpoint, version, lastDeploy,
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Status", "Name", "Url", "Version", "Last deploy"})

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}
