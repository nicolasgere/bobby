package commands

import (
	"bobby/services"
	"context"
	"encoding/base64"
	"github.com/urfave/cli"
	"google.golang.org/api/container/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"log"
)

func ServicesDelete(c *cli.Context) {
	ctx := context.Background()

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

	step = services.NewStepper("Deleting services")
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
	///////// DEPLOY NEW VERSION OF IMAGE\
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
	err = kub.CoreV1().Services("default").Delete(dbc.Config.Services[found].Name, &v1.DeleteOptions{})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	dbc.Config.Services = append(dbc.Config.Services[:found], dbc.Config.Services[found+1:]...)
	dbc.Save()
	step.Complete()

}
