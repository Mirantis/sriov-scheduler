package extender

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

func MakeServer(ext *Extender, addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/filter", ext)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return srv
}

func (ext *Extender) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("error dumping request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println(string(body))

	if r.Method == "POST" {
		var args ExtenderArgs
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&args); err != nil {
			if err != io.EOF {
				log.Printf("error unmarshalling body: %v\n", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if result, err := ext.FilterArgs(&args); err != nil {
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
			w.Write(body)
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
