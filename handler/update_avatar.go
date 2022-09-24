package handler

import (
	"encoding/json"
	"net/http"
)

type UpdateAvatarResp struct {
	Success bool `json:"success"`
}

func (h *Handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	// Get address from path params
	var (
		// vars = mux.Vars(r)
		// address = vars["address"]
		resp UpdateAvatarResp
	)

	json.NewEncoder(w).Encode(resp)
}
