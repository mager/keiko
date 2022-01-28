package main

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"github.com/mager/go-opensea/opensea"
	bq "github.com/mager/keiko/bigquery"
	cs "github.com/mager/keiko/coinstats"
	"github.com/mager/keiko/config"
	"github.com/mager/keiko/database"
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
			bq.Options,
			config.Options,
			cs.Options,
			database.Options,
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
	bq *bigquery.Client,
	cfg config.Config,
	cs cs.CoinstatsClient,
	etherscanClient *ethscan.EtherscanClient,
	database *firestore.Client,
	infuraClient *infura.InfuraClient,
	logger *zap.SugaredLogger,
	openSeaClient *opensea.OpenSeaClient,
	router *mux.Router,
	sweeper sweeper.SweeperClient,
) {
	// TODO: Remove global context
	var ctx = context.Background()

	// Route handler
	handler.New(
		ctx,
		logger,
		router,
		openSeaClient,
		bq,
		cs,
		database,
		infuraClient,
		etherscanClient,
		sweeper,
	)
}
