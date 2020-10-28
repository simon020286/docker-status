package main

import (
	"docker-status/utils"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
)

// Model struct.
type Model struct {
	BaseURL    string      `json:"-"`
	Containers []Container `json:"containers"`
	Services   []Compose   `json:"services"`
}

// Compose struct.
type Compose struct {
	Name       string       `json:"name"` // "com.docker.compose.project"
	ConfigFile string       `json:"-"`    // "com.docker.compose.project.config_files"
	ConfigDir  string       `json:"-"`    // "com.docker.compose.project.working_dir"
	Containers []*Container `json:"containers"`
	// "com.docker.compose.service" nome del servizio all'interno di compose
	// "com.docker.compose.container-number" ordine del container all'interno del servizio
	// "com.docker.compose.oneoff" ???
}

func newCompose(container *types.Container, baseURL string) *Compose {
	return &Compose{
		Name:       utils.ProjectName(container),
		ConfigFile: utils.ConfigFile(container),
		ConfigDir:  utils.ConfigDir(container),
		Containers: []*Container{
			newContainer(container, baseURL),
		},
	}
}

// Container struct.
type Container struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	ID     string   `json:"id"`
	Ports  []string `json:"ports"`
}

func newContainer(container *types.Container, baseURL string) *Container {
	var ports []string
	for i := 0; i < len(container.Ports); i++ {
		if container.Ports[i].PublicPort != 0 {
			ports = append(ports, fmt.Sprintf("%s:%d", baseURL, container.Ports[i].PublicPort))
		}
	}
	return &Container{
		ID:     container.ID,
		Name:   strings.Join(container.Names, " ")[1:],
		Status: container.State,
		Ports:  ports,
	}
}
