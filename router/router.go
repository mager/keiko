package router

import (
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/bigquery"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	bq "github.com/mager/keiko/bigquery"
	"github.com/mager/keiko/database"
	"github.com/mager/keiko/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProvideRouter provides a gorilla mux router
func ProvideRouter(
	lc fx.Lifecycle,
	logger *zap.SugaredLogger,
	dbClient *database.DatabaseClient,
	bqClient *bigquery.Client,
) *mux.Router {
	var router = mux.NewRouter()

	router.Use(
		jsonMiddleware,
		verifySignatureMiddleware,
	)

	return router
}

func authMiddleware(dbClient *database.DatabaseClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Make sure they are sending an API key
			apiKey := r.Header.Get("X-API-KEY")
			if apiKey == "" {
				http.Error(w, "Missing API key", http.StatusUnauthorized)
				return
			}

			// Make sure the API key is in the apps map
			for _, app := range dbClient.Apps {
				if app.APIKey == apiKey {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Invalid API key", http.StatusUnauthorized)
		})
	}
}

func apiLoggingMiddleware(dbClient *database.DatabaseClient, bqClient *bigquery.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Make sure they are sending an API key
			apiKey := r.Header.Get("X-API-KEY")

			// Log the request in BigQuery
			bq.RecordAPICall(bqClient, apiKey, r.URL.Path)

			next.ServeHTTP(w, r)
		})
	}
}

// jsonMiddleware makes sure that every response is JSON
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func verifySignatureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			sig              = r.Header.Get("X-Signature")
			address          = r.Header.Get("X-Address")
			msg              = r.Header.Get("X-Message")
			currentRoute     = mux.CurrentRoute(r).GetName()
			restrictedRoutes = []string{"followCollection", "unfollowCollection", "updateUser"}
		)

		if utils.Contains(restrictedRoutes, currentRoute) {
			if sig == "" {
				http.Error(w, "Missing X-Signature header", http.StatusBadRequest)
				return
			}

			if address == "" {
				http.Error(w, "Missing X-Address header", http.StatusBadRequest)
				return
			}

			if msg == "" {
				http.Error(w, "Missing X-Message header", http.StatusBadRequest)
				return
			}

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

var Options = ProvideRouter
