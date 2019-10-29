package commands

import (
	"bobby/services"
	"github.com/urfave/cli"
	"log"
)

const BASE_NAME = "bobby-home"

func ServicesCreate(c *cli.Context) {

	///////// DEPLOY NEW VERSION OF IMAGE\

	name := c.Args().Get(0)
	if name == "" {
		log.Fatal("first arg name is required")
	}

	env := c.Args().Get(1)
	if env == "" {
		log.Fatal("second arg environments is required")
	}
	//url := c.Args().Get(1)
	//if url == "" {
	//	log.Fatal("second arg url is required")
	//}

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
	found := false
	for _, e := range dbc.Config.Environments {
		if e == env {
			found = true
			break
		}
	}
	if !found {

	}
	for _, s := range dbc.Config.Services {
		if s.Name == name {
			log.Fatal("A service already exist")
			return
		}
	}

	dbc.Config.Services = append(dbc.Config.Services, &services.Services{Name: name, Environment: env})

	//	file := fmt.Sprintf(`
	//apiVersion: networking.gke.io/v1beta1
	//kind: ManagedCertificate
	//metadata:
	//  name: %s-certificate
	//spec:
	//  domains:
	//    - %s
	//`, name, url)
	//	f, err := ioutil.TempFile("", "")
	//	f.Write([]byte(file))
	//	out, err := exec.Command("sh", "-c", "gcloud --project="+project.ProjectId+" container clusters get-credentials bobby-cluster --zone us-east1-c").Output()
	//	if err != nil {
	//		//step.Fail("Cannot get gcloud token")
	//		log.Fatal(err)
	//		return
	//	}
	//	fmt.Println(out)
	//	out, err = exec.Command("sh", "-c", "kubectl apply -f "+f.Name()).Output()
	//	if err != nil {
	//		//step.Fail("Cannot get gcloud token")
	//		log.Fatal(err)
	//		return
	//	}
	dbc.Save()
	step.Complete()
}
