package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mager/keiko/constants"
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
	Name        string              `json:"name"`
	Slug        string              `json:"slug"`
	Floor       float64             `json:"floor"`
	Updated     time.Time           `json:"updated"`
	Thumb       string              `json:"thumb"`
	Stats       []Stat              `json:"stats"`
	IsFollowing bool                `json:"isFollowing"`
	Collection  database.Collection `json:"collection"`
	Contract    database.Contract   `json:"contract"`
}

// getCollection is the route handler for the GET /collection/{slug} endpoint
func (h *Handler) getCollection(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		resp        = GetCollectionResp{}
		collections = h.dbClient.Client.Collection("collections")
		users       = h.dbClient.Client.Collection("users")
		address     = r.Header.Get("X-Address")
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
	resp.Floor = d["floor"].(float64)
	resp.Updated = d["updated"].(time.Time)

	thumb, ok := d["thumb"].(string)
	if ok {
		resp.Thumb = thumb
	}

	// Check if the user is following the collection
	if address != "" && address != constants.DefaultAddress {
		var db database.User
		docsnap, err := users.Doc(address).Get(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := docsnap.DataTo(&db); err != nil {
			h.logger.Error(err.Error())
		} else {
			if utils.Contains(db.Collections, slug) {
				resp.IsFollowing = true
			}
		}
	}

	json.NewEncoder(w).Encode(resp)
}
