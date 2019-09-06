package services

import (
	"context"
	"encoding/base64"
	"errors"
	"google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetCluster(projectId string) (kub *kubernetes.Clientset, err error) {
	ctx := context.Background()
	containerService, _ := container.NewService(ctx)
	resp, err := containerService.Projects.Locations.Clusters.List("projects/" + projectId + "/locations/-").Do()
	if err != nil {
		return
	}
	index := FindCluster(resp.Clusters, "bobby-cluster")
	if index == -1 {
		err = errors.New("You dont have any bobby cluster ready")
		return
	}
	cluster := resp.Clusters[index]
	cert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return
	}

	kub, err = kubernetes.NewForConfig(&rest.Config{
		Username: cluster.MasterAuth.Username,
		Password: cluster.MasterAuth.Password,
		Host:     cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: cert,
		},
	})
	return
}
