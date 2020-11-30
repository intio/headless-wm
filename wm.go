package main

import (
	"errors"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

// WM holds the global window manager state.
type WM struct {
	xc *xgb.Conn

	xroot           xproto.ScreenInfo
	attachedScreens []xinerama.ScreenInfo

	clients      map[xproto.Window]*Client
	activeClient *Client

	activeWs int
}

// NewWM allocates internal WM data structures and creates a WM
// instance. No X11 calls are made until WM.Init() is called.
func NewWM() *WM {
	return &WM{
		clients: map[xproto.Window]*Client{},
	}
}

// Init opens the X11 connections, performs the necessary calls to set
// up the internal WM state, and to start managing windows. You should
// also call Deinit before you exit.
func (wm *WM) Init() error {
	var err error
	wm.xc, err = xgb.NewConn()
	if err != nil {
		return err
	}

	if err = wm.initScreens(); err != nil {
		return err
	}
	if err = wm.initAtoms(); err != nil {
		return err
	}
	if err = wm.initWM(); err != nil {
		return err
	}
	if err = wm.initClients(); err != nil {
		return err
	}

	return nil
}

// Deinit cleans up internal WM state before exiting.
func (wm *WM) Deinit() {
	if wm.xc != nil {
		wm.xc.Close()
	}
}

func (wm *WM) initScreens() error {
	setup := xproto.Setup(wm.xc)
	if setup == nil || len(setup.Roots) < 1 {
		return errors.New("Could not parse SetupInfo.")
	}
	if err := xinerama.Init(wm.xc); err != nil {
		return err
	}
	if r, err := xinerama.QueryScreens(wm.xc).Reply(); err != nil {
		return err
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

	coninfo := xproto.Setup(wm.xc)
	if coninfo == nil {
		return errors.New("Could not parse X connection info")
	}
	if len(coninfo.Roots) != 1 {
		return errors.New("Bad number of roots. Did Xinerama initialize correctly?")
	}
	wm.xroot = coninfo.Roots[0]
	return nil
}

func (wm *WM) initClients() error {
	tree, err := xproto.QueryTree(wm.xc, wm.xroot.Root).Reply()
	if err != nil {
		return err
	}
	if tree == nil {
		return nil
	}
	for _, win := range tree.Children {
		if wm.GetClient(win) != nil {
			panic("window already managed by a client - what happened?")
		}
		c := NewClient(wm.xc, win)
		err := c.Init()
		if err != nil {
			return err
		}
		wm.AddClient(c)
	}
	return nil
}

func (wm *WM) initWM() error {
	err := xproto.ChangeWindowAttributesChecked(
		wm.xc,
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
			return errorAnotherWM
		}
	}
	return err
}

// AddClient adds the client to WM's internal client list.
func (wm *WM) AddClient(c *Client) {
	w := c.window // private!
	wm.clients[w] = c
}

// GetClient returns the Client from associated Window ID, or nil.
func (wm *WM) GetClient(w xproto.Window) *Client {
	c, _ := wm.clients[w]
	return c
}
