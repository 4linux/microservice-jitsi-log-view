package main


import (
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/gorilla/mux"
)


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


func last(w http.ResponseWriter, r *http.Request) {
	queryParams :=  r.URL.Query()
	fmt.Fprintf(w, "Last %s", queryParams["last"][0])
	// Trazer os ultimos N registros
}


func searchCourse(w http.ResponseWriter, r *http.Request) {
	queryParams :=  r.URL.Query()
	fmt.Fprintf(w, "Course %s", queryParams["courseid"][0])
	// Trazer todas as turmas do curso N
}


func searchClass(w http.ResponseWriter, r *http.Request) {
	queryParams :=  r.URL.Query()
	fmt.Fprintf(w, "Class %s", queryParams["classid"][0])
	// Trazer todas as aulas da turma N
}


func searchStudentEmail(w http.ResponseWriter, r *http.Request) {
	queryParams :=  r.URL.Query()
	fmt.Fprintf(w, "Student %s", queryParams["studentEmail"][0])
	// TODO regex for email
	// Trazer todos os registros do aluno@domain.tld
}


func searchLesson(w http.ResponseWriter, r *http.Request) {
	queryParams :=  r.URL.Query()
	fmt.Fprintf(w, "Class %s", queryParams["classid"][0])
	fmt.Fprintf(w, "Lesson")
	// Trazer todos os registros de aulas da turma N
}


func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", handler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck", healthcheck).Methods(http.MethodGet)
	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/logs", last).Methods("GET").Queries("last", "{last:[0-9]+}")
	api.HandleFunc("/logs", searchCourse).Methods("GET").Queries("courseid", "{last:[0-9]+}")
	api.HandleFunc("/logs", searchClass).Methods("GET").Queries("classid", "{last:[0-9]+}")
	api.HandleFunc("/logs", searchStudentEmail).Methods("GET").Queries("studentEmail", "{studentEmail}")
	api.HandleFunc("/logs", searchLesson).Methods("GET").Queries("classid", "{last:[0-9]+}")
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":8080", nil))
	// TODO Log requests
}
