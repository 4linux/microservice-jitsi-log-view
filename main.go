package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"microservice-jitsi-log-view/iterators"
	"microservice-jitsi-log-view/setup"
	"microservice-jitsi-log-view/types"
	"microservice-jitsi-log-view/utils"
)

// Find logs with filter and ordered by decrescent timestamp, can limit & skip items in dataset.
func findLogsFilter(filter bson.D, size, skip string) (types.JitsilogSlice, error) {
	client := setup.GetMongoClient()
	optFind := options.Find()
	var jitsilogs types.JitsilogSlice

	sizeAndSkip, failIdx, err := utils.ConvertToInt([]string{size, skip})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info(fmt.Sprintf(
			"Failed to convert argument at index `%d` to int", failIdx))
		return nil, err
	}

	log.Debug("Dataset row limit ", sizeAndSkip[0])
	log.Debug("Dataset row skip ", sizeAndSkip[1])
	collection := client.Database(setup.GetDatabase()).Collection(setup.GetCollection())
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
		var jitsilog types.Jitsilog
		err = cursor.Decode(&jitsilog)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Error on decoding the document")
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, jitsilog.Timestamp)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err}).Info("Failed to parse RFC3339 on " + jitsilog.Timestamp)
		} else {
			jitsilog.SetTime(t.In(setup.GetTimezone()))
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

type MongoQueryHandler struct {
	Key           string
	QueryParamKey string
}

// Query logs according to given keys. See `this` commit for more info.
func (hnd MongoQueryHandler) Handler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	filter := bson.D{}

	if hnd.Key != "" && hnd.QueryParamKey != "" {
		filter = append(
			filter,
			bson.E{Key: hnd.Key, Value: bson.D{{
				"$regex",
				primitive.Regex{Pattern: queryParams[hnd.QueryParamKey][0], Options: "gi"},
			}}})
	}
	jitsilogs, err := findLogsFilter(filter, queryParams["size"][0], queryParams["skip"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Info("Failed to get logs!")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jitsilogs)
}

func selectLogsByTimeAndAction(timeMs, action string, logsToRead, logsToWrite types.JitsilogSlice) types.JitsilogSlice {
	byEmail := func(jl *types.Jitsilog) string { return jl.Email }
	byDate := func(jl *types.Jitsilog) string { return jl.GetTime().Format("2006-01-02") }

	if timeMs != "" {
		t, err := time.ParseDuration(timeMs + "ms")
		if err == nil {
			actionLogs := iterators.FilterByAction(action, iterators.IterLogs(logsToRead))
			logsByEmail := iterators.GroupByField(byEmail, actionLogs)

			for userLog := range logsByEmail {
				for dailyUserLog := range iterators.GroupByField(byDate, iterators.IterLogs(userLog.Logs)) {
					logsToWrite = append(
						logsToWrite, utils.FindClosestTimeTo(t, iterators.IterLogs(dailyUserLog.Logs)))
				}
			}
		}
	} else {
		logsToWrite = iterators.IteratorToSlice(
			logsToWrite, iterators.FilterByAction(action, iterators.IterLogs(logsToRead)))
	}

	return logsToWrite
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

	jitsilogs, err := findLogsFilter(filter, "0", "0")
	var logsToWrite types.JitsilogSlice

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
			logsToWrite = selectLogsByTimeAndAction(t0s, "login", jitsilogs, logsToWrite)
			logsToWrite = selectLogsByTimeAndAction(t1s, "logout", jitsilogs, logsToWrite)

			sort.SliceStable(
				logsToWrite, func(earlier, later int) bool {
					return logsToWrite[later].GetTime().Before(logsToWrite[earlier].GetTime())
				})
		}

		csvWriter.Write(types.CabecalhoCSV())
		for _, log := range logsToWrite {
			csvWriter.Write(log.RegistroCSV())
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
	api.HandleFunc("/last", MongoQueryHandler{}.Handler).Methods("GET")
	api.HandleFunc("/course", MongoQueryHandler{"curso", "id"}.Handler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/class", MongoQueryHandler{"turma", "id"}.Handler).Methods("GET").Queries("id", "{id}")
	api.HandleFunc("/student", MongoQueryHandler{"email", "email"}.Handler).Methods("GET").Queries("email", "{email}")
	api.HandleFunc("/room", MongoQueryHandler{"sala", "id"}.Handler).Methods("GET").Queries("id", "{id}")
	http.Handle("/", router)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(setup.GetPort(), handlers.CORS()(loggedRouter)))
}
