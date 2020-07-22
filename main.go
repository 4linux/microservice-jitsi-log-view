package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// URI for MongoDB connection.
	URI_MONGODB string

	// Database to use for storing logs.
	DATABASE string

	// Collection to use for storing logs.
	COLLECTION string

	// Port to listen.
	PORT string

	// Timezone to display datetime.
	TIMEZONE string
)

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
	log.Debug("microservice-jitsi-log-view init")
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	if len(os.Getenv("DEBUG")) > 0 && os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
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
	if len(os.Getenv("TIMEZONE")) > 0 {
		TIMEZONE = os.Getenv("TIMEZONE")
	} else {
		TIMEZONE = "America/Sao_Paulo"
	}
	if len(os.Getenv("PORT")) > 0 && strings.HasPrefix(PORT, ":") == true {
		PORT = os.Getenv("PORT")
	} else {
		PORT = ":8080"
		log.Info("Port variable is missing or in wrong format (missing a colon ( : ) at start. It should be like ':8080'), using default one")
	}
	log.WithFields(log.Fields{
		"URI":        URI_MONGODB,
		"Database":   DATABASE,
		"Collection": COLLECTION}).Info("Database Connection Info")
	log.Info("Listening at ", PORT)
	log.Info("Using ", TIMEZONE, " as timezone")
}

// Creates and return a MongoDB client.
func getClient() *mongo.Client {
	context, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(context, options.Client().ApplyURI(URI_MONGODB))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to create the Mongo client!")
	}
	return client
}

// Find logs with filter and ordered by decrescent timestamp, can limit & skip items in dataset.
func findLogsFilter(size string, filter bson.D, skip string) (error, []*Jitsilog) {
	tz, err := time.LoadLocation(TIMEZONE)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to load TZ info")
	}
	client := getClient()
	optFind := options.Find()
	var jitsilogs []*Jitsilog
	sizeInt, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to convert size argument to int")
		return err, nil
	}
	skipInt, err := strconv.ParseInt(skip, 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to convert skip argument to int")
		return err, nil
	}
	log.Debug("Dataset row limit ", sizeInt)
	log.Debug("Dataset row skip ", skipInt)
	collection := client.Database(DATABASE).Collection(COLLECTION)
	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Error on count of the documents")
		return err, nil
	}
	if skipInt > count {
		skipInt = count
	} else if skipInt < 0 {
		skipInt = 0
	}
	if sizeInt < 0 {
		sizeInt = 20
	}
	log.Debug("Dataset row max: ", count)
	optFind.SetSkip(skipInt)
	optFind.SetLimit(sizeInt)
	optFind.SetSort(bson.D{{"timestamp", -1}})
	cursor, err := collection.Find(context.TODO(), filter, optFind)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Error on finding the documents")
		return err, nil
	}
	log.Debug("Connection to MongoDB opened.")
	for cursor.Next(context.TODO()) {
		var jitsilog Jitsilog
		err = cursor.Decode(&jitsilog)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Error on decoding the document")
			return err, nil
		}
		t, err := time.ParseInLocation(time.RFC3339, jitsilog.Timestamp, tz)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Failed to parse ISO8601")
		}
		jitsilog.Timestamp = t.In(tz).String()
		jitsilogs = append(jitsilogs, &jitsilog)
	}
	log.Debug("Data retrived")
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to disconnect from database!")
	}
	log.Debug("Connection to MongoDB closed.")
	return nil, jitsilogs
}

// Default handler, return the name of this service.
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "microservice-jitsi-log-view")
}

// Check health of the microservice. Returns the hostname of the machine or container running on.
func checkHealth(w http.ResponseWriter, r *http.Request) {
	name, err := os.Hostname()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to get hostname!")
	}
	fmt.Fprintf(w, "Awake and alive from %s", name)
}

// Query the latest logs with a variable dataset size based on the URL.
func latestLogsHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	err, jitsilogs := findLogsFilter(queryParams["size"][0], bson.D{}, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired courseid.
func searchCourseHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{}}
	filter = append(filter, bson.E{Key: "curso", Value: bson.D{{"$regex", primitive.Regex{Pattern: queryParams["id"][0], Options: "gi"}}}})
	err, jitsilogs := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired classid
func searchClassHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{}}
	filter = append(filter, bson.E{Key: "turma", Value: bson.D{{"$regex", primitive.Regex{Pattern: queryParams["id"][0], Options: "gi"}}}})
	err, jitsilogs := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired roomid
func searchRoomHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{}}
	filter = append(filter, bson.E{Key: "sala", Value: bson.D{{"$regex", primitive.Regex{Pattern: queryParams["id"][0], Options: "gi"}}}})
	err, jitsilogs := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs that correspond with desired student email
func searchStudentHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{{}}
	filter = append(filter, bson.E{Key: "email", Value: bson.D{{"$regex", primitive.Regex{Pattern: queryParams["email"][0], Options: "gi"}}}})
	err, jitsilogs := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck", checkHealth).Methods(http.MethodGet)
	version := router.PathPrefix("/v1").Subrouter()
	api := version.PathPrefix("/logs").Subrouter()
	api.HandleFunc("/last", latestLogsHandler).Methods("GET")
	api.HandleFunc("/course", searchCourseHandler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/class", searchClassHandler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/student", searchStudentHandler).Methods("GET").Queries("email", "{email}")
	api.HandleFunc("/room", searchRoomHandler).Methods("GET").Queries("id", "{id}")
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(PORT, nil))
}
