package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type UpdateUserResp struct {
	Success bool `json:"success"`
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	// Get address from path params
	var (
		vars    = mux.Vars(r)
		address = vars["address"]
		resp    UpdateUserResp
	)

	// Update user
	resp.Success = h.sweeper.UpdateUser(address)

	json.NewEncoder(w).Encode(resp)
}
