package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mager/keiko/database"

	"go.uber.org/zap"
)

type GetHomeResp struct {
	RandomNFT RandomNFT `json:"randomNFT"`
	Stats     Stats     `json:"stats"`
}

type RandomNFT struct {
	CollectionName string    `json:"collectionName"`
	CollectionSlug string    `json:"collectionSlug"`
	Expires        time.Time `json:"expires"`
	ImageURL       string    `json:"imageUrl"`
	Name           string    `json:"name"`
	Owner          string    `json:"owner"`
	OwnerName      string    `json:"ownerName"`
	Updated        time.Time `json:"updated"`
}

type Stats struct {
	TotalCollections int       `json:"totalCollections"`
	TotalUsers       int       `json:"totalUsers"`
	Updated          time.Time `json:"updated"`
}

// getStats is the route handler for the GET /home endpoint
func (h *Handler) getHome(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = context.TODO()
		resp = GetHomeResp{}
	)

	// Get stats
	resp.Stats = getStats(ctx, h.logger, h.dbClient)

	// Get Random NFT
	resp.RandomNFT = getRandomNFT(ctx, h.logger, h.dbClient)

	json.NewEncoder(w).Encode(resp)
}

func getStats(ctx context.Context, logger *zap.SugaredLogger, db *database.DatabaseClient) Stats {
	data, err := db.Client.Collection("features").Doc("stats").Get(ctx)
	if err != nil {
		logger.Errorf("Error fetching stats: %v", err)
	}

	var stats = Stats{}
	err = data.DataTo(&stats)
	if err != nil {
		logger.Errorf("Error fetching stats: %v", err)
	}

	return stats
}

func getRandomNFT(ctx context.Context, logger *zap.SugaredLogger, db *database.DatabaseClient) RandomNFT {
	nft, err := db.Client.Collection("features").Doc("nftoftheday").Get(ctx)
	if err != nil {
		logger.Errorf("Error fetching nftoftheday: %v", err)
	}

	var n RandomNFT
	err = nft.DataTo(&n)
	if err != nil {
		logger.Errorf("Error fetching nftoftheday: %v", err)
	}

	return n
}
