package commands

import (
	"bobby/services"
	"context"
	"encoding/base64"
	docker "github.com/fsouza/go-dockerclient"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/urfave/cli"
	"google.golang.org/api/container/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func Deploy(c *cli.Context) {
	ctx := context.Background()
	target := c.Args().Get(0)
	if target == "" {
		log.Fatal("first arg service name is required")
		return
	}
	image := c.Args().Get(1)
	if image == "" {
		log.Fatal("second arg image url is required")
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

	/////// STEP 2 GET SERVICES
	step = services.NewStepper("Get service config")
	var service *services.Services
	for _, s := range dbc.Config.Services {
		if s.Name == target {
			service = s
			break
		}
	}
	if service == nil {
		step.Fail("Service not found")
		return
	}
	if service.Versions == nil {
		service.Versions = []services.Version{}
	}
	step.Complete()

	/////// STEP 3 TAGGING IMAGE
	step = services.NewStepper("Tagging new image")
	out, err := exec.Command("sh", "-c", "gcloud --project="+p.ProjectId+" auth print-access-token").Output()
	if err != nil {
		step.Fail("Cannot get gcloud token")
		log.Fatal(err)
		return
	}
	pwd := strings.TrimSpace(string(out))
	cl, err := docker.NewClientFromEnv()
	if err != nil {
		step.Fail("Cannot get docker client from env")
		log.Fatal(err)
		return
	}
	imageUrl := "us.gcr.io/" + p.ProjectId + "/" + service.Name
	imageVersion := strconv.Itoa(len(service.Versions))

	err = cl.TagImage(image, docker.TagImageOptions{
		Repo: imageUrl,
		Tag:  imageVersion,
	})
	if err != nil {
		step.FailWithError("cannot tag the image", err)
		return
	}
	step.Complete()

	//////// PUSHING NEW IMAGE
	step = services.NewStepper("Pushing new image " + imageUrl)
	err = cl.PushImage(docker.PushImageOptions{
		Name:     "us.gcr.io/" + p.ProjectId + "/" + service.Name,
		Tag:      strconv.Itoa(len(service.Versions)),
		Registry: "us.gcr.io",
	}, docker.AuthConfiguration{
		Username: "oauth2accesstoken",
		Password: pwd,
	})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()

	///////// DEPLOY NEW VERSION OF IMAGE\
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
	//HARDCODED FOR NOW
	replica := int32(1)

	step = services.NewStepper("Deploying")
	deployment, err := kub.AppsV1().Deployments("default").Get(service.Name, metav1.GetOptions{})
	if err != nil {
		deployment, err = kub.AppsV1().Deployments("default").Create(&appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: service.Name,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": service.Name}},
				Replicas: &replica,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   service.Name,
						Labels: map[string]string{"app": service.Name},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  service.Name,
								Image: imageUrl + ":" + imageVersion,
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										"cpu": resource.MustParse("1"),
										"memory": resource.MustParse("500Mi	"),
									},
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("0.25"),
										"memory": resource.MustParse("250Mi"),
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			step.Fail(err.Error())
			return
		}
	} else {
		deployment.Spec.Template.Spec.Containers[0].Image = imageUrl + ":" + imageVersion
		deployment, err = kub.AppsV1().Deployments("default").Update(deployment)

		if err != nil {
			step.Fail(err.Error())
			return
		}

	}
	for deployment.Status.AvailableReplicas != deployment.Status.Replicas {
		step.Info("Target: " + strconv.Itoa(int(deployment.Status.Replicas)) + "  Current: " + strconv.Itoa(int(deployment.Status.AvailableReplicas)))
		deployment, err = kub.AppsV1().Deployments("default").Get(service.Name, metav1.GetOptions{})
		time.Sleep(10 * time.Second)
	}
	step.Complete()

	step = services.NewStepper("Updating ingress")
	_, err = kub.CoreV1().Services("default").Get(service.Name, metav1.GetOptions{})
	if err != nil {
		_, err = kub.CoreV1().Services("default").Create(&corev1.Service{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: service.Name,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:     "",
						Protocol: "TCP",
						Port:     8080,
						TargetPort: intstr.IntOrString{
							IntVal: 80,
						},
					},
				},
				Selector: map[string]string{
					"app": service.Name,
				},
				Type: "NodePort",
			},
		})
		if err != nil {
			step.Fail(err.Error())
			return
		}

	}

	ir := []v1beta1.IngressRule{}
	certs := []string{}
	for _, s := range dbc.Config.Services {
		certs = append(certs, s.Name+"-certificate")
		ir = append(ir, v1beta1.IngressRule{
			Host: s.Url,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: s.Name,
								ServicePort: intstr.IntOrString{
									IntVal: 8080,
								},
							},
						},
					},
				},
			},
		})
	}

	_, err = kub.ExtensionsV1beta1().Ingresses("default").Update(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bobby-ingress",
			Annotations: map[string]string{
				"networking.gke.io/managed-certificates": strings.Join(certs, ","),
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: ir,
		},
	})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	step.Complete()

	step = services.NewStepper("Saving update")
	service.Versions = append(service.Versions, services.Version{Value: imageVersion, LastDeploy: time.Now()})
	dbc.Save()
	step.Complete()
}
