package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Protected handlers
func (a *API) handleUserGuilds(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*Claims)

	guilds, err := a.getDiscordGuilds(claims.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get guilds: %v", err), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(guilds)
}
