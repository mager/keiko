package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
)

type FollowCollectionResp struct {
	Success bool `json:"success"`
}

func (h *Handler) followCollection(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = context.TODO()
		err     error
		resp    = FollowCollectionResp{}
		users   = h.dbClient.Client.Collection("users")
		address = r.Header.Get("X-Address")
		slug    = mux.Vars(r)["slug"]
		db      database.User
	)

	if address == "" {
		http.Error(w, "X-Address is required", http.StatusBadRequest)
		return
	}

	docsnap, err := users.Doc(address).Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := docsnap.DataTo(&db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if utils.Contains(db.Collections, slug) {
		http.Error(w, "Collection already followed", http.StatusBadRequest)
		return
	}

	db.Collections = append(db.Collections, slug)

	_, err = users.Doc(address).Set(ctx, db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success = true

	json.NewEncoder(w).Encode(resp)
}
