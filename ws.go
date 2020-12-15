package main

import (
	"context"
	"log"
	"net/http"

	"nhooyr.io/websocket"
)

func __wsHandlerEcho(ctx context.Context, c *websocket.Conn) {
	for {
		t, bs, err := c.Read(ctx)
		if err != nil {
			c.Close(websocket.StatusInternalError, "")
			return
		}
		log.Printf("recv: %#v", bs)
		c.Write(ctx, t, bs)
	}
	c.Close(websocket.StatusNormalClosure, "")
}

func makeWSHandler(
	handler func(context.Context, *websocket.Conn),
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Printf("connect error: %v", err)
			return
		}
		log.Printf("connect: %s %s %s", r.URL.Path, r.RemoteAddr, c.Subprotocol())
		defer log.Printf("disconnect: %s", r.RemoteAddr)
		defer c.Close(websocket.StatusInternalError, "")
		handler(r.Context(), c)
	}
}

func __wsExample() {
	addr := "127.0.0.1:8081"
	log.Printf("listen %s", addr)
	err := http.ListenAndServe(
		addr,
		http.HandlerFunc(makeWSHandler(__wsHandlerEcho)),
	)
	if err != nil {
		log.Fatal(err)
	}
}
