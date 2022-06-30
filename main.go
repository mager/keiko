package main

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mager/go-opensea/opensea"
	cs "github.com/mager/keiko/coinstats"
	"github.com/mager/keiko/config"
	db "github.com/mager/keiko/database"
	ethscan "github.com/mager/keiko/etherscan"
	"github.com/mager/keiko/handler"
	"github.com/mager/keiko/infura"
	"github.com/mager/keiko/logger"
	os "github.com/mager/keiko/opensea"
	"github.com/mager/keiko/router"
	"github.com/mager/keiko/sweeper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			config.Options,
			cs.Options,
			db.Options,
			ethscan.Options,
			infura.Options,
			logger.Options,
			os.Options,
			router.Options,
			sweeper.Options,
		),
		fx.Invoke(Register),
	).Run()
}

func Register(
	lc fx.Lifecycle,
	cfg config.Config,
	cs cs.CoinstatsClient,
	etherscanClient *ethscan.EtherscanClient,
	dbClient *db.DatabaseClient,
	infuraClient *infura.InfuraClient,
	logger *zap.SugaredLogger,
	openSeaClient *opensea.OpenSeaClient,
	router *mux.Router,
	sweeper sweeper.SweeperClient,
) {
	// TODO: Remove global context
	ctx := context.Background()

	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				addr := ":8081"
				logger.Info("Listening on ", addr)

				go http.ListenAndServe(addr, router)

				return nil
			},
		},
	)

	// Route handler
	handler.New(
		ctx,
		logger,
		router,
		openSeaClient,
		cs,
		dbClient,
		infuraClient,
		etherscanClient,
		sweeper,
	)
}
