package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

// Pensamentos
// Ultimos registros = v1/logs?last=25 -> Retorna os ultimos 25 registros
// Buscar por curso = v1/logs?courseid=2 -> Retorna as aulas do curso com ID 2
// Buscar por turma = v1/logs?groupid=8 -> Retorna as aulas do curso com ID 2
// Buscar por aluno = v1/logs?student=alfa@domain.tld -> Retorna as aulas do curso com ID 2

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "microservice-jitsi-log-view")
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	name, err := os.Hostname()
	if err != nil {
	  log.Panic(err)
	}
	fmt.Fprintf(w, "Awake and alive from %s", name)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler).Methods(http.MethodGet)
	r.HandleFunc("/healthcheck", healthcheck).Methods(http.MethodGet)
	api := r.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/logs", handler).Methods("GET")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
