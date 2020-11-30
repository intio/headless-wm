package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type APIServer struct {
	server *http.Server
	wm     *WM
}

func jsonResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	log.Printf("%d %s", status, r.URL.Path)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	e := json.NewEncoder(w)
	e.Encode(data)
}

func NewAPIServer(wm *WM, listenAddr string) (as *APIServer) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:           listenAddr,
		Handler:        mux,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, r, 200, map[string]interface{}{
			"message": "hello",
		})
	})
	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, r, 200,
			map[string]interface{}{
				"clients": as.wm.clients,
			},
		)
	})
	mux.Handle("/", http.NotFoundHandler())
	as = &APIServer{
		server: server,
		wm:     wm,
	}
	return as
}

func (as *APIServer) Start() {
	log.Printf("Listening on http://%s", as.server.Addr)
	log.Fatal(as.server.ListenAndServe())
}
