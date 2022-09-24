package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mager/sweeper/database"
)

type UpdateSettingsReq struct {
	HideZeroETHCollections bool `json:"hide0ETHCollections"`
}

type UpdateSettingsResp struct {
	Success bool `json:"success"`
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	// Get address from path params
	var (
		vars    = mux.Vars(r)
		address = vars["address"]
		req     UpdateSettingsReq
		resp    UpdateSettingsResp
	)

	// Decode request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error(err)
	}

	// Adapt request to database.UserSettings
	var settings database.UserSettings
	settings.HideZeroETHCollections = req.HideZeroETHCollections

	resp.Success = h.sweeper.UpdateUserSettings(address, settings)

	json.NewEncoder(w).Encode(resp)
}
