package main

import (
	"errors"
	"log"
	"os"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

var xc *xgb.Conn
var errorQuit error = errors.New("Quit")

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

func (wm *WM) initScreens() {
	setup := xproto.Setup(xc)
	if setup == nil || len(setup.Roots) < 1 {
		log.Fatal("Could not parse SetupInfo.")
	}
	if err := xinerama.Init(xc); err != nil {
		log.Fatal(err)
	}
	if r, err := xinerama.QueryScreens(xc).Reply(); err != nil {
		log.Fatal(err)
	} else {
		if len(r.ScreenInfo) == 0 {
			// If Xinerama does not return useful information, we can
			// still query the root window, and create a fake
			// ScreenInfo structure.
			wm.attachedScreens = []xinerama.ScreenInfo{
				xinerama.ScreenInfo{
					Width:  setup.Roots[0].WidthInPixels,
					Height: setup.Roots[0].HeightInPixels,
				},
			}
		} else {
			wm.attachedScreens = r.ScreenInfo
		}
	}

	coninfo := xproto.Setup(xc)
	if coninfo == nil {
		log.Fatal("Could not parse X connection info")
	}
	if len(coninfo.Roots) != 1 {
		log.Fatal("Bad number of roots. Did Xinerama initialize correctly?")
	}
	wm.xroot = coninfo.Roots[0]
}

func (wm *WM) initWM() {
	err := xproto.ChangeWindowAttributesChecked(
		xc,
		wm.xroot.Root,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease |
				xproto.EventMaskButtonPress |
				xproto.EventMaskButtonRelease |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskSubstructureRedirect,
		},
	).Check()
	if err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			log.Fatal("Could not become the WM. Is another WM already running?")
		}
		log.Fatal(err)
	}
}

func (wm *WM) initWorkspaces() {
	tree, err := xproto.QueryTree(xc, wm.xroot.Root).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if tree != nil {
		w := wm.GetActiveWorkspace()
		for _, win := range tree.Children {
			if c, err := NewClient(win); err != nil {
				log.Println(err)
			} else {
				w.AddClient(c)
			}
		}
	}
	if len(wm.attachedScreens) == 0 {
		panic("no attached screens!?")
	}
	for _, workspace := range wm.workspaces {
		workspace.Screen = &wm.attachedScreens[0]
		if err := workspace.Arrange(); err != nil {
			log.Println(err)
		}
	}
}

func main() {
	var wm = &WM{
		workspaces: []*Workspace{
			&Workspace{
				Layout: &ColumnLayout{},
			},
		},
	}

	var err error
	xc, err = xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	defer xc.Close()

	wm.initScreens()
	initAtoms()
	wm.initWM()
	wm.initKeys()
	wm.initWorkspaces()

	for {
		err = wm.handleEvent()
		switch err {
		case errorQuit:
			os.Exit(0)
		case nil:
		default:
			log.Print(err)
		}
	}
}
