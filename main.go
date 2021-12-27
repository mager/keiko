package main

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	bq "github.com/mager/sweeper/bigquery"
	cs "github.com/mager/sweeper/coinstats"
	"github.com/mager/sweeper/config"
	"github.com/mager/sweeper/database"
	ethscan "github.com/mager/sweeper/etherscan"
	"github.com/mager/sweeper/handler"
	"github.com/mager/sweeper/infura"
	"github.com/mager/sweeper/logger"
	"github.com/mager/sweeper/opensea"
	"github.com/mager/sweeper/router"
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
			opensea.Options,
			router.Options,
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
	openSeaClient opensea.OpenSeaClient,
	router *mux.Router,
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
	)
}
