package handler

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/mager/go-opensea/opensea"
	"github.com/mager/keiko/coinstats"
	"github.com/mager/keiko/database"
	"github.com/mager/keiko/etherscan"
	"github.com/mager/keiko/infura"
	"github.com/mager/keiko/sweeper"
	"go.uber.org/zap"
)

// Handler struct for HTTP requests
type Handler struct {
	ctx             context.Context
	logger          *zap.SugaredLogger
	router          *mux.Router
	os              *opensea.OpenSeaClient
	cs              coinstats.CoinstatsClient
	dbClient        *database.DatabaseClient
	infuraClient    *infura.InfuraClient
	etherscanClient *etherscan.EtherscanClient
	sweeper         sweeper.SweeperClient
}

// New creates a Handler struct
func New(
	ctx context.Context,
	logger *zap.SugaredLogger,
	router *mux.Router,
	os *opensea.OpenSeaClient,
	cs coinstats.CoinstatsClient,
	dbClient *database.DatabaseClient,
	infuraClient *infura.InfuraClient,
	etherscanClient *etherscan.EtherscanClient,
	sweeper sweeper.SweeperClient,
) *Handler {
	h := Handler{
		ctx,
		logger,
		router,
		os,
		cs,
		dbClient,
		infuraClient,
		etherscanClient,
		sweeper,
	}
	h.registerRoutes()
	return &h
}

// RegisterRoutes registers all the routes for the route handler
func (h *Handler) registerRoutes() {
	// Address
	h.router.HandleFunc("/address/{address}", h.getAddress).
		Methods("GET")

	// Stats
	h.router.HandleFunc("/stats", h.getStats).
		Methods("GET")
	h.router.HandleFunc("/trending", h.getTrending).
		Methods("GET")

	// Users
	h.router.HandleFunc("/user/{address}", h.getUser).
		Methods("GET")
	h.router.HandleFunc("/user/{address}", h.updateUser).
		Methods("POST").
		Name("updateUser")
	h.router.HandleFunc("/users", h.newUser).
		Methods("POST")
	h.router.HandleFunc("/following", h.getFollowing).
		Methods("GET")

		// Frens
	h.router.HandleFunc("/frens", h.getFrens).
		Methods("GET")

	// Collections
	h.router.HandleFunc("/collection/{slug}", h.getCollection).
		Methods("GET")
	h.router.HandleFunc("/collection/{slug}/follow", h.followCollection).
		Methods("POST").
		Name("followCollection")
	h.router.HandleFunc("/collection/{slug}/unfollow", h.unfollowCollection).
		Methods("POST").
		Name("unfollowCollection")

	// Testing
	h.router.HandleFunc("/collection/{slug}/tokens", h.getCollectionTokens).
		Methods("GET")
}
