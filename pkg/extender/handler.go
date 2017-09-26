package extender

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
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
	var args ExtenderArgs
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	if err := json.Unmarshal(body, &args); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	if result, err := ext.FilterArgs(&args); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
	} else {
		body, err := json.Marshal(result)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		w.Write(body)
	}
}
