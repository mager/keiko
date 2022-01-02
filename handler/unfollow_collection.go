package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
)

type UnfollowCollectionResp struct {
	Success bool `json:"success"`
}

func (h *Handler) unfollowCollection(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = context.TODO()
		err     error
		resp    = UnfollowCollectionResp{}
		users   = h.database.Collection("users")
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

	if !utils.Contains(db.Collections, slug) {
		http.Error(w, "Collection not followed", http.StatusBadRequest)
		return
	}

	// Remove the slug from the list
	db.Collections = utils.Remove(db.Collections, slug)

	_, err = users.Doc(address).Set(ctx, db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success = true

	json.NewEncoder(w).Encode(resp)
}
