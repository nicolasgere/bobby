package commands

import (
	"fmt"
	"github.com/ericchiang/k8s"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/urfave/cli"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"tempv2/services"
	"time"
)

func Deploy(c *cli.Context) {

	target := "toto"
	image := "karthequian/helloworld:latest"

	p, err := services.GetBobbyProject()
	if err != nil {
		log.Fatal("Not able to found a project. Did you init a bobby project in gloud?")
		return
	}
	//TODO VERIFY PROJECT IS READY

	dbc, err := services.GetConfig(p.ProjectId)
	if err != nil {
		log.Fatal("Not able to load config")
		return
	}
	var service *services.Services
	for _, s := range dbc.Config.Services {
		if s.Name == target {
			service = &s
			break
		}
	}
	if service == nil {
		log.Fatal("Servie not found")
		return
	}
	out, err := exec.Command("sh", "-c", "gcloud --project="+p.ProjectId+" auth print-access-token").Output()
	if err != nil {
		log.Fatal("Cannot login to docker")
		return
	}
	pwd := strings.TrimSpace(string(out))
	cl, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal("Cannot login to docker")
		return
	}
	fmt.Println(pwd)
	cl.TagImage(image, docker.TagImageOptions{
		Repo: "us.gcr.io/" + p.ProjectId + "/" + service.Name,
		Tag:  strconv.Itoa(len(service.Versions)),
	})
	//err = cl.PushImage(docker.PushImageOptions{
	//	Name:     "us.gcr.io/" + p.ProjectId + "/" + service.Name,
	//	Tag:      strconv.Itoa(len(service.Versions)),
	//	Registry: "us.gcr.io",
	//}, docker.AuthConfiguration{
	//	Username: "oauth2accesstoken",
	//	Password: pwd,
	//})
	//if err != nil {
	//	log.Fatal("Cannot login to docker")
	//	return
	//}
	kub, err := kubernetes.NewForConfig(&rest.Config{
		Username: dbc.Config.Cluster.Username,
		Password: dbc.Config.Cluster.Password,
		Host:     "34.74.118.86 ",
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	replica := int32(1)
	kub.AppsV1().Deployments("default").Create(&v1.Deployment{
		Spec: v1.DeploymentSpec{
			Selector: &v12.LabelSelector{MatchLabels: map[string]string{"app": service.Name}},
			Replicas: &replica,
			Template: v13.PodTemplateSpec{
				Spec:v13.PodSpec{
					Containers: C
				}
			},
		},
	})
	service.Versions = append(service.Versions, time.Now())
}
