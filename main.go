package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
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

// Model struct.
type Model struct {
	BaseURL    string
	Containers []Container
}

// Container struct.
type Container struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	ID     string   `json:"id"`
	Ports  []string `json:"ports"`
}

func newContainer(container types.Container, baseURL string) *Container {
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
		model := Model{
			BaseURL: getBaseURL(r, port),
		}
		containers, err := containersList(cli, model)
		if err != nil {
			returnError(w, errorTmp, err)
			return
		}

		model.Containers = containers

		indexTmp.Execute(w, model)
	}).Methods("GET")

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

func containersList(cli *client.Client, model Model) ([]Container, error) {
	var data []Container

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		data = append(data, *newContainer(container, model.BaseURL))
	}

	sort.Slice(data, func(i int, j int) bool {
		return data[i].Name < data[j].Name
	})

	return data, nil
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

	return newContainer(containers[0], baseURL), nil
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
