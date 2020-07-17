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
	"net/http"
	"os"
	"strconv"
	"time"
)

// URI for MongoDB connection.
var URI_MONGODB string

// Database to use for storing logs.
var DATABASE string

// Collection to user for storing logs.
var COLLECTION string

// Data structure as defined in https://github.com/bryanasdev000/microservice-jitsi-log .
type Jitsilog struct {
	Sala      string `json:"sala"`
	Curso     string `json:"curso"`
	Turma     string `json:"turma"`
	Aluno     string `json:"aluno"`
	Jid       string `json:"jid"`
	Email     string `json:"email"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
}

// Setup of logs and database related configs.
func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if len(os.Getenv("DEBUG")) > 0 {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
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
		"URI":        URI_MONGODB,
		"DATABASE":   DATABASE,
		"COLLECTION": COLLECTION}).Info("Database Connection Info")
}

// Creates and return a MongoDB client.
func getClient() *mongo.Client {
	context, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(context, options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		log.Fatal("Failed to create the client ", err)
	}
	return client
}

// Find logs with filter and ordered by decrescent timestamp, can limit & skip items in dataset.
func findLogsFilter(size string, filter bson.D, skip string) []*Jitsilog {
	client := getClient()
	optFind := options.Find()
	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		log.Fatal("Failed to convert size argument to int ", err)
	}
	skipInt, err := strconv.ParseInt(skip, 10, 64)
	if err != nil {
		log.Fatal("Failed to convert skip argument to int ", err)
	}
	optFind.SetLimit(sizeInt)
	log.Debug("Dataset row limit ", sizeInt)
	log.Debug("Dataset row skip ", skipInt)
	optFind.SetSkip(skipInt)
	optFind.SetLimit(sizeInt)
	optFind.SetSort(bson.D{{"timestamp", -1}})
	var jitsilogs []*Jitsilog
	collection := client.Database(DATABASE).Collection(COLLECTION)
	cursor, err := collection.Find(context.TODO(), filter, optFind)
	if err != nil {
		log.Fatal("Error on finding the documents ", err)
	}
	log.Debug("Connection to MongoDB opened.")
	for cursor.Next(context.TODO()) {
		var jitsilog Jitsilog
		err = cursor.Decode(&jitsilog)
		if err != nil {
			log.Fatal("Error on decoding the document ", err)
		}
		jitsilogs = append(jitsilogs, &jitsilog)
	}
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal("Failed to disconnect from database! ", err)
	}
	log.Debug("Connection to MongoDB closed.")
	return jitsilogs
}

// Default handler, return the name of this service.
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "microservice-jitsi-log-view")
}

// Check health of the microservice. Returns the hostname of the machine or container running on.
func checkHealth(w http.ResponseWriter, r *http.Request) {
	name, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}
	fmt.Fprintf(w, "Awake and alive from %s", name)
}

// Query the latest logs with a variable dataset size based on the URL.
func latestLogsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	var jitsilogs []*Jitsilog
	jitsilogs = findLogsFilter(queryParams["size"][0], bson.D{}, queryParams["skip"][0])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired courseid.
func searchCourseHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{"curso", queryParams["courseid"][0]}}
	var jitsilogs []*Jitsilog
	jitsilogs = findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired classid
func searchClassHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{"turma", queryParams["classid"][0]}}
	var jitsilogs []*Jitsilog
	jitsilogs = findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired roomid
func searchRoomHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{"sala", queryParams["roomid"][0]}}
	var jitsilogs []*Jitsilog
	jitsilogs = findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired student email
func searchStudentHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{"email", queryParams["studentEmail"][0]}}
	var jitsilogs []*Jitsilog
	jitsilogs = findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck", checkHealth).Methods(http.MethodGet)
	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/logs", latestLogsHandler).Methods("GET").Queries("last", "{last:[0-9]+}")
	api.HandleFunc("/logs", searchCourseHandler).Methods("GET").Queries("courseid", "{courseid}")
	api.HandleFunc("/logs", searchClassHandler).Methods("GET").Queries("classid", "{classid}")
	api.HandleFunc("/logs", searchStudentHandler).Methods("GET").Queries("studentEmail", "{studentEmail}")
	api.HandleFunc("/logs", searchRoomHandler).Methods("GET").Queries("roomid", "{roomid}")
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
