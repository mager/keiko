package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"google.golang.org/api/iterator"
)

type GetFrensResp struct {
	Users []Fren `json:"users"`
}

type Fren struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Photo   bool   `json:"photo"`
	Slug    string `json:"slug"`
	ENSName string `json:"ensName"`
}

func (h *Handler) getFrens(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = context.TODO()
		resp  = GetFrensResp{}
		users = h.dbClient.Client.Collection("users")
	)

	// Fetch the list of collections that the user follows
	q := users.Where("IsFren", "==", true)
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

		var fren Fren
		if err := doc.DataTo(&fren); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fren.Address = doc.Ref.ID

		resp.Users = append(resp.Users, fren)
	}

	json.NewEncoder(w).Encode(resp)
}
