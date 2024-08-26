package middlewares

import (
	"encoding/json"
	"log"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			return
		}
	}
}

func HttpError(w http.ResponseWriter, message string, status int, err error) {
	log.Printf("HTTP %d - %s: %v", status, message, err)
	http.Error(w, message, status)
}
