package handler

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/mager/keiko/database"
	sweeper "github.com/mager/sweeper/database"

	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type GetHomeResp struct {
	Total     int       `json:"total"`
	Updated   time.Time `json:"updated"`
	RandomNFT RandomNFT `json:"randomNFT"`
}

type RandomNFT struct {
	Collection string `json:"collection"`
	ImageURL   string `json:"imageUrl"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
}

// getStats is the route handler for the GET /home endpoint
func (h *Handler) getHome(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = context.TODO()
		resp = GetHomeResp{}
	)

	// Fetch stats
	total, updated := getStats(ctx, h.logger, h.dbClient)

	// Set total
	resp.Total = total

	// Set last updated
	resp.Updated = updated

	// Get Random NFT
	resp.RandomNFT = getRandomNFT(ctx, h.logger, h.dbClient)

	json.NewEncoder(w).Encode(resp)
}

func getStats(ctx context.Context, logger *zap.SugaredLogger, db *database.DatabaseClient) (int, time.Time) {
	var (
		docs        = make([]*firestore.DocumentRef, 0)
		collections = db.Client.Collection("collections")
		updated     = time.Time{}
	)

	// Fetch all collections
	iter := collections.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Errorf("Error fetching collections: %v", err)
			break
		}
		docs = append(docs, doc.Ref)
		if doc.Data()["updated"].(time.Time).After(updated) {
			updated = doc.Data()["updated"].(time.Time)
		}
	}

	return len(docs), updated
}

func getRandomNFT(ctx context.Context, logger *zap.SugaredLogger, db *database.DatabaseClient) RandomNFT {
	var (
		docs        = make([]*firestore.DocumentRef, 0)
		collections = db.Client.Collection("users")
	)

	// Initialize local pseudorandom generator
	rand.Seed(time.Now().Unix())

	// Fetch a random user
	iter := collections.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Errorf("Error fetching collections: %v", err)
			break
		}
		docs = append(docs, doc.Ref)
	}

	// Get random user
	user := docs[rand.Intn(len(docs))]
	u, err := user.Get(ctx)
	if err != nil {
		logger.Errorf("Error fetching user: %v", err)
	}

	// Get random NFT
	var userData sweeper.User
	err = u.DataTo(&userData)
	if err != nil {
		logger.Errorf("Error fetching user: %v", err)
	}
	collection := userData.Wallet.Collections[rand.Intn(len(userData.Wallet.Collections))]
	nft := collection.NFTs[rand.Intn(len(collection.NFTs))]
	var resp = RandomNFT{
		Collection: collection.Name,
		ImageURL:   nft.ImageURL,
		Name:       nft.Name,
		Owner:      getOwner(u, userData),
	}

	return resp
}

func getOwner(u *firestore.DocumentSnapshot, user sweeper.User) string {
	if user.ENSName != "" {
		return user.ENSName
	}
	return u.Ref.ID
}
