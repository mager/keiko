package handler

import (
	"encoding/json"
	"net/http"
)

type GetCollectionsResp struct {
	Trending GetTrendingResp `json:"trending"`
}

// getCollections is the route handler for the GET /collections endpoint
func (h *Handler) getCollections(w http.ResponseWriter, r *http.Request) {
	var resp GetCollectionsResp

	// Get trending collections
	resp.Trending = h.GetTrending()

	json.NewEncoder(w).Encode(resp)
}
