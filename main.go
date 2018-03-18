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
var keymap [256][]xproto.Keysym

var wm = &WM{
	workspaces: []*Workspace{
		&Workspace{
			Layout: &ColumnLayout{},
		},
	},
	active: 0,
}

func closeClientGracefully() error {
	if activeClient == nil {
		log.Println("Tried to close client, but no active client")
		return nil
	}
	return activeClient.CloseGracefully()
}

func closeClientForcefully() error {
	if activeClient == nil {
		log.Println("Tried to close client, but no active client")
		return nil
	}
	return activeClient.CloseForcefully()
}

func initScreens() {
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

func initWM() {
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

func initWorkspaces() {
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
	var err error
	xc, err = xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	defer xc.Close()

	initScreens()
	initAtoms()
	initWM()
	initKeys()
	initWorkspaces()

	for {
		err = handleEvent()
		switch err {
		case errorQuit:
			os.Exit(0)
		case nil:
		default:
			log.Print(err)
		}
	}
}

func handleEvent() error {
	xev, err := xc.WaitForEvent()
	if err != nil {
		return err
	}
	switch e := xev.(type) {
	case xproto.KeyPressEvent:
		return handleKeyPressEvent(e)
	case xproto.KeyReleaseEvent: // xxx
		return nil
	case xproto.DestroyNotifyEvent:
		return handleDestroyNotifyEvent(e)
	case xproto.ConfigureRequestEvent:
		return handleConfigureRequestEvent(e)
	case xproto.MapRequestEvent:
		return handleMapRequestEvent(e)
	case xproto.EnterNotifyEvent:
		return handleEnterNotifyEvent(e)
	default:
		// log.Printf("Unhandled event: %#v", xev)
	}
	return nil
}

func handleKeyPressEvent(key xproto.KeyPressEvent) error {
	for _, grab := range grabs {
		if grab.modifiers == key.State &&
			grab.sym == keymap[key.Detail][0] &&
			grab.callback != nil {
			return grab.callback()
		}
	}
	return nil
}

func handleDestroyNotifyEvent(e xproto.DestroyNotifyEvent) error {
	for _, w := range wm.workspaces {
		if w.HasWindow(e.Window) {
			w.RemoveWindow(e.Window)
			w.Arrange()
		}
	}
	if activeClient != nil && e.Window == activeClient.Window {
		activeClient = nil
		// Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.
		xproto.SetInputFocus(
			xc,
			xproto.InputFocusPointerRoot, // revert to
			wm.xroot.Root,                // focus
			xproto.TimeCurrentTime,       // time
		)
	}
	return nil
}

func handleConfigureRequestEvent(e xproto.ConfigureRequestEvent) error {
	ev := xproto.ConfigureNotifyEvent{
		Event:            e.Window,
		Window:           e.Window,
		AboveSibling:     0,
		X:                e.X,
		Y:                e.Y,
		Width:            e.Width,
		Height:           e.Height,
		BorderWidth:      0,
		OverrideRedirect: false,
	}
	xproto.SendEventChecked(
		xc,
		false,                           // propagate
		e.Window,                        // target
		xproto.EventMaskStructureNotify, // mask
		string(ev.Bytes()),              // event
	)
	return nil
}

func handleMapRequestEvent(e xproto.MapRequestEvent) error {
	var err error
	winattrib, err := xproto.GetWindowAttributes(xc, e.Window).Reply()
	if err != nil || !winattrib.OverrideRedirect {
		w := wm.GetActiveWorkspace()
		xproto.MapWindowChecked(xc, e.Window)
		c, err := NewClient(e.Window)
		if err == nil {
			w.AddClient(c)
			w.Arrange()
		} else {
			return err
		}
		if activeClient == nil {
			activeClient = c
		}
	}
	return err
}

func handleEnterNotifyEvent(e xproto.EnterNotifyEvent) error {
	for _, ws := range wm.workspaces {
		if c := ws.GetClient(e.Event); c != nil {
			activeClient = c
		}
	}
	prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
		xproto.GetPropertyTypeAny, 0, 64).Reply()
	if err != nil {
		return err
	}
	focused := false
TakeFocusPropLoop:
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		if decodeAtom(v) == atomWMTakeFocus {
			xproto.SendEventChecked(
				xc,
				false,
				e.Event,
				xproto.EventMaskNoEvent,
				string(xproto.ClientMessageEvent{
					Format: 32,
					Window: activeClient.Window,
					Type:   atomWMProtocols,
					Data: xproto.ClientMessageDataUnionData32New([]uint32{
						uint32(atomWMTakeFocus),
						uint32(e.Time),
						0,
						0,
						0,
					}),
				}.Bytes())).Check()
			focused = true
			break TakeFocusPropLoop
		}
	}
	if !focused {
		// Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.
		xproto.SetInputFocus(
			xc,
			xproto.InputFocusPointerRoot, // revert
			e.Event, // focus
			e.Time,  // timestamp
		)
	}
	return nil
}
