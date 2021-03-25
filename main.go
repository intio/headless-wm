package main

import (
	"errors"
	"git.sr.ht/~sircmpwn/getopt"
	"log"
	"os"
)

var (
	version    string
	listenAddr string = "127.0.0.1:8080"
)

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
	opts, _, err := getopt.Getopts(os.Args, "l:")
	if err != nil {
		log.Fatal(err)
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'l':
			listenAddr = opt.Value
		}
	}
	if version != "" {
		log.Printf("version: %s", version)
	}
	var wm = NewWM()
	err = wm.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer wm.Deinit()
	var api = NewAPIServer(wm, listenAddr)
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
