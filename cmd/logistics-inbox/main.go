package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"

	"logistics-inbox/internal/logistics"
)

func main() {
	client, err := logistics.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	inbox := os.Getenv("TEAM_INBOX")
	if inbox == "" {
		log.Fatal("TEAM_INBOX is required")
	}
	router := logistics.NewRouter(client, inbox)
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var form logistics.ContactForm
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		clientIP, _, splitErr := net.SplitHostPort(r.RemoteAddr)
		if splitErr != nil {
			clientIP = r.RemoteAddr
		}
		result, err := router.Route(r.Context(), form, clientIP)
		if err != nil {
			var infraiErr *logistics.InfraiError
			if errors.As(err, &infraiErr) {
				http.Error(w, infraiErr.Message, infraiErr.StatusCode)
				return
			}
			http.Error(w, "delivery request could not be completed", http.StatusBadGateway)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message_id": result.MessageID})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
