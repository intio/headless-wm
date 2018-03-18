package main

import (
	"time"

	"github.com/BurntSushi/xgb/xproto"
)

// Client is an X11 client managed by us.
type Client struct {
	// Window represents X11's internal window ID.
	Window xproto.Window
	// X and Y are the coordinates of the topleft corner of the window.
	X, Y uint32
	// W and H are width and height. Zero means don't change.
	W, H uint32
	// Width of the window border (1 by default).
	BorderWidth uint32
	// one of: StackModeAbove (default), StackModeBelow,
	// StackModeTopIf, StackModeBottomIf, StackModeOpposite.
	StackMode uint32
}

var activeClient *Client

// NewClient initializes the Client struct from Window ID
func NewClient(w xproto.Window) (*Client, error) {
	c := &Client{
		Window:      w,
		X:           0,
		Y:           0,
		W:           0,
		H:           0,
		BorderWidth: 1,
		StackMode:   xproto.StackModeAbove,
	}

	// Ensure that we can manage this window.
	if err := c.Configure(); err != nil {
		return nil, err
	}

	// Get notifications when this window is deleted.
	if err := xproto.ChangeWindowAttributesChecked(
		xc,
		c.Window,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskStructureNotify |
				xproto.EventMaskEnterWindow,
		},
	).Check(); err != nil {
		return nil, err
	}
	return c, nil
}

// Configure sends a configuration request to inflict Client's
// internal state on the real world.
func (c *Client) Configure() error {
	valueMask := uint16(xproto.ConfigWindowX |
		xproto.ConfigWindowY |
		xproto.ConfigWindowBorderWidth |
		xproto.ConfigWindowStackMode)
	valueList := []uint32{}
	valueList = append(valueList, c.X, c.Y)
	if c.W > 0 {
		valueMask |= xproto.ConfigWindowWidth
		valueList = append(valueList, c.W)
	}
	if c.H > 0 {
		valueMask |= xproto.ConfigWindowHeight
		valueList = append(valueList, c.H)
	}
	valueList = append(valueList, c.BorderWidth, c.StackMode)
	return xproto.ConfigureWindowChecked(
		xc,
		c.Window,
		valueMask,
		valueList,
	).Check()
}

// WarpPointer puts the mouse pointer inside of this client's window.
func (c *Client) WarpPointer() error {
	return xproto.WarpPointerChecked(
		xc,       // conn
		0,        // src
		c.Window, // dst
		0,        // src x
		0,        // src x
		0,        // src w
		0,        // src h
		10,       // dst x
		10,       // dst y
	).Check()
}

func (c *Client) CloseGracefully() error {
	prop, err := xproto.GetProperty(
		xc,
		false,                     // delete
		c.Window,                  // window
		atomWMProtocols,           // property
		xproto.GetPropertyTypeAny, // atom
		0,  // offset
		64, // length
	).Reply()
	if err != nil {
		return err
	}
	if prop == nil {
		// There were no properties, so the window doesn't follow ICCCM.
		// Just destroy it.
		return c.CloseForcefully()
	}
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		if decodeAtom(v) == atomWMDeleteWindow {
			// ICCCM 4.2.8 ClientMessage
			t := time.Now().Unix()
			ev := xproto.ClientMessageEvent{
				Format: 32,
				Window: activeClient.Window,
				Type:   atomWMProtocols,
				Data: xproto.ClientMessageDataUnionData32New([]uint32{
					uint32(atomWMDeleteWindow),
					uint32(t),
					0,
					0,
					0,
				}),
			}
			return xproto.SendEventChecked(
				xc,
				false,                   // propagate
				activeClient.Window,     // destination
				xproto.EventMaskNoEvent, // eventmask
				string(ev.Bytes()),      // event
			).Check()
		}
	}
	// No WM_DELETE_WINDOW protocol, so destroy.
	closeClientForcefully()
	if activeClient != nil {
		return xproto.DestroyWindowChecked(xc, activeClient.Window).Check()
	}
	return nil

}

func (c *Client) CloseForcefully() error {
	return xproto.DestroyWindowChecked(xc, c.Window).Check()
}
