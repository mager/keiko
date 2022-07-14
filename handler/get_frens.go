package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/firestore"
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
}

func (h *Handler) getFrens(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = context.TODO()
		resp  = GetFrensResp{}
		users = h.dbClient.Client.Collection("users")
	)

	// Fetch the list of collections that the user follows
	q := users.Where("isFren", "==", true).Where("photo", "==", true)
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

		var user User
		if err := doc.DataTo(&user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var fren Fren

		fren.Address = doc.Ref.ID
		fren.Name = getName(user)
		fren.Photo = user.Photo
		fren.Slug = getSlug(user, doc)

		resp.Users = append(resp.Users, fren)
	}

	json.NewEncoder(w).Encode(resp)
}

func getName(user User) string {
	if user.Name != "" {
		return user.Name
	}
	if user.ENSName != "" {
		return user.ENSName
	}

	return ""
}

func getSlug(user User, doc *firestore.DocumentSnapshot) string {
	if user.ENSName != "" {
		return user.ENSName
	}

	return doc.Ref.ID
}
