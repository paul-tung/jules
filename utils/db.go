package utils

import (
	"auth-service/config"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client
var cfg *config.Config

// InitDB initializes the database connection.
func InitDB() {
	cfg = config.LoadConfig()
	var err error
	client, err = mongo.NewClient(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Error creating MongoDB client: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal("Error connecting to MongoDB: ", err)
	}

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("Error pinging MongoDB: ", err)
	}
	log.Println("Connected to MongoDB!")

	// Create indexes
	err = CreateIndexes(client)
	if err != nil {
		log.Fatal("Error creating indexes: ", err)
	}
	log.Println("Indexes created successfully!")
}

var dbName = "authdb" // Default database name

// SetDBName allows changing the database name, primarily for testing.
func SetDBName(name string) {
	dbName = name
}

// GetDB returns the MongoDB database instance.
func GetDB() *mongo.Database {
	if client == nil {
		log.Fatal("MongoDB client not initialized. Call InitDB first.")
	}
	return client.Database(dbName)
}

// CreateIndexes creates the necessary indexes for the users collection.
// It now uses the configured dbName.
func CreateIndexes(client *mongo.Client) error {
	collection := client.Database(dbName).Collection("users")

	// Email index
	emailIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true),
	}
	_, err := collection.Indexes().CreateOne(context.Background(), emailIndex)
	if err != nil {
		return err
	}

	// Username index
	usernameIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"username": 1},
		Options: options.Index().SetUnique(true).SetSparse(true), // Sparse for optional unique username
	}
	_, err = collection.Indexes().CreateOne(context.Background(), usernameIndex)
	if err != nil {
		return err
	}

	return nil
}

// DisconnectDB disconnects the MongoDB client.
func DisconnectDB() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal("Error disconnecting from MongoDB: ", err)
		}
		log.Println("Disconnected from MongoDB.")
	}
}
