package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
)

type BQStat struct {
	Floor     float64   `json:"floor"`
	Timestamp time.Time `json:"timestamp"`
}
type Stat struct {
	Date  string  `json:"date"`
	Floor float64 `json:"floor"`
}

type GetCollectionResp struct {
	Name       string              `json:"name"`
	Slug       string              `json:"slug"`
	FloorETH   float64             `json:"floorETH"`
	FloorUSD   float64             `json:"floorUSD"`
	Updated    time.Time           `json:"updated"`
	Thumb      string              `json:"thumb"`
	Stats      []Stat              `json:"stats"`
	Collection database.Collection `json:"collection"`
}

// getCollection is the route handler for the GET /collection/{slug} endpoint
func (h *Handler) getCollection(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		resp        = GetCollectionResp{}
		collections = h.dbClient.Client.Collection("collections")
		slug        = mux.Vars(r)["slug"]
	)

	// Fetch collection from database
	docsnap, err := collections.Doc(slug).Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d := docsnap.Data()
	var c database.Collection
	if err := docsnap.DataTo(&c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Collection = c
	// Set slug
	resp.Name = d["name"].(string)
	resp.Slug = slug

	ethPriceUSD := h.cs.GetETHPrice()

	resp.FloorETH = d["floor"].(float64)
	resp.FloorUSD = utils.AdaptTotalUSD(resp.FloorETH, ethPriceUSD)

	resp.Updated = d["updated"].(time.Time)

	thumb, ok := d["thumb"].(string)
	if ok {
		resp.Thumb = thumb
	}

	json.NewEncoder(w).Encode(resp)
}
