package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mager/sweeper/database"
)

type User struct {
	Name        string   `json:"name"`
	Bio         string   `json:"bio"`
	Photo       bool     `json:"photo"`
	ENSName     string   `json:"ensName"`
	Collections []string `json:"collections"`
	Slug        string   `json:"slug"`
	Twitter     string   `json:"twitter"`
	OpenSea     string   `json:"openSea"`
	IsWhale     bool     `json:"isWhale"`
	DiscordID   string   `json:"discordID"`
}

// UserReq is a request to /user/{address}
type UserReq struct {
	Address string `json:"address"`
}

type UserResp struct {
	User
}

func (h *Handler) fetchUser(address string) (database.User, error) {
	// Fetch user from Firestore
	var user database.User

	docsnap, err := h.dbClient.Client.Collection("users").Doc(address).Get(h.ctx)
	if err != nil {
		h.logger.Error(err)
		return user, err
	}

	if docsnap.Exists() {
		err = docsnap.DataTo(&user)
		if err != nil {
			h.logger.Error(err)
		}
	} else {
		h.logger.Info("User not found in Firestore")
		return user, err
	}

	return user, nil
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		address = strings.ToLower(mux.Vars(r)["address"])
	)

	user, err := h.fetchUser(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := UserResp{
		User: User{
			Name:        user.Name,
			Bio:         user.Bio,
			Photo:       user.Photo,
			ENSName:     user.ENSName,
			Collections: user.Collections,
			Slug:        user.Slug,
			Twitter:     user.Twitter,
			IsWhale:     user.IsWhale,
			DiscordID:   user.DiscordID,
		},
	}

	json.NewEncoder(w).Encode(resp)
}
