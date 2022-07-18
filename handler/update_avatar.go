package handler

import (
	"encoding/json"
	"net/http"
)

type UpdateUserMetadataResp struct {
	Success bool `json:"success"`
}

func (h *Handler) updateUserMetadata(w http.ResponseWriter, r *http.Request) {
	// Get address from path params
	var (
		// vars = mux.Vars(r)
		// address = vars["address"]
		resp UpdateUserResp
	)

	json.NewEncoder(w).Encode(resp)
}
