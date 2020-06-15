package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

// Pensamentos
// domain.com/course/531/class/1/users
// domain.com/courses/531/class/1/users/13

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
	api.HandleFunc("/students", handler).Methods(http.MethodGet)
	api.HandleFunc("/courses", handler).Methods(http.MethodGet)
	api.HandleFunc("/class", handler).Methods(http.MethodGet)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
