package main

import (
	"errors"
	"log"
	"os"

	"github.com/BurntSushi/xgb/xproto"
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
	var wm = NewWM()
	err := wm.Init()
	if err != nil {
		panic(err)
	}
	defer wm.Deinit()

	for i := 1; i < 8; i++ {
		wm.AddWorkspace(&Workspace{Layout: &ColumnLayout{}})
	}

	for {
		err := wm.handleEvent()
		switch err {
		case errorQuit:
			os.Exit(0)
		case nil:
		default:
			log.Print(err)
		}
		assert(wm.painter.DrawText(
			xproto.Drawable(wm.xroot.Root),
			0, 12,
			wm.xroot.WhitePixel,
			"look mama, I'm drawing some random text on the root window",
		))
	}
}
