package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/mager/sweeper/database"
	"google.golang.org/api/iterator"
)

type GetFrensResp struct {
	Users []database.User `json:"users"`
}

func (h *Handler) getFrens(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = context.TODO()
		resp  = GetFrensResp{}
		users = h.database.Collection("users")
	)

	// Fetch the list of collections that the user follows
	q := users.Where("isWhale", "==", true).OrderBy("name", firestore.Asc)
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

		var db database.User
		if err := doc.DataTo(&db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Users = append(resp.Users, db)
	}

	json.NewEncoder(w).Encode(resp)
}
