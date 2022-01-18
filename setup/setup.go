package setup

import (
	"context"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"microservice-jitsi-log-view/utils"
)

const (
	DEFAULT_TIMEZONE = "America/Sao_Paulo"
)

var (
	// URI for MongoDB connection.
	mongodb_uri string

	// Database to use for storing logs.
	database string

	// Collection to use for storing logs.
	collection string

	// Port to listen.
	port string

	// Timezone as *time.Location
	timezone *time.Location
)

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

	mongodb_uri = utils.Getenv("URI_MONGODB", "mongodb://localhost:27017")
	database = utils.Getenv("DATABASE", "jitsilog")
	collection = utils.Getenv("COLLECTION", "logs")
	TZ := utils.Getenv("TIMEZONE", DEFAULT_TIMEZONE)

	tz, err := time.LoadLocation(TZ)
	if err != nil {
		log.WithField("timezone", TZ).Warnf(
			"could not parse timezone; falling back to %s", DEFAULT_TIMEZONE)
		timezone, _ = time.LoadLocation(DEFAULT_TIMEZONE)
	} else {
		timezone = tz
	}

	port = os.Getenv("PORT")
	if !strings.HasPrefix(port, ":") {
		port = ":8080"
		log.Info("Port variable is missing or in wrong format (missing a colon ( : ) at start. It should be like ':8080'), using default: :8080")
	}

	log.WithFields(log.Fields{
		"URI":        mongodb_uri,
		"Database":   database,
		"Collection": collection}).Info("Database Connection Info")

	log.Info("Listening at ", port)
	log.Info("Using ", timezone.String(), " as timezone")
	log.Info("CORS Enabled")
}

func GetMongoDBUri() string {
	return mongodb_uri
}

func GetDatabase() string {
	return database
}

func GetCollection() string {
	return collection
}

func GetPort() string {
	return port
}

func GetTimezone() *time.Location {
	return timezone
}

// GetMongoClient creates and returns a MongoDB client.
func GetMongoClient() *mongo.Client {
	context, fn := context.WithTimeout(context.Background(), 10*time.Second)
	fn()

	client, err := mongo.Connect(context, options.Client().ApplyURI(GetMongoDBUri()))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err}).Fatal("Failed to create the Mongo client!")
	}
	return client
}
