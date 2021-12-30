package handler

import (
	"encoding/json"
	"net/http"
)

type NewUserReq struct {
	ENSName string `json:"ensName"`
}

type NewUserResp struct {
	Created bool `json:"created"`
}

func (h *Handler) newUser(w http.ResponseWriter, r *http.Request) {
	var (
		// ctx  = context.TODO()
		req     NewUserReq
		users   = h.database.Collection("users")
		resp    = NewUserResp{}
		address = r.Header.Get("X-Address")
	)

	// Process the request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the user already exists
	docsnap, _ := users.Doc(address).Get(h.ctx)

	// If the user already exists, return success
	if !docsnap.Exists() {
		// Create the user
		_, err := users.Doc(address).Create(h.ctx, map[string]interface{}{
			"collections": []string{},
			"ensName":     req.ENSName,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Created = true
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(resp)
}
