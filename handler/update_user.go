package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"github.com/mager/sweeper/database"
)

// UpdateUserReq is a request to /user/{address}
type UpdateUserReq struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type UpdateUserResp struct {
	Success bool   `json:"success"`
	Field   string `json:"field"`
	Value   string `json:"value"`
}

func (h *Handler) upsertUser(address, field, value string) bool {
	// Fetch user from Firestore
	docsnap, err := h.dbClient.Client.Collection("users").Doc(address).Get(h.ctx)
	if err != nil {
		h.logger.Error(err)
		return false
	}

	h.logger.Infow("upsertUser", "address", address, "field", field, "value", value)
	// Update user
	docsnap.Ref.Update(h.ctx, []firestore.Update{{Path: field, Value: value}})

	return true
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	var (
		ctx          = context.TODO()
		err          error
		req          = UpdateUserReq{}
		resp         = UpdateUserResp{}
		users        = h.dbClient.Client.Collection("users")
		address      = r.Header.Get("X-Address")
		addressParam = mux.Vars(r)["address"]
		db           database.User
	)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if address != addressParam {
		http.Error(w, "address mismatch", http.StatusBadRequest)
		return
	}

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

	success := h.upsertUser(address, req.Field, req.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success = success
	resp.Field = req.Field
	resp.Value = req.Value

	json.NewEncoder(w).Encode(resp)
}
