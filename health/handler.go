package health

import (
	"encoding/json"
	"net/http"
	"proxy/config"
	"time"
)

var healthResponse map[string]any

func init() {
	healthResponse = map[string]any{
		"status":    "UP",
		"startTime": time.Now().Format(time.RFC3339),
		"go-proxy": map[string]any{
			"commit":    Commit,
			"buildTime": BuildTime,
			"source":    Source,
		},
	}
}

func SetConfigVersion(cfg *config.Config) {
	healthResponse["config"] = map[string]any{
		"version": cfg.Version,
		"routes":  len(cfg.Routes),
	}
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}
