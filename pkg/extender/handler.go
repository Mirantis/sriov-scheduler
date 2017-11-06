package extender

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

func MakeServer(ext *Extender, addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/filter", MakeHandler(ext.FilterArgs))
	mux.HandleFunc("/prioritize", MakeHandler(ext.Prioritize))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return srv
}

func MakeHandler(f func(*ExtenderArgs) (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var args ExtenderArgs
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&args); err != nil {
				if err != io.EOF {
					log.Printf("error unmarshalling body: %v\n", err)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if result, err := f(&args); err != nil {
				log.Printf("error running filter: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				body, err := json.Marshal(result)
				if err != nil {
					log.Printf("error marshalling result: %v\n", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(200)
				if _, err := w.Write(body); err != nil {
					log.Printf("error writing response body: %v", err)
				}
			}
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	}
}
