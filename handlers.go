package main

import (
	"log"

	"github.com/BurntSushi/xgb/xproto"
)

func (wm *WM) handleEvent() error {
	xev, err := wm.xc.WaitForEvent()
	if err != nil {
		return err
	}
	switch e := xev.(type) {
	case xproto.KeyPressEvent:
		return wm.handleKeyPressEvent(e)
	case xproto.KeyReleaseEvent:
		return wm.handleKeyReleaseEvent(e)
	case xproto.ButtonPressEvent:
		return wm.handleButtonPressEvent(e)
	case xproto.ButtonReleaseEvent:
		return wm.handleButtonReleaseEvent(e)
	case xproto.DestroyNotifyEvent:
		return wm.handleDestroyNotifyEvent(e)
	case xproto.ConfigureRequestEvent:
		return wm.handleConfigureRequestEvent(e)
	case xproto.MapRequestEvent:
		return wm.handleMapRequestEvent(e)
	case xproto.EnterNotifyEvent:
		return wm.handleEnterNotifyEvent(e)
	case xproto.MapNotifyEvent:
		return wm.handleMapNotifyEvent(e)
	case xproto.UnmapNotifyEvent:
		return wm.handleUnmapNotifyEvent(e)
	case xproto.ConfigureNotifyEvent:
		return wm.handleConfigureNotifyEvent(e)
	default:
		log.Printf("Unhandled event: %#v", xev)
	}
	return nil
}

func (wm *WM) handleKeyPressEvent(key xproto.KeyPressEvent) error {
	return nil
}

func (wm *WM) handleKeyReleaseEvent(key xproto.KeyReleaseEvent) error {
	return nil
}

func (wm *WM) handleButtonPressEvent(btn xproto.ButtonPressEvent) error {
	return nil
}

func (wm *WM) handleButtonReleaseEvent(btn xproto.ButtonReleaseEvent) error {
	return nil
}

func (wm *WM) handleDestroyNotifyEvent(e xproto.DestroyNotifyEvent) error {
	c := wm.GetClient(e.Window)
	if wm.activeClient != nil && wm.activeClient == c {
		wm.activeClient = nil
		// Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.
		xproto.SetInputFocus(
			wm.xc,                        // conn
			xproto.InputFocusPointerRoot, // revert to
			wm.xroot.Root,                // focus
			xproto.TimeCurrentTime,       // time
		)
	}
	return nil
}

func (wm *WM) handleConfigureRequestEvent(e xproto.ConfigureRequestEvent) error {
	ev := xproto.ConfigureNotifyEvent{
		Event:            e.Window,
		Window:           e.Window,
		AboveSibling:     0,
		X:                e.X,
		Y:                e.Y,
		Width:            e.Width,
		Height:           e.Height,
		OverrideRedirect: false,
	}
	xproto.SendEventChecked(
		wm.xc,                           // conn
		false,                           // propagate
		e.Window,                        // target
		xproto.EventMaskStructureNotify, // mask
		string(ev.Bytes()),              // event
	)
	return nil
}

func (wm *WM) handleMapRequestEvent(e xproto.MapRequestEvent) (err error) {
	winattrib, err := xproto.GetWindowAttributes(wm.xc, e.Window).Reply()
	if err != nil || !winattrib.OverrideRedirect {
		xproto.MapWindowChecked(wm.xc, e.Window)
		if c := wm.GetClient(e.Window); c != nil {
			log.Printf("MapRequest already managed: %v", e.Window)
			return nil
		}
		c := NewClient(wm.xc, e.Window)
		err := c.Init()
		if err == nil {
			wm.AddClient(c)
		} else {
			return err
		}
		if wm.activeClient == nil {
			wm.activeClient = c
		}
	}
	return err
}

func (wm *WM) handleEnterNotifyEvent(e xproto.EnterNotifyEvent) error {
	wm.activeClient = wm.GetClient(e.Event)
	if wm.activeClient == nil {
		panic("no workspace is managing this window - what happened?")
	}
	prop, err := xproto.GetProperty(wm.xc, false, e.Event, atomWMProtocols,
		xproto.GetPropertyTypeAny, 0, 64).Reply()
	if err != nil {
		return err
	}
	focused := false
TakeFocusPropLoop:
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		if decodeAtom(v) == atomWMTakeFocus {
			xproto.SendEventChecked(
				wm.xc,
				false,
				e.Event,
				xproto.EventMaskNoEvent,
				string(xproto.ClientMessageEvent{
					Format: 32,
					Window: wm.activeClient.window, // private!
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
			wm.xc,
			xproto.InputFocusPointerRoot, // revert
			e.Event,                      // focus
			e.Time,                       // timestamp
		)
	}
	return nil
}

func (wm *WM) handleMapNotifyEvent(e xproto.MapNotifyEvent) error {
	// TODO: focus stealing prevention?
	c := wm.GetClient(e.Window)
	if c == nil {
		panic("mapped a window that was not being managed!?")
	}
	wm.activeClient = c
	return nil
}

func (wm *WM) handleUnmapNotifyEvent(e xproto.UnmapNotifyEvent) error {
	c := wm.GetClient(e.Window)
	if c == nil {
		panic("unmapped a window that was not being managed!?")
	}
	if wm.activeClient == c {
		// TODO: look for the active window?
		wm.activeClient = nil
	}
	return nil
}

func (wm *WM) handleConfigureNotifyEvent(e xproto.ConfigureNotifyEvent) error {
	return nil
}
