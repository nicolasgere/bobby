package commands

import (
	"bobby/services"
	"fmt"
	"github.com/urfave/cli"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type node struct {
	total_cpu         int
	total_memory      int
	use_cpu_system    int
	use_memory_system int
	use_memory_bobby  int
	use_cpu_bobby     int
}

func ClusterInfo(c *cli.Context) {

	/////////  STEP 1 GET CONFIG
	step := services.NewStepper("Loading config")
	p, err := services.GetBobbyProject()
	if err != nil {
		step.Fail("Not able to found a project. Did you init a bobby project in gloud?")
		return
	}
	//TODO VERIFY PROJECT IS READY
	_, err = services.GetConfig(p.ProjectId)
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
	step = services.NewStepper("Fetching information")
	ls, err := kub.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		step.Fail(err.Error())
		return
	}
	t := map[string]*node{}
	node_count := 0
	for _, n := range ls.Items {
		node_count++
		t[n.ObjectMeta.Name] = &node{
			total_cpu:         int(n.Status.Allocatable.Cpu().ScaledValue(-3)),
			total_memory:      int(n.Status.Allocatable.Memory().Value()),
			use_cpu_system:    0,
			use_memory_system: 0,
			use_cpu_bobby:     0,
			use_memory_bobby:  0,
		}
	}
	pod_current := 0
	pod_available := 0

	ls1, err := kub.CoreV1().Pods("").List(v1.ListOptions{})

	for _, n := range ls1.Items {
		node := t[n.Spec.NodeName]
		if n.ObjectMeta.Namespace == "default" {
			for _, c := range n.Spec.Containers {
				pod_current++
				if c.Resources.Requests.Cpu() != nil {
					node.use_cpu_bobby = node.use_cpu_bobby + int(c.Resources.Requests.Cpu().ScaledValue(-3))
				}
				if c.Resources.Requests.Memory() != nil {
					node.use_memory_bobby = node.use_memory_bobby + int(c.Resources.Requests.Memory().ScaledValue(-9))
				}
			}
		} else {
			for _, c := range n.Spec.Containers {
				if c.Resources.Requests.Cpu() != nil {
					node.use_cpu_system = node.use_cpu_system + int(c.Resources.Requests.Cpu().ScaledValue(-3))
				}
				if c.Resources.Requests.Memory() != nil {
					node.use_memory_system = node.use_memory_system + int(c.Resources.Requests.Memory().Value())
				}
			}
		}
	}

	total_mem := 0
	total_cpu := 0

	for _, n := range t {
		total_mem = total_mem + n.total_memory
		total_cpu = total_cpu + n.total_cpu
		//memory_available := n.total_memory - n.use_memory_system - n.use_memory_bobby

		cpu_available := n.total_cpu - n.use_cpu_system - n.use_cpu_bobby
		pod_possible := cpu_available / 250
		pod_available = pod_available + pod_possible
	}
	step.Complete()
	fmt.Printf("Dyno capacity: %d (cluster node %d, total cpu %.2f, total memory %s)  \n", pod_current+pod_available, node_count, float64(total_cpu)/float64(1000), ByteCountBinary(int64(total_mem)))
	fmt.Printf("Used Dyno %d Free Dyno %d", pod_current, pod_available)

}
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
