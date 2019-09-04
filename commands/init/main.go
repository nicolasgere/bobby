package commands

import (
	"bobby/services"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/urfave/cli"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/servicemanagement/v1"
	"math/rand"
	"os/exec"
	"strconv"
	"time"
)

const BASE_NAME = "bobby-home"

func Init(c *cli.Context) {
	ctx := context.Background()

	/// STEP 0 Check dependency
	step := services.NewStepper("Check dependency")
	_, err := exec.Command("sh", "-c", "which gcloud").Output()
	if err != nil {
		step.Fail("Please install gcloud cli and login")
		return
	}
	_, err = exec.Command("sh", "-c", "which kubectl").Output()
	if err != nil {
		step.Fail("Please install kubectl with 'gcloud components install kubectl' ")
		return
	}
	step.Complete()

	/// STEP 1 CREATE NEW PROJECT OR USE ALREADY ONE
	step = services.NewStepper("Initializing bobby project")
	cloudResourceService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		step.FailWithError("cannot get cloud ressource manager", err)
	}
	project, err := services.GetBobbyProject()
	if err != nil {
		step.FailWithError("cannot not fetch project", err)
	}

	if project == nil {
		project = &cloudresourcemanager.Project{
			Name:      BASE_NAME,
			ProjectId: BASE_NAME + "-" + strconv.Itoa(rand.Intn(100000000)),
		}
		operation, err := cloudResourceService.Projects.Create(project).Do()
		if err != nil {
			step.FailWithError("Cannot create project", err)
			return
		}
		done := false
		for !done {
			result, _ := cloudResourceService.Operations.Get(operation.Name).Do()
			done = result.Done
			time.Sleep(5 * time.Second)
		}
	}
	step.Complete()
	fmt.Println("Using now project: " + project.ProjectId)

	//STEP BILLING CHECK
	step = services.NewStepper("Verifying billing")
	client, err := google.DefaultClient(ctx, cloudbilling.CloudPlatformScope)
	billingService, err := cloudbilling.New(client)
	if err != nil {
		step.FailWithError("Cannot get cloud billing service", err)
		return
	}
	info, err := billingService.Projects.GetBillingInfo("projects/" + project.ProjectId).Do()
	if err != nil {
		step.FailWithError("Cannot get billing info", err)
		return
	}

	if info.BillingEnabled == false {
		step.Fail(fmt.Sprintf("Please visit https://console.cloud.google.com/billing/linkedaccount?project=%s", project.ProjectId))
		return
	}
	step.Complete()

	//ENABLING API REGISTRY
	step = services.NewStepper("Enabling container registry api")
	serviceManagement, err := servicemanagement.NewService(ctx)
	if err != nil {
		step.FailWithError("cannot get service client", err)
		return
	}
	operation, err := serviceManagement.Services.Enable("containerregistry.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	if err != nil {
		step.FailWithError("cannot enable containerregistry", err)
		return
	}
	done := operation.Done
	for !done {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			step.FailWithError("Cannot get operation", err)
			return
		}
		done = op.Done
		time.Sleep(time.Second * 2)
	}
	step.Complete()

	step = services.NewStepper("Enabling container/kubernetes api")
	operation, err = serviceManagement.Services.Enable("container.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	if err != nil {
		step.FailWithError("cannot enable container/kubernetes", err)
		return
	}
	done = operation.Done
	for done == false {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			step.FailWithError("Cannot get operation", err)
			return
		}
		done = op.Done
		time.Sleep(time.Second * 1)
	}
	step.Complete()

	step = services.NewStepper("Enabling storage api")
	operation, err = serviceManagement.Services.Enable("storage-component.googleapis.com", &servicemanagement.EnableServiceRequest{
		ConsumerId: "project:" + project.ProjectId,
	}).Do()
	if err != nil {
		step.FailWithError("Enabling storage api", err)
		return
	}
	done = operation.Done
	for done == false {
		op, err := serviceManagement.Operations.Get(operation.Name).Do()
		if err != nil {
			step.FailWithError("Cannot get operation", err)
			return
		}
		done = op.Done
		time.Sleep(time.Second * 1)
	}
	step.Complete()

	//CHECK BUCKET CONFIG
	step = services.NewStepper("Initializing bobby bucket")
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		step.FailWithError("fail to create storage client", err)
	}
	bucketName := project.ProjectId + "-config"
	bucket := storageClient.Bucket(bucketName)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		if err := bucket.Create(ctx, project.ProjectId, nil); err != nil {
			step.FailWithError("cannot create bucket", err)
		}

	}
	step.Complete()
	fmt.Println("Use bucket: " + bucketName)

	//INIT CONFIG FILE
	step = services.NewStepper("Instantiate json config")
	dbC := services.DbConfig{}
	err = dbC.Init(project.ProjectId)
	if err != nil {
		step.FailWithError("cannot find bucket", err)
		return
	}
	err = dbC.Load()
	if err != nil {
		err = dbC.Initialize()
		if err != nil {
			step.FailWithError("cannot create config.json", err)
		}
	}
	step.Complete()

	//CHECK IF A CLUSTER EXIST
	step = services.NewStepper("Creating kubernetes cluster")
	containerService, err := container.NewService(ctx)
	if err != nil {
		step.FailWithError("Cannot have kub services", err)
		return
	}

	resp, err := containerService.Projects.Locations.Clusters.List("projects/" + project.ProjectId + "/locations/-").Do()
	if err != nil {
		step.FailWithError("Not able to list cluster", err)
		return
	}
	index := services.FindCluster(resp.Clusters, "bobby-cluster")
	var cluster *container.Cluster
	if index != -1 {
		cluster = resp.Clusters[index]
	} else {
		_, err := containerService.Projects.Locations.Clusters.Create("projects/"+project.ProjectId+"/locations/us-east1-c", &container.CreateClusterRequest{
			ProjectId: project.ProjectId,
			Zone:      "us-east1-c",
			Cluster: &container.Cluster{
				Name:             "bobby-cluster",
				Description:      "A cluster managed by bobby",
				InitialNodeCount: 1,
				MasterAuth: &container.MasterAuth{
					Password: RandStringRunes(50),
					Username: RandStringRunes(30),
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
			step.FailWithError("cannot create kub cluster", err)
			return
		}
		done := false
		for done == false {
			resp, err := containerService.Projects.Locations.Clusters.List("projects/" + project.ProjectId + "/locations/-").Do()
			if err != nil {
				step.FailWithError("cannot get operation", err)
			}
			i := services.FindCluster(resp.Clusters, "bobby-cluster")
			if i != -1 {
				done = true
				cluster = resp.Clusters[i]
			}
			time.Sleep(5 * time.Second)
		}
	}
	step.Complete()
	fmt.Println("User cluster " + cluster.Name)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
