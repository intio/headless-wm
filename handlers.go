package main

import (
	"github.com/BurntSushi/xgb/xproto"
)

func (wm *WM) handleEvent() error {
	xev, err := xc.WaitForEvent()
	if err != nil {
		return err
	}
	switch e := xev.(type) {
	case xproto.KeyPressEvent:
		return wm.handleKeyPressEvent(e)
	case xproto.KeyReleaseEvent: // xxx
		return nil
	case xproto.DestroyNotifyEvent:
		return wm.handleDestroyNotifyEvent(e)
	case xproto.ConfigureRequestEvent:
		return wm.handleConfigureRequestEvent(e)
	case xproto.MapRequestEvent:
		return wm.handleMapRequestEvent(e)
	case xproto.EnterNotifyEvent:
		return wm.handleEnterNotifyEvent(e)
	default:
		// log.Printf("Unhandled event: %#v", xev)
	}
	return nil
}

func (wm *WM) handleKeyPressEvent(key xproto.KeyPressEvent) error {
	for _, grab := range wm.grabs {
		if grab.modifiers == key.State &&
			grab.sym == wm.keymap[key.Detail][0] &&
			grab.callback != nil {
			return grab.callback()
		}
	}
	return nil
}

func (wm *WM) handleDestroyNotifyEvent(e xproto.DestroyNotifyEvent) error {
	for _, w := range wm.workspaces {
		if w.HasWindow(e.Window) {
			w.RemoveWindow(e.Window)
			w.Arrange()
		}
	}
	if wm.activeClient != nil && e.Window == wm.activeClient.Window {
		wm.activeClient = nil
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

func (wm *WM) handleConfigureRequestEvent(e xproto.ConfigureRequestEvent) error {
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

func (wm *WM) handleMapRequestEvent(e xproto.MapRequestEvent) error {
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
		if wm.activeClient == nil {
			wm.activeClient = c
		}
	}
	return err
}

func (wm *WM) handleEnterNotifyEvent(e xproto.EnterNotifyEvent) error {
	for _, ws := range wm.workspaces {
		if c := ws.GetClient(e.Event); c != nil {
			wm.activeClient = c
		}
	}
	if wm.activeClient == nil {
		panic("no workspace is managing this window - what happened?")
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
					Window: wm.activeClient.Window,
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
