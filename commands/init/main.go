package commands

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/urfave/cli"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/servicemanagement/v1"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"tempv2/services"
	"time"
)

const BASE_NAME = "bobby-home"

func Init(c *cli.Context) {
	ctx := context.Background()

	//PROJECT CREATION
	fmt.Println("Create gcloud project...")
	cloudResourceService, _ := cloudresourcemanager.NewService(ctx)
	responseProjects, _ := cloudResourceService.Projects.List().Do()
	var project *cloudresourcemanager.Project
	for _, p := range responseProjects.Projects {
		if p.LifecycleState == "ACTIVE" && strings.Contains(p.Name, BASE_NAME) {
			project = p
		}
	}
	if project == nil {
		project = &cloudresourcemanager.Project{
			Name:      BASE_NAME,
			ProjectId: BASE_NAME + "-" + strconv.Itoa(rand.Intn(100000000)),
		}
		fmt.Println("Creating " + project.ProjectId)
		operation, err := cloudResourceService.Projects.Create(project).Do()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Something happens during project creation...")
		}
		done := false
		for done {
			result, _ := cloudResourceService.Operations.Get(operation.Name).Do()
			fmt.Println(operation.Name)
			fmt.Println(operation.Done)
			done = result.Done
			time.Sleep(5 * time.Second)
		}
	}
	fmt.Println("Use project: " + project.ProjectId)

	//BILLING CHECK
	client, err := google.DefaultClient(ctx, cloudbilling.CloudPlatformScope)
	billingService, err := cloudbilling.New(client)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Something happens during billing check")
		return
	}
	info, err := billingService.Projects.GetBillingInfo("projects/" + project.ProjectId).Do()
	if err != nil {
		fmt.Println(err)
		fmt.Println("Something happens during billing check")
		return
	}
	if info.BillingEnabled == false {
		fmt.Println("Please visit blabla and enable billing")
		return
	}

	//ENABLING API REGISTRY
	serviceManagement, err := servicemanagement.NewService(ctx)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Something happens during enabling api")
		return
	}
	operation, err := serviceManagement.Services.Enable("containerregistry.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	done := operation.Done
	for done == false {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Something happens during enabling api container registry")
			return
		}
		done = op.Done
		time.Sleep(time.Second * 1)
	}
	fmt.Println("Container registry enable")

	//ENABLING API KUBERNETES
	operation, err = serviceManagement.Services.Enable("container.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	done = operation.Done
	for done == false {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Something happens during enabling api kubernetes")
			return
		}
		done = op.Done
		time.Sleep(time.Second * 1)
	}
	fmt.Println("kubernetes enable")

	//ENABLING STORAGE
	operation, err = serviceManagement.Services.Enable("storage-component.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	done = operation.Done
	for done == false {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Something happens during enabling storage api")
			return
		}
		done = op.Done
		time.Sleep(time.Second * 1)
	}
	fmt.Println("storage enable")

	//CHECK BUCKET CONFIG
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	bucketName := project.ProjectId + "-config"
	bucket := storageClient.Bucket(bucketName)
	_, err = bucket.Attrs(ctx)

	if err != nil {
		if err := bucket.Create(ctx, project.ProjectId, nil); err != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}

	}
	fmt.Println("use config bucket: " + bucketName)

	//INIT CONFIG FILE
	rc, err := bucket.Object("config.json").NewReader(ctx)
	dbC := services.DbConfig{}
	dbC.Init(project.ProjectId)
	if err != nil {
		dbC.Initialize()
		return
	}
	defer rc.Close()
	fmt.Println("Config file created")

	//CHECK IF A CLUSTER EXIST
	containerService, _ := container.NewService(ctx)
	resp, err := containerService.Projects.Locations.Clusters.List("projects/" + project.ProjectId + "/locations/-").Do()
	if err != nil {
		log.Fatal("WOW somehint happens", err)
	}
	dbC.Config.Cluster.Username = RandStringRunes(30)
	dbC.Config.Cluster.Password = RandStringRunes(50)
	dbC.Save()
	if len(resp.Clusters) == 0 || findCluster(resp.Clusters, "bobby-cluster") == -1 {
		_, err := containerService.Projects.Locations.Clusters.Create("projects/"+project.ProjectId+"/locations/us-east1-c", &container.CreateClusterRequest{
			ProjectId: project.ProjectId,
			Zone:      "us-east1-c",
			Cluster: &container.Cluster{
				Name:             "bobby-cluster",
				Description:      "A cluster managed by bobby",
				InitialNodeCount: 1,
				MasterAuth: &container.MasterAuth{
					Password: dbC.Config.Cluster.Password,
					Username: dbC.Config.Cluster.Username,
				},
				NodeConfig: &container.NodeConfig{
					MachineType: "n1-standard-2",
					OauthScopes: []string{
						"https://www.googleapis.com/auth/devstorage.read_only",
						"https://www.googleapis.com/auth/logging.write",
						"https://www.googleapis.com/auth/monitoring",
						"https://www.googleapis.com/auth/service.management.readonly",
						"https://www.googleapis.com/auth/servicecontrol",
						"https://www.googleapis.com/auth/trace.append",
					},
				},
			},
		}).Do()
		if err != nil {
			log.Fatal("Something happens ", err)
		}
		done := false
		for done == false {
			resp, err := containerService.Projects.Locations.Clusters.List("projects/" + project.ProjectId + "/locations/-").Do()
			if err != nil {
				log.Fatal("WOW somehint happens", err)
			}
			i := findCluster(resp.Clusters, "bobby-cluster")
			if i != -1 {
				done = true
			}
			time.Sleep(5 * time.Second)
		}
		if err != nil {

		}
	}
	fmt.Println("Kubernetes ready")

}

func findCluster(s []*container.Cluster, name string) int {
	for i, item := range s {
		if item.Status == "RUNNING" && strings.Contains(item.Name, name) {
			return i
		}
	}
	return -1
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
