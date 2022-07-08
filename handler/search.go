package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/api/iterator"
)

type SearchReq struct {
	Query string `json:"query"`
}

type SearchResp struct {
	Collections []SearchCollection `json:"collections"`
}

type SearchCollection struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// search is the route handler for the POST /search endpoint
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		err         error
		req         SearchReq
		resp        SearchResp
		collections = h.dbClient.Client.Collection("collections")
	)

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	queryLower := strings.ToLower(req.Query)
	iter := collections.
		Where("slug", ">=", queryLower).
		Where("slug", "<=", queryLower+"\uf8ff").
		Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var collection SearchCollection
		if err := doc.DataTo(&collection); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		collection.Slug = doc.Ref.ID

		resp.Collections = append(resp.Collections, collection)
	}

	json.NewEncoder(w).Encode(resp)
}
