package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/mager/keiko/database"
	"google.golang.org/api/iterator"
)

type GetFollowingResp struct {
	Collections []database.CollectionV2 `json:"collections"`
	Addresses   []string                `json:"addresses"`
}

func (h *Handler) getFollowing(w http.ResponseWriter, r *http.Request) {
	var (
		ctx         = context.TODO()
		resp        = GetFollowingResp{}
		users       = h.database.Collection("users")
		collections = h.database.Collection("collections")
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

	// Fetch the list of collections that the user follows
	q := collections.Where("slug", "in", db.Collections).OrderBy("floor", firestore.Desc)
	iter := q.Documents(ctx)
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

		var db database.CollectionV2
		if err := doc.DataTo(&db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Collections = append(resp.Collections, db)
	}

	json.NewEncoder(w).Encode(resp)
}
