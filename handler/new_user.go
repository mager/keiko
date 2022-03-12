package handler

import (
	"encoding/json"
	"net/http"
)

type NewUserReq struct {
	ENSName string `json:"ensName"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Photo   string `json:"photo"`
	Twitter string `json:"twitter"`
	OpenSea string `json:"openSea"`
	IsWhale bool   `json:"isWhale"`
}

type NewUserResp struct {
	Created bool `json:"created"`
}

func (h *Handler) newUser(w http.ResponseWriter, r *http.Request) {
	var (
		req     NewUserReq
		users   = h.dbClient.Client.Collection("users")
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
		d := map[string]interface{}{
			"collections": []string{},
		}
		if req.ENSName != "" {
			d["ensName"] = req.ENSName
		}
		if req.Slug != "" {
			d["slug"] = req.Slug
		}
		if req.Name != "" {
			d["name"] = req.Name
		}
		if req.Photo != "" {
			d["photo"] = req.Photo
		}
		if req.Twitter != "" {
			d["twitter"] = req.Twitter
		}
		if req.OpenSea != "" {
			d["openSea"] = req.OpenSea
		}
		if req.IsWhale {
			d["isWhale"] = true
		}

		_, err := users.Doc(address).Create(h.ctx, d)
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
