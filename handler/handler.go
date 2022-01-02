package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/coinstats"
	"github.com/mager/keiko/etherscan"
	"github.com/mager/keiko/infura"
	"github.com/mager/keiko/opensea"
	"github.com/mager/keiko/sweeper"
	"github.com/mager/keiko/utils"
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
	sweeper         sweeper.SweeperClient
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
	sweeper sweeper.SweeperClient,
) *Handler {
	h := Handler{
		ctx,
		logger,
		router,
		os,
		bq,
		cs,
		database,
		infuraClient,
		etherscanClient,
		sweeper,
	}
	h.registerRoutes()
	h.router.Use(verifySignatureMiddleware)
	return &h
}

// RegisterRoutes registers all the routes for the route handler
func (h *Handler) registerRoutes() {
	// Address
	h.router.HandleFunc("/address/{address}", h.getInfoV3).
		Methods("GET")

	// Stats
	h.router.HandleFunc("/stats", h.getStats).
		Methods("GET")
	h.router.HandleFunc("/trending", h.getTrending).
		Methods("GET")

	// Users
	h.router.HandleFunc("/users", h.newUser).
		Methods("POST")
	h.router.HandleFunc("/following", h.getFollowing).
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

func verifySignatureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Signature")
		address := r.Header.Get("X-Address")
		msg := r.Header.Get("X-Message")
		currentRoute := mux.CurrentRoute(r).GetName()

		signedRoutes := []string{"followCollection", "unfollowCollection"}
		if utils.Contains(signedRoutes, currentRoute) {
			if !verifySig(address, sig, []byte(msg)) {
				log.Println("Signature verification failed")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func verifySig(from, sigHex string, msg []byte) bool {
	fromAddr := common.HexToAddress(from)

	sig := hexutil.MustDecode(sigHex)
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	if sig[64] != 27 && sig[64] != 28 {
		return false
	}
	sig[64] -= 27

	pubKey, err := crypto.SigToPub(signHash(msg), sig)
	if err != nil {
		return false
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	return fromAddr == recoveredAddr
}

// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L404
// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
