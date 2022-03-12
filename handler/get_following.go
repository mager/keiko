package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/mager/sweeper/database"
)

type GetFollowingResp struct {
	Collections []database.Collection `json:"collections"`
	Addresses   []string              `json:"addresses"`
}

func (h *Handler) getFollowing(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		resp        = GetFollowingResp{}
		users       = h.dbClient.Client.Collection("users")
		collections = h.dbClient.Client.Collection("collections")
		address     = r.Header.Get("X-Address")
	)

	if address == "" {
		http.Error(w, "X-Address is required", http.StatusBadRequest)
		return
	}

	// Fetch user from database
	docsnap, err := users.Doc(address).Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the list of collections that the user follows
	var db database.User
	if err := docsnap.DataTo(&db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make a slice of document references
	var docRefs []*firestore.DocumentRef
	for _, collection := range db.Collections {
		docRefs = append(docRefs, collections.Doc(collection))
	}

	// Fetch the list of collections that the user follows
	docsnaps, err := h.dbClient.Client.GetAll(ctx, docRefs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make a slice of collections
	for _, docsnap := range docsnaps {
		var collection database.Collection
		if err := docsnap.DataTo(&collection); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Collections = append(resp.Collections, collection)
	}

	json.NewEncoder(w).Encode(resp)
}
