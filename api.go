package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/gorilla/mux"
	"nhooyr.io/websocket"
)

type WSClient struct {
	ch chan interface{}
}

type APIServer struct {
	server  *http.Server
	wm      *WM
	clients map[*WSClient]interface{}
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
			if fullscreenOn := getInt("FullscreenOn", data); fullscreenOn != nil {
				if int(*fullscreenOn) < len(as.wm.attachedScreens) {
					screen := &as.wm.attachedScreens[int(*fullscreenOn)]
					client.MakeFullscreen(screen)
				}
			}
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
		jsonResponse(w, r, 200,
			map[string]interface{}{
				"item": client,
			},
		)
	}).Methods("GET", "POST", "DELETE")

	router.HandleFunc(
		"/events/",
		makeWSHandler(func(ctx context.Context, c *websocket.Conn) {
			client := &WSClient{ch: make(chan interface{}, 10)}
			as.clients[client] = nil
			defer func() {
				// Normally only the writer should ever close the
				// channel.  However we never terminate the WS
				// connection on our own so this seems fair.
				close(client.ch)
				delete(as.clients, client)
			}()
			for v := range client.ch {
				data, err := json.Marshal(v)
				if err != nil {
					log.Print(err)
					continue
				}
				c.Write(ctx, websocket.MessageText, data)
			}
			c.Close(websocket.StatusNormalClosure, "")
		}),
	)

	router.PathPrefix("/").Handler(http.NotFoundHandler())
	as = &APIServer{
		server:  server,
		wm:      wm,
		clients: make(map[*WSClient]interface{}),
	}
	wm.api = as
	return as
}

func (as *APIServer) Start() {
	log.Printf("Listening on http://%s", as.server.Addr)
	log.Fatal(as.server.ListenAndServe())
}
