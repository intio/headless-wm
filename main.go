package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

var xc *xgb.Conn
var xroot xproto.ScreenInfo
var errorQuit error = errors.New("Quit")
var keymap [256][]xproto.Keysym
var attachedScreens []xinerama.ScreenInfo

// ICCCM related atoms
var (
	atomWMProtocols    xproto.Atom
	atomWMDeleteWindow xproto.Atom
	atomWMTakeFocus    xproto.Atom
)

// Grab represents a key grab and its callback
type Grab struct {
	sym       xproto.Keysym
	modifiers uint16
	codes     []xproto.Keycode
	callback  func() error
}

var grabs = []*Grab{
	{
		sym:       XK_q,
		modifiers: xproto.ModMaskControl | xproto.ModMaskShift | xproto.ModMask1,
		callback:  func() error { return errorQuit },
	},
	{
		sym:       XK_Return,
		modifiers: xproto.ModMask1,
		callback: func() error {
			go func() {
				cmd := exec.Command("x-terminal-emulator")
				if err := cmd.Start(); err == nil {
					cmd.Wait()
				}
			}()
			return nil
		},
	},
	{
		sym:       XK_q,
		modifiers: xproto.ModMask1,
		callback:  quitWindowGracefully,
	},
	{
		sym:       XK_q,
		modifiers: xproto.ModMask1 | xproto.ModMaskShift,
		callback:  quitWindowForcefully,
	},

	{
		sym:       XK_h,
		modifiers: xproto.ModMask1,
		callback: func() error {
			if activeWindow == nil {
				return nil
			}
			for _, wp := range workspaces {
				if err := wp.Left(&ManagedWindow{*activeWindow}); err == nil {
					wp.TileWindows()
				}
			}
			return nil
		},
	},
	{
		sym:       XK_j,
		modifiers: xproto.ModMask1,
		callback: func() error {
			if activeWindow == nil {
				return nil
			}
			for _, wp := range workspaces {
				if err := wp.Down(&ManagedWindow{*activeWindow}); err == nil {
					wp.TileWindows()
				}
			}
			return nil
		},
	},
	{
		sym:       XK_k,
		modifiers: xproto.ModMask1,
		callback: func() error {
			if activeWindow == nil {
				return nil
			}
			for _, wp := range workspaces {
				if err := wp.Up(&ManagedWindow{*activeWindow}); err == nil {
					wp.TileWindows()
				}
			}
			return nil
		},
	},
	{
		sym:       XK_l,
		modifiers: xproto.ModMask1,
		callback: func() error {
			if activeWindow == nil {
				return nil
			}
			for _, wp := range workspaces {
				if err := wp.Right(&ManagedWindow{*activeWindow}); err == nil {
					wp.TileWindows()
				}
			}
			return nil
		},
	},

	{
		sym:       XK_d,
		modifiers: xproto.ModMask1,
		callback:  cleanupColumns,
	},
	{
		sym:       XK_n,
		modifiers: xproto.ModMask1,
		callback:  addColumn,
	},
	{
		sym:       XK_m,
		modifiers: xproto.ModMask1,
		callback:  maximizeActiveWindow,
	},
}

func quitWindowGracefully() error {
	if activeWindow == nil {
		log.Println("Tried to close window, but no active window")
		return nil
	}
	prop, err := xproto.GetProperty(xc, false, *activeWindow, atomWMProtocols,
		xproto.GetPropertyTypeAny, 0, 64).Reply()
	if err != nil {
		return err
	}
	if prop == nil {
		// There were no properties, so the window doesn't follow ICCCM.
		// Just destroy it.
		if activeWindow != nil {
			return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
		}
	}
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		switch decodeAtom(v) {
		case atomWMDeleteWindow:
			t := time.Now().Unix()
			return xproto.SendEventChecked(
				xc,
				false,
				*activeWindow,
				xproto.EventMaskNoEvent,
				string(xproto.ClientMessageEvent{
					Format: 32,
					Window: *activeWindow,
					Type:   atomWMProtocols,
					Data: xproto.ClientMessageDataUnionData32New([]uint32{
						uint32(atomWMDeleteWindow),
						uint32(t),
						0,
						0,
						0,
					}),
				}.Bytes())).Check()
		}
	}
	// No WM_DELETE_WINDOW protocol, so destroy.
	if activeWindow != nil {
		return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
	}
	return nil
}

func quitWindowForcefully() error {
	if activeWindow != nil {
		return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
	}
	return nil
}

func cleanupColumns() error {
	for _, w := range workspaces {
		if w.IsActive() {
			newColumns := make([]*Column, 0, len(w.columns))
			for _, c := range w.columns {
				if len(c.Windows) > 0 {
					newColumns = append(newColumns, c)
				}
			}
			// Don't bother using the newColumns if it didn't change
			// anything. Just let newColumns get GCed.
			if len(newColumns) != len(w.columns) {
				w.columns = newColumns
				w.TileWindows()
			}
		}
	}
	return nil
}

func addColumn() error {
	for _, w := range workspaces {
		if w.IsActive() {
			w.columns = append(w.columns, &Column{})
			w.TileWindows()
		}
	}
	return nil
}

func maximizeActiveWindow() error {
	for _, w := range workspaces {
		if !w.IsActive() {
			continue
		}
		if w.maximizedWindow == nil {
			w.maximizedWindow = activeWindow
		} else {
			if err := xproto.ConfigureWindowChecked(
				xc,
				*w.maximizedWindow,
				xproto.ConfigWindowBorderWidth,
				[]uint32{2},
			).Check(); err != nil {
				log.Print(err)
			}
			w.maximizedWindow = nil
		}
		w.TileWindows()
	}
	return nil
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
			attachedScreens = []xinerama.ScreenInfo{
				xinerama.ScreenInfo{
					Width:  setup.Roots[0].WidthInPixels,
					Height: setup.Roots[0].HeightInPixels,
				},
			}
		} else {
			attachedScreens = r.ScreenInfo
		}
	}

	coninfo := xproto.Setup(xc)
	if coninfo == nil {
		log.Fatal("Could not parse X connection info")
	}
	if len(coninfo.Roots) != 1 {
		log.Fatal("Inappropriate number of roots. Did Xinerama initialize correctly?")
	}
	xroot = coninfo.Roots[0]
}

func initWM() {
	err := xproto.ChangeWindowAttributesChecked(
		xc,
		xroot.Root,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease |
				xproto.EventMaskButtonPress |
				xproto.EventMaskButtonRelease |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskSubstructureRedirect,
		}).Check()
	if err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			log.Fatal("Could not become the WM. Is another WM already running?")
		}
		log.Fatal(err)
	}
}

func initAtoms() {
	atomWMProtocols = getAtom("WM_PROTOCOLS")
	atomWMDeleteWindow = getAtom("WM_DELETE_WINDOW")
	atomWMTakeFocus = getAtom("WM_TAKE_FOCUS")
}

func initKeys() {
	const (
		loKey = 8
		hiKey = 255
	)

	m := xproto.GetKeyboardMapping(xc, loKey, hiKey-loKey+1)
	reply, err := m.Reply()
	if err != nil {
		log.Fatal(err)
	}
	if reply == nil {
		log.Fatal("Could not load keyboard map")
	}

	for i := 0; i < hiKey-loKey+1; i++ {
		keymap[loKey+i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode) : (i+1)*int(reply.KeysymsPerKeycode)]
	}

	for i, syms := range keymap {
		for _, sym := range syms {
			for c := range grabs {
				if grabs[c].sym == sym {
					grabs[c].codes = append(grabs[c].codes, xproto.Keycode(i))
				}
			}
		}
	}
	for _, grabbed := range grabs {
		for _, code := range grabbed.codes {
			if err := xproto.GrabKeyChecked(
				xc,
				false,
				xroot.Root,
				grabbed.modifiers,
				code,
				xproto.GrabModeAsync,
				xproto.GrabModeAsync,
			).Check(); err != nil {
				log.Print(err)
			}

		}
	}
}

func initWorkspaces() {
	tree, err := xproto.QueryTree(xc, xroot.Root).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if tree != nil {
		defaultw := workspaces["default"]
		for _, c := range tree.Children {
			if err := defaultw.Add(c); err != nil {
				log.Println(err)
			}
		}
		if len(attachedScreens) == 0 {
			panic("no attached screens!?")
		}
	}
	for _, workspace := range workspaces {
		workspace.Screen = &attachedScreens[0]
		if err := workspace.TileWindows(); err != nil {
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

	// Main X Event loop
eventloop:
	for {
		xev, err := xc.WaitForEvent()
		if err != nil {
			log.Println(err)
			continue
		}
		switch e := xev.(type) {
		case xproto.KeyPressEvent:
			err := HandleKeyPressEvent(e)
			switch err {
			case nil:
				continue eventloop
			case errorQuit:
				os.Exit(0)
			default:
				log.Fatal(err)
			}
		case xproto.DestroyNotifyEvent:
			for _, w := range workspaces {
				if err := w.RemoveWindow(e.Window); err == nil {
					w.TileWindows()
				}
			}
			if activeWindow != nil && e.Window == *activeWindow {
				activeWindow = nil
				// Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.
				xproto.SetInputFocus(xc, xproto.InputFocusPointerRoot, xroot.Root, xproto.TimeCurrentTime)
			}
		case xproto.ConfigureRequestEvent:
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
			xproto.SendEventChecked(xc, false, e.Window, xproto.EventMaskStructureNotify, string(ev.Bytes()))
		case xproto.MapRequestEvent:
			if winattrib, err := xproto.GetWindowAttributes(xc, e.Window).Reply(); err != nil || !winattrib.OverrideRedirect {
				w := workspaces["default"]
				xproto.MapWindowChecked(xc, e.Window)
				w.Add(e.Window)
				w.TileWindows()
			}
		case xproto.EnterNotifyEvent:
			activeWindow = &e.Event

			prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
				xproto.GetPropertyTypeAny, 0, 64).Reply()
			focused := false
			if err == nil {
			TakeFocusPropLoop:
				for v := prop.Value; len(v) >= 4; v = v[4:] {
					switch decodeAtom(v) {
					case atomWMTakeFocus:
						xproto.SendEventChecked(
							xc,
							false,
							e.Event,
							xproto.EventMaskNoEvent,
							string(xproto.ClientMessageEvent{
								Format: 32,
								Window: *activeWindow,
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
			}
			if !focused {
				// Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.
				xproto.SetInputFocus(xc, xproto.InputFocusPointerRoot, e.Event, e.Time)
			}
		default:
			log.Println(xev)
		}
	}
}

func HandleKeyPressEvent(key xproto.KeyPressEvent) error {
	for _, grab := range grabs {
		if grab.modifiers == key.State &&
			grab.sym == keymap[key.Detail][0] &&
			grab.callback != nil {
			return grab.callback()
		}
	}
	return nil
}

func getAtom(name string) xproto.Atom {
	rply, err := xproto.InternAtom(xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if rply == nil {
		return 0
	}
	return rply.Atom
}

// decodeAtom decodes an xproto.Atom from a property value (expressed
// as bytes). Note that v has to be at least 4 bytes long.
func decodeAtom(v []byte) xproto.Atom {
	return xproto.Atom(uint32(v[0]) | uint32(v[1])<<8 | uint32(v[2])<<16 | uint32(v[3])<<24)
}
