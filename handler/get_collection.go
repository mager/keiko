package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/database"
	"github.com/mager/keiko/opensea"
	"github.com/mager/keiko/utils"
	"google.golang.org/api/iterator"
)

type Stat struct {
	Floor     float64   `json:"floor"`
	Timestamp time.Time `json:"timestamp"`
}

type GetCollectionResp struct {
	Name              string                    `json:"name"`
	Slug              string                    `json:"slug"`
	Floor             float64                   `json:"floor"`
	WeeklyVolumeETH   float64                   `json:"weeklyVolumeETH"`
	Updated           time.Time                 `json:"updated"`
	Thumb             string                    `json:"thumb"`
	OpenSeaCollection opensea.OpenSeaCollection `json:"opensea_collection"`
	Stats             []Stat                    `json:"stats"`
	IsFollowing       bool                      `json:"isFollowing"`
}

// getCollection is the route handler for the GET /collection/{slug} endpoint
func (h *Handler) getCollection(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		resp        = GetCollectionResp{}
		collections = h.database.Collection("collections")
		users       = h.database.Collection("users")
		address     = r.Header.Get("X-Address")
		slug        = mux.Vars(r)["slug"]
		stats       = make([]Stat, 0)
	)

	// Fetch collection from database
	docsnap, err := collections.Doc(slug).Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d := docsnap.Data()

	// Set slug
	resp.Name = d["name"].(string)
	resp.Slug = slug
	resp.Floor = d["floor"].(float64)
	resp.WeeklyVolumeETH = d["7d"].(float64)
	resp.Updated = d["updated"].(time.Time)

	// Fetch collection from OpenSea
	openSeaCollection, err := h.os.GetCollection(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumb, ok := d["thumb"].(string)
	if ok {
		resp.Thumb = thumb
	}

	// Hydrate response with OpenSea content
	resp.OpenSeaCollection = openSeaCollection.Collection

	// Fetch time-series data from BigQuery
	q := h.bq.Query(fmt.Sprintf(`
		SELECT Floor, RequestTime, SevenDayVolume
		FROM `+"`floor-report-327113.collections.update`"+`
		WHERE slug = "%s"
		ORDER BY RequestTime DESC
	`, slug))
	it, _ := q.Read(ctx)
	for {
		var values []bigquery.Value
		err := it.Next(&values)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		stats = append(stats, Stat{
			Floor:     values[0].(float64),
			Timestamp: values[1].(time.Time),
		})
	}

	resp.Stats = stats

	// Check if the user is following the collection
	if address != "" {
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
