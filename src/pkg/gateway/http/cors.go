package http

import (
	"net/http"
	"os"
	"strings"
)

// applyAPICORS sets Access-Control-Allow-* for API routes (P4-1).
// OPENOCTA_CORS_ORIGINS: comma-separated allowlist; empty = * (dev default).
func applyAPICORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	allowlist := strings.TrimSpace(os.Getenv("OPENOCTA_CORS_ORIGINS"))

	if allowlist == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else if origin != "" && corsOriginAllowed(allowlist, origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
}

func corsOriginAllowed(allowlist, origin string) bool {
	for _, part := range strings.Split(allowlist, ",") {
		if strings.TrimSpace(part) == origin {
			return true
		}
	}
	return false
}
