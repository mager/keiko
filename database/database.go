package database

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type DatabaseClient struct {
	Client *firestore.Client
	Apps   map[string]Application
}

// ProvideDB provides a firestore client
func ProvideDB() *DatabaseClient {
	projectID := "floor-report-327113"

	client, err := firestore.NewClient(context.TODO(), projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	apps := GetApplicationMap(client)

	return &DatabaseClient{
		Client: client,
		Apps:   apps,
	}
}

var Options = ProvideDB

type Application struct {
	Name   string `firestore:"name" json:"name"`
	APIKey string `firestore:"apiKey" json:"apiKey"`
}

func GetApplicationMap(db *firestore.Client) map[string]Application {
	// Fetch apps from database
	ctx := context.Background()
	iter := db.Collection("applications").Documents(ctx)
	apps := make(map[string]Application)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		app := Application{}
		doc.DataTo(&app)
		apps[doc.Ref.ID] = app
	}

	return apps
}
