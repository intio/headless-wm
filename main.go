package main

import (
	"errors"
	"log"
	"os"
)

var version string

var (
	errorQuit      = errors.New("Quit")
	errorAnotherWM = errors.New("Another WM already running")
)

func (wm *WM) closeClientGracefully() error {
	if wm.activeClient == nil {
		log.Println("Tried to close client, but no active client")
		return nil
	}
	return wm.activeClient.CloseGracefully()
}

func (wm *WM) closeClientForcefully() error {
	if wm.activeClient == nil {
		log.Println("Tried to close client, but no active client")
		return nil
	}
	return wm.activeClient.CloseForcefully()
}

func main() {
	if version != "" {
		log.Printf("version: %s", version)
	}
	var wm = NewWM()
	err := wm.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer wm.Deinit()
	var api = NewAPIServer(wm, "127.0.0.1:8080")
	go api.Start()

	for {
		err := wm.handleEvent()
		switch err {
		case errorQuit:
			os.Exit(0)
		case nil:
		default:
			log.Print(err)
		}
	}
}
