package main

import (
	"context"
	"docker-status/utils"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

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

	errorTmp := template.Must(template.ParseFiles("error.html"))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html, err := ioutil.ReadFile("index.html")
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		w.Write(html)
	}).Methods("GET")

	router.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
		model := Model{
			BaseURL: getBaseURL(r, port),
		}
		containers, services, err := containersList(cli, model)
		if err != nil {
			returnError(w, errorTmp, err)
			return
		}

		model.Containers = containers
		model.Services = services

		jsonString, err := json.Marshal(model)
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		w.Write(jsonString)
	})

	router.HandleFunc("/containers/stop/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ID := vars["id"]
		timeout := time.Minute
		err := cli.ContainerStop(context.Background(), ID, &timeout)
		w.Header().Add("content-type", "application/json")
		if err != nil {
			writeErrorResp(w, err)
			return
		}

		container, err := containerInfo(cli, ID, getBaseURL(r, port))
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		jsonString, err := json.Marshal(container)
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		w.Write(jsonString)
	})

	router.HandleFunc("/containers/start/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ID := vars["id"]
		err := cli.ContainerStart(context.Background(), ID, types.ContainerStartOptions{})
		w.Header().Add("content-type", "application/json")
		if err != nil {
			writeErrorResp(w, err)
			return
		}

		container, err := containerInfo(cli, ID, getBaseURL(r, port))
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		jsonString, err := json.Marshal(container)
		if err != nil {
			writeErrorResp(w, err)
			return
		}
		w.Write(jsonString)
	})

	router.HandleFunc("/containers/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ID := vars["id"]
		err := cli.ContainerRemove(context.Background(), ID, types.ContainerRemoveOptions{})
		w.Header().Add("content-type", "application/json")
		if err != nil {
			writeErrorResp(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":"+port, router))
}

func containersList(cli *client.Client, model Model) ([]Container, []Compose, error) {
	services := make(map[string]*Compose)
	var data []Container

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, nil, err
	}

	for _, container := range containers {
		projectName := utils.ProjectName(&container)
		if projectName == "" {
			data = append(data, *newContainer(&container, model.BaseURL))
			continue
		}
		compose, ok := services[projectName]
		if !ok {
			compose = newCompose(&container, model.BaseURL)
			services[projectName] = compose
		} else {
			compose.Containers = append(compose.Containers, newContainer(&container, model.BaseURL))
		}
	}

	sort.Slice(data, func(i int, j int) bool {
		return data[i].Name < data[j].Name
	})

	var compose []Compose

	for key := range services {
		compose = append(compose, *services[key])
	}

	return data, compose, nil
}

func containerInfo(cli *client.Client, id string, baseURL string) (*Container, error) {
	filt := filters.NewArgs()

	filt.Add("id", id)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filt})
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("Container not found")
	}

	return newContainer(&containers[0], baseURL), nil
}

func writeErrorResp(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
}

func returnError(w http.ResponseWriter, errTmp *template.Template, err error) {
	errTmp.Execute(w, err)
}

func getBaseURL(r *http.Request, port string) string {
	schema := "http"
	if r.TLS != nil {
		schema = "https"
	}
	return fmt.Sprintf("%s://%s", schema, strings.Replace(r.Host, fmt.Sprintf(":%s", port), "", 1))
}
