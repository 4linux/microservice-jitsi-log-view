package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Timezone as *time.Location
	TIMEZONE *time.Location
)

const (
	DEFAULT_TIMEZONE = "America/Sao_Paulo"
	CHAN_BUFFER_SIZE = 100
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
	timestamp time.Time
	Action    string `json:"action"`
}

func cabecalhoCSV() (c []string) {
	c = append(c, "sala")
	c = append(c, "curso")
	c = append(c, "turma")
	c = append(c, "aluno")
	c = append(c, "jid")
	c = append(c, "email")
	c = append(c, "timestamp")
	c = append(c, "action")
	return
}

func (jl *Jitsilog) registroCSV() (r []string) {
	r = append(r, jl.Sala)
	r = append(r, jl.Curso)
	r = append(r, jl.Turma)
	r = append(r, jl.Aluno)
	r = append(r, jl.Jid)
	r = append(r, jl.Email)
	r = append(r, jl.Timestamp)
	r = append(r, jl.Action)
	return
}

func iterLogs(logs []*Jitsilog) <-chan *Jitsilog {
	ch := make(chan *Jitsilog, CHAN_BUFFER_SIZE)

	go func() {
		for _, log := range logs {
			ch <- log
		}
		close(ch)
	}()

	return ch
}

func filterByAction(logs <-chan *Jitsilog, action string) <-chan *Jitsilog {
	ch := make(chan *Jitsilog, CHAN_BUFFER_SIZE)

	go func() {
		for log := range logs {
			if log.Action == action {
				ch <- log
			}
		}
		close(ch)
	}()

	return ch
}

type logsByEmail struct {
	email string
	logs  []*Jitsilog
}

func groupbyEmail(logs <-chan *Jitsilog) <-chan logsByEmail {
	ch := make(chan logsByEmail, CHAN_BUFFER_SIZE)

	go func() {
		entries := make(map[string][]*Jitsilog)
		for log := range logs {
			entries[log.Email] = append(entries[log.Email], log)
		}

		for email, logs := range entries {
			ch <- logsByEmail{email, logs}
		}
		close(ch)
	}()

	return ch
}

func findClosestTimeTo(dur time.Duration, logs <-chan *Jitsilog) *Jitsilog {
	// helper abs function because Go doesn't fucking provide a
	// decent abs() for integers
	abs := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	var closest *Jitsilog
	offset := 24 * time.Hour

	for log := range logs {
		h, m, s := log.timestamp.Clock()
		logDur, _ := time.ParseDuration(fmt.Sprintf("%dh%dm%ds", h, m, s))
		tmpOffset := logDur - dur
		if abs(tmpOffset) < abs(offset) {
			offset = tmpOffset
			closest = log
		}
	}

	return closest
}

// Setup of logs and database related configs.
func init() {
	log.Debug("microservice-jitsi-log-view init")
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	URI_MONGODB = getenv("URI_MONGODB", "mongodb://localhost:27017")
	DATABASE = getenv("DATABASE", "jitsilog")
	COLLECTION = getenv("COLLECTION", "logs")
	TZ := getenv("TIMEZONE", DEFAULT_TIMEZONE)
	tz, err := time.LoadLocation(TZ)
	if err != nil {
		log.WithField("timezone", TZ).Warnf(
			"could not parse timezone; falling back to %s", DEFAULT_TIMEZONE)
		TIMEZONE, _ = time.LoadLocation(DEFAULT_TIMEZONE)
	} else {
		TIMEZONE = tz
	}

	if port := os.Getenv("PORT"); strings.HasPrefix(port, ":") {
		PORT = port
	} else {
		PORT = ":8080"
		log.Info("Port variable is missing or in wrong format (missing a colon ( : ) at start. It should be like ':8080'), using default: :8080")
	}

	log.WithFields(log.Fields{
		"URI":        URI_MONGODB,
		"Database":   DATABASE,
		"Collection": COLLECTION}).Info("Database Connection Info")

	log.Info("Listening at ", PORT)
	log.Info("Using ", TIMEZONE.String(), " as timezone")
	log.Info("CORS Enabled")
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

// convertToInt tries to convert all strings passed to int64,
// returning a slice of int64 on success. If any conversion fails,
// the error is returned alongside the index of the non-integer string.
func convertToInt(values []string) ([]int64, int, error) {
	ints := make([]int64, len(values))

	for idx, v := range values {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, idx, err
		}
		ints[idx] = i
	}

	return ints, -1, nil
}

// getenv gets the value associated with the environtment variable
// passed in as `key` and returns it, if found. Otherwise, it returns
// `defaultValue`.
func getenv(key, defaultValue string) string {
	if value, found := os.LookupEnv(key); found {
		return value
	}

	return defaultValue
}

// Find logs with filter and ordered by decrescent timestamp, can limit & skip items in dataset.
func findLogsFilter(size string, filter bson.D, skip string) ([]*Jitsilog, error) {
	client := getClient()
	optFind := options.Find()
	var jitsilogs []*Jitsilog

	sizeAndSkip, failIdx, err := convertToInt([]string{size, skip})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info(fmt.Sprintf(
			"Failed to convert argument at index `%d` to int", failIdx))
		return nil, err
	}

	log.Debug("Dataset row limit ", sizeAndSkip[0])
	log.Debug("Dataset row skip ", sizeAndSkip[1])
	collection := client.Database(DATABASE).Collection(COLLECTION)
	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Error on count of the documents")
		return nil, err
	}
	if sizeAndSkip[1] > count {
		sizeAndSkip[1] = count
	} else if sizeAndSkip[1] < 0 {
		sizeAndSkip[1] = 0
	}
	if sizeAndSkip[0] < 0 {
		sizeAndSkip[0] = 20
	}
	log.Debug("Dataset row max: ", count)
	optFind.SetLimit(sizeAndSkip[0])
	optFind.SetSkip(sizeAndSkip[1])
	optFind.SetSort(bson.D{{"timestamp", -1}})
	cursor, err := collection.Find(context.TODO(), filter, optFind)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Error on finding the documents")
		return nil, err
	}
	log.Debug("Connection to MongoDB opened.")
	for cursor.Next(context.TODO()) {
		var jitsilog Jitsilog
		err = cursor.Decode(&jitsilog)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Error on decoding the document")
			return nil, err
		}
		t, err := time.ParseInLocation(time.RFC3339, jitsilog.Timestamp, TIMEZONE)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Failed to parse ISO8601")
			jitsilog.Timestamp = "Falha no parser"
		} else {
			jitsilog.timestamp = t.In(TIMEZONE)
			jitsilog.Timestamp = jitsilog.timestamp.String()
		}
		jitsilogs = append(jitsilogs, &jitsilog)
	}
	log.Debug("Data retrived")
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to disconnect from database!")
	}
	log.Debug("Connection to MongoDB closed.")
	return jitsilogs, nil
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
	jitsilogs, err := findLogsFilter(queryParams["size"][0], bson.D{}, queryParams["skip"][0])
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
	jitsilogs, err := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
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
	jitsilogs, err := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
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
	jitsilogs, err := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
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
	jitsilogs, err := findLogsFilter(queryParams["size"][0], filter, queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

// Query all logs earlier than a timestamp and export them as a CSV file
func searchAndExportAsCSV(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	timestamp := queryParams.Get("ts")
	curso := queryParams.Get("curso")
	turma := queryParams.Get("turma")
	email := queryParams.Get("email")
	sala := queryParams.Get("sala")
	t0s := queryParams.Get("t0")
	t1s := queryParams.Get("t1")
	now := time.Now()

	// preparing the response to output a csv file
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition",
		"attachment; filename=jitsi-presence-logger."+now.Format(time.RFC3339)+".csv")
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = ';'

	// querying database
	filter := bson.D{{"timestamp", bson.D{{"$gte", timestamp}}}}

	if curso != "" {
		filter = append(filter, bson.E{Key: "curso", Value: curso})
	}

	if turma != "" {
		filter = append(filter,
			bson.E{Key: "turma", Value: bson.D{{"$regex", primitive.Regex{Pattern: turma, Options: "gi"}}}})
	}

	if email != "" {
		filter = append(filter, bson.E{
			Key: "email", Value: bson.D{{"$regex", primitive.Regex{Pattern: email, Options: "gi"}}}},
		)
	}

	if sala != "" {
		filter = append(filter, bson.E{
			Key: "sala", Value: bson.D{{"$regex", primitive.Regex{Pattern: sala, Options: "gi"}}}})
	}
	fmt.Printf("%+v\n", filter)

	jitsilogs, err := findLogsFilter("0", filter, "0")
	var logsToWrite []*Jitsilog

	// writing response
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
		csvWriter.Write([]string{
			"Ocorreu um erro ao realizar a requisição", err.Error(),
		})
	} else {
		if t0s == "" && t1s == "" {
			logsToWrite = jitsilogs
		} else {
			if t0s != "" {
				t0, err := time.ParseDuration(t0s + "ms")
				if err == nil {
					loginLogs := filterByAction(iterLogs(jitsilogs), "login")
					loginLogsByEmail := groupbyEmail(loginLogs)

					for userLog := range loginLogsByEmail {
						logsToWrite = append(
							logsToWrite, findClosestTimeTo(t0, iterLogs(userLog.logs)))
					}
				}
			}

			if t1s != "" {
				t1, err := time.ParseDuration(t1s + "ms")
				if err == nil {
					logoutLogs := filterByAction(iterLogs(jitsilogs), "logout")
					logoutLogsByEmail := groupbyEmail(logoutLogs)

					for userLog := range logoutLogsByEmail {
						logsToWrite = append(
							logsToWrite, findClosestTimeTo(t1, iterLogs(userLog.logs)))
					}
				}
			}
		}

		csvWriter.Write(cabecalhoCSV())
		for _, log := range logsToWrite {
			csvWriter.Write(log.registroCSV())
		}
	}

	csvWriter.Flush()
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck", checkHealth).Methods(http.MethodGet)
	version := router.PathPrefix("/v1").Subrouter()
	version.HandleFunc("/csv", searchAndExportAsCSV).Methods(http.MethodGet).Queries(
		"ts", "{ts}", "curso", "{curso:(?:\\d+)?}", "turma", "{turma:(?:\\d+)?}",
		"email", "{email}", "sala", "{sala}",
		"t0", "{t0:(?:\\d+)?}", "t1", "{t1:(?:\\d+)?}")
	api := version.PathPrefix("/logs").Subrouter()
	api.HandleFunc("/last", latestLogsHandler).Methods("GET")
	api.HandleFunc("/course", searchCourseHandler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/class", searchClassHandler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/student", searchStudentHandler).Methods("GET").Queries("email", "{email}")
	api.HandleFunc("/room", searchRoomHandler).Methods("GET").Queries("id", "{id}")
	http.Handle("/", router)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(PORT, handlers.CORS()(loggedRouter)))
}
