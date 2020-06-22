package main


import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/http"
	"os"
	"time"
)


var URI_MONGODB string
var DATABASE string
var COLLECTION string


type Jitsilog struct {
	Sala       string `json:"sala"`
	Curso      string `json:"curso"`
	Turma      string `json:"turma"`
	Aluno      string `json:"aluno"`
	Jid        string `json:"jid"`
	Email      string `json:"email"`
	Timestramp int    `json:"timestamp"`
	Action     string `json:"action"`
}


func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true)
	if len(os.Getenv("URI_MONGODB")) > 0 {
		URI_MONGODB = os.Getenv("URI_MONGODB")
	} else {
		URI_MONGODB = "mongodb://localhost:27017"
	}
	if len(os.Getenv("DATABASE")) > 0 {
		DATABASE = os.Getenv("DATABASE")
	} else {
		DATABASE = "jitsilog"
	}
	if len(os.Getenv("COLLECTION")) > 0 {
		COLLECTION = os.Getenv("COLLECTION")
	} else {
		COLLECTION = "logs"
	}
	log.Debug("microservice-jitsi-log-view init")
	log.WithFields(log.Fields{
		"URI": URI_MONGODB,
		"DATABASE": DATABASE,
		"COLLECTION": COLLECTION}).Info("Database Connection Info")
}


func GetClient() *mongo.Client {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		log.Fatal("Failed to create the client ",err)
	}
	return client
}


func getLogs(size string, filter bson.M) []*Jitsilog {
	client := GetClient()
	err := client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal("Failed to connect to database!", err)
	} else {
		log.Info("Connected!")
	}
	log.Info("Dataset row limit ", size)
	var jitsilogs []*Jitsilog
	collection := client.Database(DATABASE).Collection(COLLECTION)
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal("Error on finding the documents ", err)
	}
	for cur.Next(context.TODO()) {
		var jitsilog Jitsilog
		err = cur.Decode(&jitsilog)
		if err != nil {
			log.Fatal("Error on decoding the document ", err)
		}
		jitsilogs = append(jitsilogs, &jitsilog)
	}
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal("Failed to disconnect from database!",err)
	}
	log.Info("Connection to MongoDB closed.")
	return jitsilogs
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "microservice-jitsi-log-view")
}

func checkHealth(w http.ResponseWriter, r *http.Request) {
	name, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}
	fmt.Fprintf(w, "Awake and alive from %s", name)
}

func latestLogs(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	var jitsilogs []*Jitsilog
	jitsilogs = getLogs(queryParams["last"][0],bson.M{})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jitsilogs)
	// Trazer os ultimos N registros
}

func searchCourse(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	fmt.Fprintf(w, "Course %s", queryParams["courseid"][0])
	// Trazer todas as turmas do curso N
}

func searchClass(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	fmt.Fprintf(w, "Class %s", queryParams["classid"][0])
	// Trazer todos os registros de aulas da turma N
}

func searchLesson(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	fmt.Fprintf(w, "Class %s", queryParams["classid"][0])
	// Trazer todas os registros de aula da turma N
}

func searchStudentEmail(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	fmt.Fprintf(w, "Student %s", queryParams["studentEmail"][0])
	// Trazer todos os registros do aluno@domain.tld
}


func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck", checkHealth).Methods(http.MethodGet)
	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/logs", latestLogs).Methods("GET").Queries("last", "{last:[0-9]+}")
	api.HandleFunc("/logs", searchCourse).Methods("GET").Queries("courseid", "{courseid:[0-9]+}")
	api.HandleFunc("/logs", searchClass).Methods("GET").Queries("classid", "{classid:[0-9]+}")
	api.HandleFunc("/logs", searchStudentEmail).Methods("GET").Queries("studentEmail", "{studentEmail}")
	api.HandleFunc("/logs", searchLesson).Methods("GET").Queries("classid", "{classid:[0-9]+}")
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":8080", nil))
	// TODO Log requests
	// TODO regex for email
	// TODO unix timestamp to datetime
	// TODO ToString from MongoDB
	// TODO Set sort and limit
}
