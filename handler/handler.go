package handler

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/coinstats"
	"github.com/mager/keiko/etherscan"
	"github.com/mager/keiko/infura"
	"github.com/mager/keiko/opensea"
	"go.uber.org/zap"
)

// Handler struct for HTTP requests
type Handler struct {
	ctx             context.Context
	logger          *zap.SugaredLogger
	router          *mux.Router
	os              opensea.OpenSeaClient
	bq              *bigquery.Client
	cs              coinstats.CoinstatsClient
	database        *firestore.Client
	infuraClient    *infura.InfuraClient
	etherscanClient *etherscan.EtherscanClient
}

// New creates a Handler struct
func New(
	ctx context.Context,
	logger *zap.SugaredLogger,
	router *mux.Router,
	os opensea.OpenSeaClient,
	bq *bigquery.Client,
	cs coinstats.CoinstatsClient,
	database *firestore.Client,
	infuraClient *infura.InfuraClient,
	etherscanClient *etherscan.EtherscanClient,
) *Handler {
	h := Handler{ctx, logger, router, os, bq, cs, database, infuraClient, etherscanClient}
	h.registerRoutes()

	return &h
}

// RegisterRoutes registers all the routes for the route handler
func (h *Handler) registerRoutes() {
	// Address
	h.router.HandleFunc("/address/{address}", h.getInfoV3).Methods("GET")

	// Stats
	h.router.HandleFunc("/stats", h.getStats).Methods("GET")
	h.router.HandleFunc("/trending", h.getTrending).Methods("GET")

	// Collections
	h.router.HandleFunc("/collection/{slug}", h.getCollection).Methods("GET")

	// Testing
	h.router.HandleFunc("/collection/{slug}/tokens", h.getCollectionTokens).Methods("GET")
}
