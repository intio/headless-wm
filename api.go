package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/gorilla/mux"
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
	router := mux.NewRouter()
	server := &http.Server{
		Addr:           listenAddr,
		Handler:        router,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	router.HandleFunc("/screens/", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, r, 200,
			map[string]interface{}{
				"items": as.wm.attachedScreens,
			},
		)
	}).Methods("GET")

	router.HandleFunc("/clients/", func(w http.ResponseWriter, r *http.Request) {
		for _, client := range as.wm.clients {
			name, _ := client.GetName()
			client.Name = name
		}
		jsonResponse(w, r, 200,
			map[string]interface{}{
				"items": as.wm.clients,
			},
		)
	}).Methods("GET")

	getIdUint := func(r *http.Request) *uint64 {
		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			return nil
		}
		return &id
	}
	getInt := func(key string, data map[string]interface{}) *uint32 {
		if value, ok := data[key]; ok {
			if f, ok := value.(float64); ok {
				u := uint32(f)
				return &u
			}
		}
		return nil
	}
	getBool := func(key string, data map[string]interface{}) *bool {
		if value, ok := data[key]; ok {
			if b, ok := value.(bool); ok {
				return &b
			}
		}
		return nil
	}

	router.HandleFunc("/clients/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		id := getIdUint(r)
		if id == nil {
			jsonResponse(w, r, http.StatusNotFound, nil)
			return
		}
		client, ok := as.wm.clients[xproto.Window(*id)]
		if !ok {
			jsonResponse(w, r, http.StatusNotFound, nil)
			return
		}
		switch r.Method {
		case "GET":
			break
		case "POST":
			d := json.NewDecoder(r.Body)
			var data map[string]interface{}
			err := d.Decode(&data)
			if err != nil {
				jsonResponse(w, r, http.StatusUnprocessableEntity, nil)
				return
			}
			log.Print("update client ", id, " with ", data)
			if X := getInt("X", data); X != nil {
				client.X = *X
			}
			if Y := getInt("Y", data); Y != nil {
				client.Y = *Y
			}
			if W := getInt("W", data); W != nil {
				client.W = *W
			}
			if H := getInt("H", data); H != nil {
				client.H = *H
			}
			if Fullscreen := getBool("Fullscreen", data); Fullscreen != nil && *Fullscreen {
				client.X = 0
				client.Y = 0
				client.W = /* uint16-> */ uint32(as.wm.xroot.WidthInPixels)
				client.H = /* uint16-> */ uint32(as.wm.xroot.HeightInPixels)
			}
			client.Configure()
		case "DELETE":
			if err := client.CloseGracefully(); err != nil {
				log.Print(err)
			}
			jsonResponse(w, r, 200, nil)
			return
		default:
			panic("unreachable")
		}
		name, _ := client.GetName()
		client.Name = name
		jsonResponse(w, r, 200,
			map[string]interface{}{
				"item": client,
			},
		)
	}).Methods("GET", "POST", "DELETE")

	router.PathPrefix("/").Handler(http.NotFoundHandler())
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
