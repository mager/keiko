package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/constants"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
	"google.golang.org/api/iterator"
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
		contracts   = h.dbClient.Client.Collection("contracts")
		address     = r.Header.Get("X-Address")
		slug        = mux.Vars(r)["slug"]
		stats       = make([]BQStat, 0)
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

	// Fetch time-series data from BigQuery
	q := h.bqClient.Query(fmt.Sprintf(`
		SELECT Floor, RequestTime, SevenDayVolume
		FROM `+"`floor-report-327113.collections.update`"+`
		WHERE slug = "%s"
		ORDER BY RequestTime
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
		stats = append(stats, BQStat{
			Floor:     values[0].(float64),
			Timestamp: values[1].(time.Time),
		})
	}

	// resp.Stats = h.adaptStats(stats)
	resp.Stats = h.adaptStats(stats)

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

	// BETA: Show contract for Proof Collective
	if slug == "proof-collective" || slug == "eightbitme" {
		c := contracts.Doc(slug)
		docsnap, err := c.Get(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var contract database.Contract
		if err := docsnap.DataTo(&contract); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Contract = contract
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) adaptStats(stats []BQStat) []Stat {
	var statsByDate = h.reduceStats(stats)

	var result = make([]Stat, 0)

	for _, stat := range statsByDate {
		result = append(result, Stat{
			Date:  h.formatStatDate(stat),
			Floor: stat.Floor,
		})
	}

	return result
}

// Only return one stat per day, filter out 0 values
func (h *Handler) reduceStats(stats []BQStat) []BQStat {
	var reducedByDate = []BQStat{}
	var reducedRemoveZeros = []BQStat{}

	// Return one stat per day
	for _, stat := range stats {
		if len(reducedByDate) == 0 {
			reducedByDate = append(reducedByDate, stat)
		} else {
			if stat.Timestamp.Day() != reducedByDate[len(reducedByDate)-1].Timestamp.Day() {
				reducedByDate = append(reducedByDate, stat)
			}
		}
	}

	// Remove 0 values
	for _, stat := range reducedByDate {
		if stat.Floor != 0 {
			reducedRemoveZeros = append(reducedRemoveZeros, stat)
		}
	}

	return reducedRemoveZeros
}

func (h *Handler) formatStatDate(stat BQStat) string {
	return stat.Timestamp.Format("01-02")
}
