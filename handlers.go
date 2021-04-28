package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/xgb/xproto"
)

func (wm *WM) handleEvent() (err error) {
	xev, err := wm.xc.WaitForEvent()
	if err != nil {
		return err
	}
	data := (map[string]interface{}{
		"type":  fmt.Sprintf("%T", xev),
		"event": xev,
	})
	switch e := xev.(type) {
	case xproto.KeyPressEvent:
		err = wm.handleKeyPressEvent(e)
	case xproto.KeyReleaseEvent:
		err = wm.handleKeyReleaseEvent(e)
	case xproto.ButtonPressEvent:
		err = wm.handleButtonPressEvent(e)
	case xproto.ButtonReleaseEvent:
		err = wm.handleButtonReleaseEvent(e)
	case xproto.DestroyNotifyEvent:
		err = wm.handleDestroyNotifyEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	case xproto.ConfigureRequestEvent:
		err = wm.handleConfigureRequestEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	case xproto.MapRequestEvent:
		err = wm.handleMapRequestEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	case xproto.EnterNotifyEvent:
		err = wm.handleEnterNotifyEvent(e)
		data["client"] = wm.GetClient(e.Event)
		data["clientID"] = e.Event
	case xproto.MapNotifyEvent:
		err = wm.handleMapNotifyEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	case xproto.UnmapNotifyEvent:
		err = wm.handleUnmapNotifyEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	case xproto.ConfigureNotifyEvent:
		err = wm.handleConfigureNotifyEvent(e)
		data["client"] = wm.GetClient(e.Window)
		data["clientID"] = e.Window
	}
	if wm.api != nil {
		go func() {
			for c := range wm.api.clients {
				c.ch <- data
			}
		}()
	}
	return err
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
	wm.handleNewWindow(e.Window)
	c := wm.GetClient(e.Window)
	c.X = e.X
	c.Y = e.Y
	c.W = e.Width
	c.H = e.Height
	// TODO: apply position/size policy
	// c.MakeFullscreen(wm.attachedScreens[0])
	return c.Configure()
}

func (wm *WM) handleMapRequestEvent(e xproto.MapRequestEvent) (err error) {
	winattrib, err := xproto.GetWindowAttributes(wm.xc, e.Window).Reply()
	if err != nil || !winattrib.OverrideRedirect {
		xproto.MapWindowChecked(wm.xc, e.Window)
		if err = wm.handleNewWindow(e.Window); err == nil {
			return
		}
		c := wm.GetClient(e.Window)
		if wm.activeClient == nil {
			wm.activeClient = c
		}
	}
	return err
}

func (wm *WM) handleEnterNotifyEvent(e xproto.EnterNotifyEvent) error {
	c := wm.GetClient(e.Event)
	wm.activeClient = c
	if wm.activeClient == nil {
		panic("no client is managing this window - what happened?")
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
		c.Focus()
	}
	return nil
}

func (wm *WM) handleMapNotifyEvent(e xproto.MapNotifyEvent) error {
	// TODO: focus stealing prevention?
	c := wm.GetClient(e.Window)
	if c == nil {
		log.Printf("mapped a window that was not being managed: %v", e)
	}
	wm.activeClient = c
	wm.activeClient.Focus()
	return nil
}

func (wm *WM) handleUnmapNotifyEvent(e xproto.UnmapNotifyEvent) error {
	c := wm.GetClient(e.Window)
	if c == nil {
		log.Printf("unmapped a window that was not being managed: %v", e)
	}
	if wm.activeClient == c {
		// TODO: look for the active window?
		wm.activeClient = nil
	}
	wm.ForgetClient(c)
	return nil
}

func (wm *WM) handleConfigureNotifyEvent(e xproto.ConfigureNotifyEvent) error {
	if err := wm.updateScreens(); err != nil {
		return err
	}
	return nil
}
