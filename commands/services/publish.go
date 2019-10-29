package commands

import (
	"bobby/services"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"log"
	"os/exec"
	"strings"
)

func ServicesPublish(c *cli.Context) {

	///////// DEPLOY NEW VERSION OF IMAGE\

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}

	url := c.Args().Get(1)
	if url == "" {
		log.Fatal("second arg url is required")
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
	step = services.NewStepper("Creating service")
	found := -1
	for i, e := range dbc.Config.Services {
		if e.Name == name {
			found = i
			break
		}
	}
	if found == -1 {
		step.Fail("Service do not exist")
	}
	service := dbc.Config.Services[found]
	service.Url = url
	file := fmt.Sprintf(`
apiVersion: networking.gke.io/v1beta1
kind: ManagedCertificate
metadata:
  name: %s-certificate
spec:
  domains:
    - %s
`, name, url)
	f, err := ioutil.TempFile("", "")
	f.Write([]byte(file))
	out, err := exec.Command("sh", "-c", "gcloud --project="+p.ProjectId+" container clusters get-credentials bobby-cluster --zone us-east1-c").Output()
	if err != nil {
		fmt.Println(out)
		step.Fail("Cannot get gcloud token")
		log.Fatal(err)
		return
	}
	out, err = exec.Command("sh", "-c", "kubectl apply -f "+f.Name()).Output()
	if err != nil {
		fmt.Println(out)

		//step.Fail("Cannot get gcloud token")
		log.Fatal(err)
		return
	}
	step = services.NewStepper("Updating ingress")
	kub, err := services.GetCluster(p.ProjectId)
	if err != nil {
		step.Fail(err.Error())
		return
	}
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
	serv, err := kub.ExtensionsV1beta1().Ingresses("default").Get("bobby-ingress", metav1.GetOptions{})
	if serv != nil || err != nil {
		_, err = kub.ExtensionsV1beta1().Ingresses("default").Create(&v1beta1.Ingress{
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
	} else {
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
	}

	step.Complete()
	dbc.Save()
	step.Complete()
}
