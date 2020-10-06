package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

// Model struct.
type Model struct {
	BaseURL    string
	Containers []Container
}

// Container struct.
type Container struct {
	Name   string
	Status string
	ID     string
	Ports  []string
}

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8860"
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter().StrictSlash(true)

	funcMap := template.FuncMap{
		"SafeURL": func(s string) template.URL {
			return template.URL(s)
		},
	}

	indexTmp := template.New("index.html").Funcs(funcMap)

	indexTmp = template.Must(indexTmp.ParseFiles("index.html"))

	errorTmp := template.Must(template.ParseFiles("error.html"))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		schema := "http"
		if r.TLS != nil {
			schema = "https"
		}
		model := Model{
			BaseURL: fmt.Sprintf("%s://%s", schema, strings.Replace(r.Host, fmt.Sprintf(":%s", port), "", 1)),
		}
		containers, err := containersList(cli, model)
		if err != nil {
			returnError(w, errorTmp, err)
			return
		}

		model.Containers = containers

		indexTmp.Execute(w, model)
	}).Methods("GET")

	router.HandleFunc("/stop/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ID := vars["id"]
		timeout := time.Minute
		err := cli.ContainerStop(context.Background(), ID, &timeout)
		w.Header().Add("content-type", "application/json")
		if err != nil {
			writeErrorResp(w, err)
			return
		}

		w.Write([]byte(`{"status": "exited"}`))
	})

	router.HandleFunc("/start/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ID := vars["id"]
		err := cli.ContainerStart(context.Background(), ID, types.ContainerStartOptions{})
		w.Header().Add("content-type", "application/json")
		if err != nil {
			writeErrorResp(w, err)
			return
		}

		w.Write([]byte(`{"status": "running"}`))
	})

	log.Fatal(http.ListenAndServe(":"+port, router))
}

func containersList(cli *client.Client, model Model) ([]Container, error) {
	var data []Container

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		var ports []string
		for i := 0; i < len(container.Ports); i++ {
			if container.Ports[i].PublicPort != 0 {
				ports = append(ports, fmt.Sprintf("%s:%d", model.BaseURL, container.Ports[i].PublicPort))
			}
		}
		data = append(data, Container{
			ID:     container.ID,
			Name:   strings.Join(container.Names, " ")[1:],
			Status: container.State,
			Ports:  ports,
		})
	}

	sort.Slice(data, func(i int, j int) bool {
		return data[i].Name < data[j].Name
	})

	return data, nil
}

func writeErrorResp(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
}

func returnError(w http.ResponseWriter, errTmp *template.Template, err error) {
	errTmp.Execute(w, err)
}
