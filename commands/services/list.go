package commands

import (
	"bobby/services"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
)

func ServicesList(c *cli.Context) {

	ctx := context.Background()
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
	compute, _ := compute.NewService(ctx)

	//// GET CLUSTER
	step = services.NewStepper("Accessing kubernetes cluster")
	containerService, _ := container.NewService(ctx)
	resp, err := containerService.Projects.Locations.Clusters.List("projects/" + p.ProjectId + "/locations/-").Do()
	if err != nil {
		step.Fail("Not able to list cluster")
		log.Fatal(err)
		return
	}
	index := services.FindCluster(resp.Clusters, "bobby-cluster")
	if index == -1 {
		step.Fail("You dont have any bobby cluster ready")
		return
	}
	cluster := resp.Clusters[index]
	cert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		step.Fail(err.Error())
		return
	}

	kub, err := kubernetes.NewForConfig(&rest.Config{
		Username: cluster.MasterAuth.Username,
		Password: cluster.MasterAuth.Password,
		Host:     cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: cert,
		},
	})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()

	data := [][]string{}

	serv, err := kub.ExtensionsV1beta1().Ingresses("default").Get("bobby-ingress", v1.GetOptions{})

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
			endpoint = fmt.Sprintf("%s (%s)", s.Url, serv.Status.LoadBalancer.Ingress[0].IP)
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
