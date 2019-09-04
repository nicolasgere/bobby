package services

import (
	"context"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/container/v1"
	"strings"
)

func GetBobbyProject() (project *cloudresourcemanager.Project, err error) {
	ctx := context.Background()
	var cloudResourceService *cloudresourcemanager.Service
	cloudResourceService, err = cloudresourcemanager.NewService(ctx)
	if err != nil {
		return
	}
	var responseProjects *cloudresourcemanager.ListProjectsResponse
	responseProjects, err = cloudResourceService.Projects.List().Do()
	if err != nil {
		return
	}
	for _, p := range responseProjects.Projects {
		if p.LifecycleState == "ACTIVE" && strings.Contains(p.Name, "bobby-home") {
			project = p
		}
	}
	return
}

func FindCluster(s []*container.Cluster, name string) int {
	for i, item := range s {
		if item.Status == "RUNNING" && strings.Contains(item.Name, name) {
			return i
		}
	}
	return -1
}
