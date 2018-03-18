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

	grabs  []*Grab
	keymap [256][]xproto.Keysym

	activeClient *Client

	workspaces []*Workspace
	activeWs   int
}

// NewWM allocates internal WM data structures and creates a WM
// instance. No X11 calls are made until WM.Init() is called.
func NewWM() *WM {
	return &WM{
		workspaces: []*Workspace{
			&Workspace{
				Layout: &ColumnLayout{},
			},
		},
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
	if err = wm.initKeys(); err != nil {
		return err
	}
	if err = wm.initWorkspaces(); err != nil {
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

func (wm *WM) initWorkspaces() error {
	tree, err := xproto.QueryTree(wm.xc, wm.xroot.Root).Reply()
	if err != nil {
		return err
	}
	if tree != nil {
		w := wm.GetActiveWorkspace()
		for _, win := range tree.Children {
			if c, err := NewClient(wm.xc, win); err != nil {
				return err
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
			return err
		}
	}
	return nil
}

// GetActiveWorkspace returns the Workspace containing the current
// active Client, or nil if no Client is active.
func (wm *WM) GetActiveWorkspace() *Workspace {
	w := wm.workspaces[wm.activeWs]
	if wm.activeClient != nil && !w.HasWindow(wm.activeClient.Window) {
		panic("marked active but don't have the active client")
	}
	return w
}
